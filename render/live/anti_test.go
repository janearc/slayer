package live

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"go.uber.org/mock/gomock"
)

// framed is a mock stage with the facts of stageOf, drawing each frame
// by draw.
func framed(control *gomock.Controller, times int,
	draw func(ctx context.Context, index int, size Size) (Drawn, error),
) *MockStage {
	plain := stageOf(control, times, nil)
	stage := NewMockStage(control)
	stage.EXPECT().Times().DoAndReturn(plain.Times).AnyTimes()
	stage.EXPECT().Tone().DoAndReturn(plain.Tone).AnyTimes()
	stage.EXPECT().Shape().DoAndReturn(plain.Shape).AnyTimes()
	stage.EXPECT().Exposure().DoAndReturn(plain.Exposure).AnyTimes()
	stage.EXPECT().Cameras().DoAndReturn(plain.Cameras).AnyTimes()
	stage.EXPECT().Close().Return(nil).AnyTimes()
	stage.EXPECT().Frame(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(draw).AnyTimes()
	return stage
}

// statusOf is the status.json in a folder, or the zero report.
func statusOf(folder string) report {
	var said report
	raw, err := os.ReadFile(filepath.Join(folder, statusFile))
	if err != nil || json.Unmarshal(raw, &said) != nil {
		return report{}
	}
	return said
}

// statusWhen waits for the folder's status to be one that holds, and
// gives it; the test stops if none comes.
func statusWhen(t *testing.T, folder, what string,
	holds func(report) bool) report {
	t.Helper()
	deadline := time.Now().Add(waitLimit)
	for time.Now().Before(deadline) {
		if said := statusOf(folder); holds(said) {
			return said
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the status never said %s: %+v", what, statusOf(folder))
	return report{}
}

// anti-test of TestChangeNamesAndFolds
// folding never takes the cheaper, whichever side it is on, and the
// four changes have four different names.
func TestChangeFoldsNeverCheapens(t *testing.T) {
	if Redraw.With(Redevelop) != Redraw ||
		Restage.With(Unchanged) != Restage ||
		Redevelop.With(Redraw) != Redraw ||
		Unchanged.With(Unchanged) != Unchanged {
		t.Error("a fold took the cheaper change")
	}
	seen := map[string]bool{}
	for _, each := range []Change{Unchanged, Redevelop, Redraw, Restage} {
		seen[each.String()] = true
	}
	if len(seen) != 4 || seen["unknown"] {
		t.Errorf("the four changes are named %v", seen)
	}
}

// anti-test of TestKeysMoveAStillsMoment
// on a film, [ and ] step frames and leave the moment alone; nothing
// restages.
func TestKeysDoNotMoveAFilmsMoment(t *testing.T) {
	watching := idle(t, filmJob)
	if got := watching.key(pressed(t, "]")); got != Redraw ||
		watching.view.said || watching.view.moment != none {
		t.Errorf("] on a film gave %v, moment %v, said %v", got,
			watching.view.moment, watching.view.said)
	}
}

// anti-test of TestCanvasResamplesAndBars
// a picture placed at its own size is copied pixel for pixel, with no
// bars and no mixing of neighbours.
func TestCanvasCopiesAPictureOfItsOwnSize(t *testing.T) {
	four := image.NewRGBA(image.Rect(0, 0, 2, 2))
	want := []color.RGBA{{R: 200, A: 255}, {G: 200, A: 255},
		{B: 200, A: 255}, {R: 90, G: 90, A: 255}}
	for i, each := range want {
		four.SetRGBA(i%2, i/2, each)
	}
	out := canvas(four, placing{shown: image.Rect(0, 0, 2, 2), paneW: 2,
		paneH: 2})
	for i, each := range want {
		if got := out.RGBAAt(i%2, i/2); got != each {
			t.Errorf("pixel %d is %v, want %v", i, got, each)
		}
	}
}

// anti-test of TestScaledChangesOnlyTheCopysColour
// a gain of nought thirds leaves colour as it was, and a stop down
// halves it.
func TestScaledByNothingAndDown(t *testing.T) {
	light := lit(1, 1)
	same := scaled(light, gainOf(none))
	if same.Pix[0].R != 0.5 || same == light {
		t.Errorf("no gain gave %+v, a copy: %v", same.Pix[0], same != light)
	}
	down := scaled(light, gainOf(-thirdsInStop))
	if down.Pix[0].R != 0.25 || down.Pix[0].Cover != 1 {
		t.Errorf("a stop down is %+v", down.Pix[0])
	}
	if light.Pix[0].R != 0.5 {
		t.Errorf("the light drawn was changed to %+v", light.Pix[0])
	}
}

// anti-test of TestNewSizeRefusesBelowNought
// each part is refused alone, the width and the rays too.
func TestNewSizeRefusesEachPartBelowNought(t *testing.T) {
	for _, each := range [][3]int{{-one, 2, 1}, {3, 2, -one},
		{-one, -one, -one}} {
		if got, err := NewSize(each[0], each[1], each[2]); err == nil ||
			got != (Size{}) {
			t.Errorf("%v was taken as %+v", each, got)
		}
	}
}

// anti-test of TestStatusSaysTheView
// a still the keys have not touched says no camera, exposure or play,
// and that nothing is drawn yet.
func TestStatusOfAPlainStillSaysLittle(t *testing.T) {
	got := idle(t, stillJob).status(nil)
	for _, words := range []string{"camera", "exposure", "playing",
		"frame "} {
		if strings.Contains(got, words) {
			t.Errorf("the status row %q says %q", got, words)
		}
	}
	if !strings.Contains(got, "nothing drawn yet") {
		t.Errorf("the status row %q does not say nothing is drawn", got)
	}
}

// anti-test of TestPlansDrawWhatTheyAsk
// a preview not playing draws the whole ladder, and play draws the
// first rung when every rung took longer than a frame.
func TestPlansDrawTheLadderUnlessPlaying(t *testing.T) {
	watching := idle(t, filmJob)
	ladder := ladderFor(watching.facts, testScreen())
	asked := plan{screen: testScreen()}
	if got := asked.rungs(watching.facts); len(got) != len(ladder) ||
		len(ladder) < pair {
		t.Errorf("a preview draws %d of %d rungs", len(got), len(ladder))
	}
	watching.view.playing = true
	for _, each := range ladder {
		watching.timings[each] = 2 * time.Second
	}
	if got := watching.fastest(); got != (rung{}) {
		t.Errorf("with every rung slower than a frame, fastest is %+v",
			got)
	}
}

// anti-test of TestWorkerTellsARefusal
// a frame that is drawn is told as a frame, never as a refusal, and the
// plan ends done with its stage.
func TestWorkerTellsADrawnFrameWithoutARefusal(t *testing.T) {
	control := gomock.NewController(t)
	stage := stageOf(control, one, nil)
	out := make(chan message, 16)
	working := worker{now: time.Now, out: out}
	working.run(context.Background(), plan{stage: stage,
		screen: testScreen()})
	close(out)
	drawn, ended := 0, false
	for said := range out {
		if said.err != nil {
			t.Errorf("a refusal was told: %v", said.err)
		}
		if said.drawn != nil {
			drawn++
		}
		ended = said.done && said.stage == stage
	}
	if drawn != len(ladderFor(factsOf(stage), testScreen())) || !ended {
		t.Errorf("%d frames told, ended %v", drawn, ended)
	}
}

// anti-test of TestWorkerTellsARefusal
// a scene that panics in a frame is told as a refusal and the plan
// still ends, so the loop can land.
func TestWorkerTellsAPanicAndEnds(t *testing.T) {
	control := gomock.NewController(t)
	stage := framed(control, one, func(context.Context, int, Size) (Drawn,
		error) {
		panic("the scene fell over")
	})
	out := make(chan message, 16)
	working := worker{now: time.Now, out: out}
	working.run(context.Background(), plan{stage: stage,
		screen: testScreen()})
	said := await(t, out, "the panic told")
	if said.err == nil || !said.done ||
		!strings.Contains(said.err.Error(), "fell over") {
		t.Errorf("a panic was told as %+v", said)
	}
}

// anti-test of TestWorkerClosesTheStageItReplaces
// a stage lent and not to be replaced is used as it is, neither closed
// nor opened again; a stage that closes well gives the new one; a
// stager that refuses gives its refusal and no stage.
func TestWorkerKeepsAStageItIsNotToReplace(t *testing.T) {
	control := gomock.NewController(t)
	lent := NewMockStage(control) // any call on it fails the test
	stager := NewMockStager(control)
	out := make(chan message, 8)
	working := worker{stager: stager, now: time.Now, out: out}
	got, err := working.staged(context.Background(), plan{stage: lent})
	if err != nil || got != Stage(lent) {
		t.Errorf("a stage not to be replaced gave %v, %v", got, err)
	}
	old := NewMockStage(control)
	old.EXPECT().Close().Return(nil).Times(one)
	opened := stageOf(control, one, nil)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(opened, nil)
	got, err = working.staged(context.Background(), plan{stage: old,
		restage: true})
	if err != nil || got != Stage(opened) {
		t.Errorf("a clean restage gave %v, %v", got, err)
	}
	if said := <-out; !said.staged || said.facts.width != 40 {
		t.Errorf("the new stage's facts were told as %+v", said)
	}
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("no such scene"))
	got, err = working.staged(context.Background(), plan{restage: true})
	if err == nil || got != nil || len(out) != none {
		t.Errorf("a refused staging gave %v, %v, %d told", got, err,
			len(out))
	}
}

// anti-test of TestPublishSaysOnceWhenItCannot
// a folder that can be written is written, once whole and with nothing
// left beside it, and nothing is said; a status that cannot be put in
// place leaves the one before and no stray file.
func TestPublishIsSilentWhenItCan(t *testing.T) {
	watching := idle(t, stillJob)
	folder := filepath.Join(t.TempDir(), "status")
	watching.given.Folder = folder
	watching.publish(nil, nil, testPixel, false)
	if said := statusOf(folder); said.Schema != statusSchema ||
		said.Job != watching.given.Path {
		t.Errorf("the status says %+v", said)
	}
	if strings.Contains(strings.Join(watching.foot.lines(), "\n"),
		"not written") {
		t.Errorf("the foot said %q", watching.foot.lines())
	}
	if entries, _ := os.ReadDir(folder); len(entries) != one {
		t.Errorf("the folder holds %d files", len(entries))
	}
	place := filepath.Join(folder, "taken")
	if err := os.Mkdir(place, folderMode); err != nil {
		t.Fatal(err)
	}
	if err := replaced(place, []byte("x"), fileMode); err == nil {
		t.Error("a file was renamed over a folder")
	}
	if entries, _ := os.ReadDir(folder); len(entries) != 2 {
		t.Errorf("a failed write left %d files", len(entries))
	}
}

// anti-test of TestDevelopmentRefusedIsSaid
// a job whose development can be made is developed again and the
// foot is quiet.
func TestDevelopmentOfAGoodJobMovesOn(t *testing.T) {
	watching := idle(t, stillJob)
	version := watching.version
	watching.developer()
	if watching.version != version+one {
		t.Errorf("the version is %d, was %d", watching.version, version)
	}
	if got := strings.TrimSpace(strings.Join(watching.foot.lines(),
		"")); got != "" {
		t.Errorf("the foot says %q", got)
	}
}

// anti-test of TestPollOfAFileThatIsGone
// a file that is there says nothing; one that goes and comes back
// clears the trouble.
func TestPollOfAFileThatIsThere(t *testing.T) {
	watching := idle(t, filmJob)
	path := watching.given.Path
	watching.poll()
	if watching.foot.trouble != "" || watching.pending != Unchanged {
		t.Errorf("a file that is there gave %q, owing %v",
			watching.foot.trouble, watching.pending)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	watching.poll()
	if watching.foot.trouble == "" {
		t.Fatal("a file that is gone was not said")
	}
	save(t, path, filmJob)
	watching.poll()
	if watching.foot.trouble != "" {
		t.Errorf("a file that came back is still troubled: %q",
			watching.foot.trouble)
	}
}

// anti-test of TestWriteEditsKeepsTheMode
// the mode kept is the file's own, not a fixed one.
func TestWriteEditsDoesNotForceAMode(t *testing.T) {
	for _, mode := range []os.FileMode{0o644, 0o640} {
		path := jobFile(t, stillJob)
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		if err := writeEdits(path, edits{camera: "near"}); err != nil {
			t.Fatal(err)
		}
		if info, err := os.Stat(path); err != nil ||
			info.Mode().Perm() != mode {
			t.Errorf("a file of %v is now %v, %v", mode, info.Mode(), err)
		}
	}
}

// anti-test of TestWatchFoldsKeysIntoOneChange
// a key that changes nothing does not cancel the frame being drawn:
// the ladder goes on up from the rung it was at, it does not begin
// again.
func TestWatchLeavesAFrameAloneForAKeyThatChangesNothing(t *testing.T) {
	control := gomock.NewController(t)
	gate := make(chan struct{})
	sizes := make(chan Size, 64)
	stage := framed(control, filmFrames, func(_ context.Context, _ int,
		size Size) (Drawn, error) {
		sizes <- size
		<-gate
		return drawnOf(control, max(one, size.Width()),
			max(one, size.Height())), nil
	})
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(stage, nil)
	events := make(chan terminal.Event, 8)
	watching := watcherOn(t, jobFile(t, filmJob), stager,
		screenOf(control, events, nil, nil), noon(), make(chan time.Time))
	running(t, watching)
	ladder := ladderFor(facts{width: 40, height: 30}, testScreen())
	if got := await(t, sizes, "the first rung"); got != ladder[0].size {
		t.Fatalf("the first rung is %+v", got)
	}
	time.Sleep(50 * time.Millisecond)                 // the loop takes the stage's facts
	events <- terminal.KeyEvent(pressed(t, "\x1b[D")) // left at the start
	time.Sleep(50 * time.Millisecond)
	close(gate)
	if got := await(t, sizes, "the second rung"); got != ladder[1].size {
		t.Errorf("after a key that changes nothing the next rung is "+
			"%+v, want %+v", got, ladder[1].size)
	}
}

// anti-test of TestWatchFoldsKeysIntoOneChange
// a frame cancelled by a key is not a trouble: nothing is said at the
// foot or in the status, and the frame the key asked for is the next.
func TestWatchCancelledFrameIsNoTroubleAtTheFoot(t *testing.T) {
	control := gomock.NewController(t)
	asked := make(chan int, 64)
	footed := make(chan []string, 256)
	stage := framed(control, filmFrames, func(ctx context.Context,
		index int, size Size) (Drawn, error) {
		asked <- index
		if index == none {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return drawnOf(control, max(one, size.Width()),
			max(one, size.Height())), nil
	})
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(stage, nil)
	events := make(chan terminal.Event, 8)
	path := jobFile(t, filmJob)
	watching := watcherOn(t, path, stager,
		screenOf(control, events, nil, footed), noon(),
		make(chan time.Time))
	running(t, watching)
	if got := await(t, asked, "the first frame"); got != none {
		t.Fatalf("the first frame asked is %d", got)
	}
	time.Sleep(50 * time.Millisecond)
	events <- terminal.KeyEvent(pressed(t, "\x1b[C"))
	if got := await(t, asked, "the next frame"); got != one {
		t.Fatalf("after a right the frame asked is %d", got)
	}
	time.Sleep(100 * time.Millisecond)
	for len(footed) > none {
		if lines := strings.Join(<-footed, "\n"); strings.Contains(lines,
			"cancel") {
			t.Errorf("a cancelled frame was said at the foot: %q", lines)
		}
	}
	if said := statusOf(filepath.Join(filepath.Dir(path),
		".test.watch")); said.Error != "" {
		t.Errorf("the status says %q", said.Error)
	}
}

// anti-test of TestWatchKeepsThePictureThroughABadSave
// a job that does not read at the start is not staged, says why at the
// foot and in the status, with no picture; the save that mends it is
// staged and clears both.
func TestWatchStartedOnABrokenJobWaitsForItToRead(t *testing.T) {
	control := gomock.NewController(t)
	staged := make(chan struct{}, 4)
	drew := make(chan Size, 64)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, job.Job) (Stage, error) {
			staged <- struct{}{}
			return stageOf(control, one, drew), nil
		}).Times(one)
	path := jobFile(t, stillJob+"film: [not a film\n")
	folder := filepath.Join(filepath.Dir(path), ".test.watch")
	ticks := make(chan time.Time)
	now := noon()
	watching := watcherOn(t, path, stager,
		screenOf(control, make(chan terminal.Event), nil, nil), now, ticks)
	running(t, watching)
	broken := statusWhen(t, folder, "the job does not read",
		func(said report) bool { return said.Error != "" })
	if broken.Pictures != none || broken.State == stateDrawing {
		t.Errorf("a job that does not read has status %+v", broken)
	}
	if _, err := os.Stat(filepath.Join(folder, framePicture)); err == nil {
		t.Error("a picture was written for a job that does not read")
	}
	now.pass(PollEvery)
	ticks <- now.Now()
	select {
	case <-staged:
		t.Fatal("a job that does not read was staged")
	case <-time.After(100 * time.Millisecond):
	}
	save(t, path, stillJob)
	now.pass(PollEvery)
	ticks <- now.Now()
	await(t, staged, "the staging of the mended job")
	ladder := ladderFor(facts{width: 40, height: 30}, testScreen())
	for range ladder {
		await(t, drew, "a rung")
	}
	mended := statusWhen(t, folder, "the mended job drawn",
		func(said report) bool { return said.Pictures == len(ladder) })
	if mended.Error != "" {
		t.Errorf("the status still says %q", mended.Error)
	}
}

// anti-test of TestWatchEndsAtTheEndOfInput
// a key that is not q, the host's key and a tick do not end the
// watcher; q does.
func TestWatchKeepsGoingOnKeysThatAreNotQuit(t *testing.T) {
	control := gomock.NewController(t)
	drew := make(chan Size, 64)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(stageOf(control, one, drew), nil)
	events := make(chan terminal.Event, 4)
	ticks := make(chan time.Time, 1)
	now := noon()
	watching := watcherOn(t, jobFile(t, stillJob), stager,
		screenOf(control, events, nil, nil), now, ticks)
	ended := running(t, watching)
	await(t, drew, "the first rung")
	events <- keyOf(t, 'x')
	events <- terminal.KeyEvent(pressed(t, "\x1d"))
	now.pass(PollEvery)
	ticks <- now.Now()
	select {
	case err := <-ended:
		t.Fatalf("the watcher ended on a key that is not q: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	events <- keyOf(t, 'q')
	if err := await(t, ended, "the end"); err != nil {
		t.Errorf("q gave %v", err)
	}
}

// anti-test of TestTheStatusFolderIsTheSeam
// while watch runs the status does not say stopped, has the schema and
// a time that parses, and the folder holds only the three files.
func TestTheStatusFolderIsNotStoppedWhileRunning(t *testing.T) {
	control := gomock.NewController(t)
	drew := make(chan Size, 16)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(stageOf(control, one, drew), nil)
	path := jobFile(t, stillJob)
	watching := watcherOn(t, path, stager,
		screenOf(control, make(chan terminal.Event), nil, nil), noon(),
		make(chan time.Time))
	running(t, watching)
	ladder := ladderFor(facts{width: 40, height: 30}, testScreen())
	folder := filepath.Join(filepath.Dir(path), ".test.watch")
	said := statusWhen(t, folder, "the ladder shown", func(said report) bool {
		return said.Pictures == len(ladder) && said.State == stateShown
	})
	if said.Schema != statusSchema || said.State == stateStopped {
		t.Errorf("the status says %+v", said)
	}
	if _, err := time.Parse(time.RFC3339Nano, said.When); err != nil {
		t.Errorf("the status time %q does not read: %v", said.When, err)
	}
	if entries, _ := os.ReadDir(folder); len(entries) != 3 {
		t.Errorf("the folder holds %d files", len(entries))
	}
}

// anti-test of TestTheStatusFolderIsTheSeam
// a job the stage refuses is in the status as an error, with no
// picture counted and none written.
func TestTheStatusFolderCarriesARefusal(t *testing.T) {
	control := gomock.NewController(t)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("no scene is called that"))
	path := jobFile(t, stillJob)
	watching := watcherOn(t, path, stager,
		screenOf(control, make(chan terminal.Event), nil, nil), noon(),
		make(chan time.Time))
	running(t, watching)
	folder := filepath.Join(filepath.Dir(path), ".test.watch")
	said := statusWhen(t, folder, "the refusal", func(said report) bool {
		return strings.Contains(said.Error, "no scene is called that")
	})
	if said.Pictures != none || said.State == stateDrawing {
		t.Errorf("a refused job has status %+v", said)
	}
	if _, err := os.Stat(filepath.Join(folder, framePicture)); err == nil {
		t.Error("a picture was written for a refused job")
	}
}
