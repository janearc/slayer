package live

import (
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/job"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v3"
)

// s writes the moment, camera and exposure the keys chose into the job,
// lets the keys' changes go, and says the editor must reload.
func TestSaveWritesWhatTheKeysChose(t *testing.T) {
	watching := idle(t, stillJob)
	for _, bytes := range []string{"]", "c", "E", "E", "E"} {
		watching.key(pressed(t, bytes))
	}
	if got := watching.key(pressed(t, "s")); got != Restage {
		t.Errorf("s gave %v; the exposure written restages", got)
	}
	raw, err := os.ReadFile(watching.given.Path)
	if err != nil {
		t.Fatal(err)
	}
	var read struct {
		At     struct{ Value float64 }
		Camera struct{ Name string }
		Film   struct{ Exposure float64 }
	}
	if err := yaml.Unmarshal(raw, &read); err != nil {
		t.Fatal(err)
	}
	if read.At.Value != 1 || read.Camera.Name != "near" ||
		read.Film.Exposure != 4 {
		t.Errorf("s wrote %s", raw)
	}
	if watching.view.thirds != none || watching.view.camera != "" ||
		!strings.Contains(watching.words, "reload") {
		t.Errorf("after s the view is %+v, %q", watching.view,
			watching.words)
	}
}

// anti-test of TestSaveWritesWhatTheKeysChose
// s with nothing chosen writes nothing; an exposure with no
// scene exposure to scale is said and not written.
func TestSaveWritesNothingUnasked(t *testing.T) {
	watching := idle(t, stillJob)
	before, _ := os.ReadFile(watching.given.Path)
	if got := watching.key(pressed(t, "s")); got != Unchanged {
		t.Errorf("s with nothing chosen gave %v", got)
	}
	watching.facts.exposure = none
	watching.key(pressed(t, "E"))
	watching.key(pressed(t, "s"))
	after, _ := os.ReadFile(watching.given.Path)
	if string(before) != string(after) {
		t.Errorf("s wrote %s", after)
	}
	if !strings.Contains(strings.Join(watching.foot.lines(), " "),
		"film.exposure") {
		t.Errorf("the foot says %q", watching.foot.lines())
	}
}

// the status row says the quality, rung, frame, camera, exposure and
// play; the film time is the frame's from the film's start.
func TestStatusSaysTheView(t *testing.T) {
	watching := idle(t, filmJob+"camera: {name: far}\n")
	if got := watching.status(nil); !strings.Contains(got,
		"preview") && !strings.Contains(got, "exact") {
		t.Errorf("the status row says %q", got)
	}
	watching.shown = &held{rung: rung{size: Size{8, 6, 1},
		words: "1/8 size, 1 ray"}, index: filmRate}
	watching.view.index = filmRate
	watching.key(pressed(t, "E"))
	watching.key(pressed(t, " "))
	got := watching.status(nil)
	for _, words := range []string{"1/8 size, 1 ray", "frame 5 of 8",
		"camera far", "exposure +1/3 stop", "playing", "? help"} {
		if !strings.Contains(got, words) {
			t.Errorf("the status row %q does not say %q", got, words)
		}
	}
	if watching.filmTime() != time.Second {
		t.Errorf("frame %d at %d a second is %v", filmRate, filmRate,
			watching.filmTime())
	}
	watching.shown.rung = rung{words: "full size"}
	picture := image.NewRGBA(image.Rect(0, 0, 4, 3))
	if got := watching.rungWords(picture); got != "full size 4x3" {
		t.Errorf("a full frame says %q", got)
	}
}

// play draws at the finest rung drawn within a frame's time, and the
// first rung before any is known; a full frame draws at the job's size.
func TestPlansDrawWhatTheyAsk(t *testing.T) {
	watching := idle(t, filmJob)
	ladder := ladderFor(watching.facts, testScreen())
	if got := watching.fastest(); got != (rung{}) {
		t.Errorf("not playing, the fastest rung is %+v", got)
	}
	watching.view.playing = true
	watching.timings[ladder[0]] = time.Millisecond
	watching.timings[ladder[1]] = time.Millisecond
	watching.timings[ladder[2]] = time.Second
	if got := watching.fastest(); got != ladder[1] {
		t.Errorf("the fastest rung is %+v, want %+v", got, ladder[1])
	}
	made := watching.facts
	if got := (plan{mode: full}).rungs(made); len(got) != one ||
		got[0].size != (Size{}) {
		t.Errorf("a full frame draws %+v", got)
	}
	playing := plan{playing: true, screen: testScreen()}
	if got := playing.rungs(made); len(got) != one || got[0] != ladder[0] {
		t.Errorf("play with nothing timed draws %+v", got)
	}
	playing.fastest = ladder[1]
	if got := playing.rungs(made); got[0] != ladder[1] {
		t.Errorf("play draws %+v", got)
	}
}

// a frame the stage refuses is told to the loop; one refused because the
// plan was cancelled is not.
func TestWorkerTellsARefusal(t *testing.T) {
	control := gomock.NewController(t)
	stage := NewMockStage(control)
	plain := stageOf(control, one, nil)
	stage.EXPECT().Times().DoAndReturn(plain.Times).AnyTimes()
	stage.EXPECT().Tone().DoAndReturn(plain.Tone).AnyTimes()
	stage.EXPECT().Shape().DoAndReturn(plain.Shape).AnyTimes()
	stage.EXPECT().Exposure().DoAndReturn(plain.Exposure).AnyTimes()
	stage.EXPECT().Cameras().DoAndReturn(plain.Cameras).AnyTimes()
	stage.EXPECT().Frame(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("no frame 9")).Times(one)
	out := make(chan message, 8)
	working := worker{now: time.Now, out: out}
	working.run(context.Background(), plan{stage: stage,
		screen: testScreen()})
	if said := <-out; said.err == nil || said.done {
		t.Errorf("a refusal was told as %+v", said)
	}
	if said := <-out; !said.done || said.stage != stage {
		t.Errorf("the plan ended with %+v", said)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	working.refused(ctx, errors.New("cancelled"))
	select {
	case said := <-out:
		t.Errorf("a cancelled plan told %+v", said)
	default:
	}
	working.run(ctx, plan{stage: stage, screen: testScreen(),
		mode: full})
	if said := <-out; !said.done {
		t.Errorf("a cancelled plan told %+v", said)
	}
}

// a stage that is restaged is closed first, and one opened over a close
// that failed is closed again and the failure said.
func TestWorkerClosesTheStageItReplaces(t *testing.T) {
	control := gomock.NewController(t)
	old := NewMockStage(control)
	old.EXPECT().Close().Return(errors.New("stuck"))
	opened := stageOf(control, one, nil)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(opened, nil)
	out := make(chan message, 8)
	working := worker{stager: stager, now: time.Now, out: out}
	_, err := working.staged(context.Background(), plan{stage: old,
		restage: true})
	if err == nil || !strings.Contains(err.Error(), "stuck") {
		t.Errorf("a stage that would not close gave %v", err)
	}
}

// the status folder that cannot be written is said once, and nothing is
// left beside where it would be.
func TestPublishSaysOnceWhenItCannot(t *testing.T) {
	watching := idle(t, stillJob)
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, nil, fileMode); err != nil {
		t.Fatal(err)
	}
	watching.given.Folder = blocked
	watching.publish(nil, nil, testPixel, false)
	watching.publish(nil, nil, testPixel, false)
	said := strings.Count(strings.Join(watching.foot.lines(), "\n"),
		"not written")
	if said != one {
		t.Errorf("the foot said it %d times: %q", said,
			watching.foot.lines())
	}
	if err := replaced(filepath.Join(blocked, "x"), nil,
		fileMode); err == nil {
		t.Error("a file was written under a file")
	}
}

// a job whose development cannot be made keeps the last and says why;
// a camera the scene has is said on the status row.
func TestDevelopmentRefusedIsSaid(t *testing.T) {
	watching := idle(t, stillJob)
	version := watching.version
	broken, err := job.Read(jobFile(t, stillJob+
		"titles: {theme: /nowhere.yaml, elements: {a: {kind: text, "+
		"words: hi, ink: ink, across: 0.5, down: 0.5}}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	watching.wanted = broken
	watching.developer()
	if watching.version != version || !strings.Contains(
		strings.Join(watching.foot.lines(), " "), "nowhere") {
		t.Errorf("a theme that is not there gave version %d, %q",
			watching.version, watching.foot.lines())
	}
	watching.view.camera = "far"
	if watching.cameraWords() != "far" {
		t.Errorf("the camera chosen is %q", watching.cameraWords())
	}
}

// a poll of a file that is gone keeps the picture and says so, and a
// stage that opens with fewer frames holds the frame shown to its end.
func TestPollOfAFileThatIsGone(t *testing.T) {
	watching := idle(t, filmJob)
	if err := os.Remove(watching.given.Path); err != nil {
		t.Fatal(err)
	}
	watching.poll()
	if watching.foot.trouble == "" {
		t.Error("a file that is gone was not said")
	}
	watching.view.index = filmFrames - one
	watching.opened(facts{times: make([]time.Duration, pair)})
	if watching.view.index != one || watching.pending != Redraw {
		t.Errorf("a shorter film shows frame %d, owing %v",
			watching.view.index, watching.pending)
	}
}
