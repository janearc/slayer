package live

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"go.uber.org/mock/gomock"
)

// noon is the tests' clock's start.
func noon() *clock {
	return &clock{now: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)}
}

// a job is staged once and its frame drawn up the ladder in order, each
// rung presented as it comes, and the status folder written.
func TestWatchDrawsCoarseThenFine(t *testing.T) {
	control := gomock.NewController(t)
	drew := make(chan Size, 16)
	shown := make(chan *terminal.Cells, 16)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(stageOf(control, one, drew), nil).Times(one)
	path := jobFile(t, stillJob)
	screen := screenOf(control, make(chan terminal.Event), shown, nil)
	watching := watcherOn(t, path, stager,
		screen, noon(),
		make(chan time.Time))
	running(t, watching)
	want := ladderFor(facts{width: 40, height: 30}, testScreen())
	for i, each := range want {
		if got := await(t, drew, "rung"); got != each.size {
			t.Fatalf("rung %d drew %+v, want %+v", i, got, each.size)
		}
		cells := await(t, shown, "picture")
		if cells.W != testCols {
			t.Errorf("a picture %d cells wide", cells.W)
		}
	}
	status := filepath.Join(filepath.Dir(path), ".test.watch", statusFile)
	deadline := time.Now().Add(waitLimit)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(status)
		var said report
		if err == nil && json.Unmarshal(raw, &said) == nil &&
			said.Rays == 4 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("the status folder never said the last rung")
}

// anti-test of TestWatchDrawsCoarseThenFine
// a rung predicted past the ceiling is not started: a clock
// that says the first rung took the ceiling stops the ladder there.
func TestWatchStopsAtTheCeiling(t *testing.T) {
	control := gomock.NewController(t)
	drew := make(chan Size, 16)
	now := noon()
	stage := NewMockStage(control)
	plain := stageOf(control, one, nil)
	stage.EXPECT().Times().DoAndReturn(plain.Times).AnyTimes()
	stage.EXPECT().Tone().DoAndReturn(plain.Tone).AnyTimes()
	stage.EXPECT().Shape().DoAndReturn(plain.Shape).AnyTimes()
	stage.EXPECT().Exposure().DoAndReturn(plain.Exposure).AnyTimes()
	stage.EXPECT().Cameras().DoAndReturn(plain.Cameras).AnyTimes()
	stage.EXPECT().Close().Return(nil).AnyTimes()
	stage.EXPECT().Frame(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int, size Size) (Drawn,
			error) {
			drew <- size
			now.pass(Ceiling)
			return drawnOf(control, size.Width(), size.Height()), nil
		}).Times(one)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(stage, nil)
	shown := make(chan *terminal.Cells, 16)
	jobPath := jobFile(t, stillJob)
	screen := screenOf(control, make(chan terminal.Event), shown, nil)
	watching := watcherOn(t, jobPath, stager,
		screen, now,
		make(chan time.Time))
	running(t, watching)
	await(t, drew, "the first rung")
	await(t, shown, "the first picture")
	select {
	case size := <-drew:
		t.Errorf("a rung past the ceiling was drawn: %+v", size)
	case <-time.After(100 * time.Millisecond):
	}
}

// keys pressed while a frame is drawn cancel it, and are folded into one
// change: three rights draw frame three once, not one, two and three.
func TestWatchFoldsKeysIntoOneChange(t *testing.T) {
	control := gomock.NewController(t)
	gate := make(chan struct{})
	asked := make(chan int, 64)
	stage := stageOf(control, filmFrames, nil)
	gated := NewMockStage(control)
	gated.EXPECT().Times().DoAndReturn(stage.Times).AnyTimes()
	gated.EXPECT().Tone().DoAndReturn(stage.Tone).AnyTimes()
	gated.EXPECT().Shape().DoAndReturn(stage.Shape).AnyTimes()
	gated.EXPECT().Exposure().DoAndReturn(stage.Exposure).AnyTimes()
	gated.EXPECT().Cameras().DoAndReturn(stage.Cameras).AnyTimes()
	gated.EXPECT().Close().Return(nil).AnyTimes()
	gated.EXPECT().Frame(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, index int, size Size) (
			Drawn, error) {
			asked <- index
			<-gate
			width := max(one, size.Width())
			height := max(one, size.Height())
			return drawnOf(control, width,
				height), nil
		}).AnyTimes()
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(gated, nil)
	events := make(chan terminal.Event, 8)
	watching := watcherOn(t, jobFile(t, filmJob), stager,
		screenOf(control, events, nil, nil), noon(), make(chan time.Time))
	running(t, watching)
	if got := await(t, asked, "the first frame"); got != none {
		t.Fatalf("the first frame asked is %d", got)
	}
	time.Sleep(50 * time.Millisecond) // the loop takes the stage's facts
	right := terminal.KeyEvent(pressed(t, "\x1b[C"))
	events <- right
	events <- right
	events <- right
	time.Sleep(50 * time.Millisecond)
	close(gate)
	if got := await(t, asked, "the next frame"); got != 3 {
		t.Errorf("after three rights the next frame drawn is %d", got)
	}
}

// a job that stops reading keeps the last picture and says why at the
// foot; fixing it clears that, and a change of print develops again
// without drawing.
func TestWatchKeepsThePictureThroughABadSave(t *testing.T) {
	control := gomock.NewController(t)
	drew := make(chan Size, 64)
	footed := make(chan []string, 64)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(stageOf(control, one, drew), nil).Times(one)
	path := jobFile(t, stillJob)
	ticks := make(chan time.Time)
	now := noon()
	screen := screenOf(control, make(chan terminal.Event), nil, footed)
	watching := watcherOn(t, path, stager,
		screen, now,
		ticks)
	running(t, watching)
	for range ladderFor(facts{width: 40, height: 30}, testScreen()) {
		await(t, drew, "a rung")
	}
	save(t, path, stillJob+"film: [not a film\n")
	now.pass(PollEvery)
	ticks <- now.Now()
	if !footSays(t, footed, "test.yaml") {
		t.Error("the foot never said the file does not read")
	}
	save(t, path, stillJob+"print: {knee: 0.5, top: 4}\n")
	now.pass(PollEvery)
	ticks <- now.Now()
	if !footSays(t, footed, "the job reads again") {
		t.Error("the foot never said the job reads again")
	}
	select {
	case size := <-drew:
		t.Errorf("a print drew the frame again at %+v", size)
	case <-time.After(100 * time.Millisecond):
	}
}

// save writes a job file with a time that differs from the last, so a
// poll sees it.
func save(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), fileMode); err != nil {
		t.Fatal(err)
	}
	age := time.Duration(len(text)) * time.Second
	later := time.Now().Add(age)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
}

// footSays is whether a foot written soon holds words.
func footSays(t *testing.T, footed <-chan []string, words string) bool {
	t.Helper()
	deadline := time.After(waitLimit)
	for {
		select {
		case lines := <-footed:
			if strings.Contains(strings.Join(lines, "\n"), words) {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// anti-test of TestWatchDrawsCoarseThenFine
// a stage that refuses the job draws nothing, says the
// refusal at the foot, and the watcher still quits on q.
func TestWatchSaysARefusalAndQuits(t *testing.T) {
	control := gomock.NewController(t)
	footed := make(chan []string, 64)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("no scene is called that"))
	events := make(chan terminal.Event, 1)
	watching := watcherOn(t, jobFile(t, stillJob), stager,
		screenOf(control, events, nil, footed), noon(),
		make(chan time.Time))
	ended := running(t, watching)
	if !footSays(t, footed, "no scene is called that") {
		t.Error("the refusal was not said")
	}
	events <- terminal.KeyEvent(pressed(t, "q"))
	if err := await(t, ended, "the end"); err != nil {
		t.Errorf("quitting gave %v", err)
	}
}

// the end of input, and a resize, are taken: the one ends the watcher,
// the other draws the ladder again.
func TestWatchEndsAtTheEndOfInput(t *testing.T) {
	control := gomock.NewController(t)
	drew := make(chan Size, 64)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(stageOf(control, one, drew), nil)
	events := make(chan terminal.Event, 2)
	watching := watcherOn(t, jobFile(t, stillJob), stager,
		screenOf(control, events, nil, nil), noon(), make(chan time.Time))
	ended := running(t, watching)
	first := await(t, drew, "the first rung")
	events <- terminal.ResizeEvent(testScreen())
	for {
		if got := await(t, drew, "a rung"); got == first {
			break
		}
	}
	events <- terminal.EndEvent()
	if err := await(t, ended, "the end"); err != nil {
		t.Errorf("the end of input gave %v", err)
	}
}

// anti-test of TestNewTakesEveryPiece
// New refuses each missing piece by name, and a path that is not
// absolute.
func TestNewRefusesWhatIsMissing(t *testing.T) {
	_, err := New(Given{Path: "relative.yaml"})
	for _, name := range []string{"absolute", "reader", "stager",
		"screen", "clock", "ticks"} {
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("a watcher with no %s was not refused by name: %v",
				name, err)
		}
	}
}

// a watcher with every piece is made, and writes nothing yet.
func TestNewTakesEveryPiece(t *testing.T) {
	control := gomock.NewController(t)
	path := jobFile(t, stillJob)
	watching, err := New(Given{Path: path, Read: job.Read,
		Stager: NewMockStager(control), Screen: NewMockScreen(control),
		Now: time.Now, Ticks: make(chan time.Time)})
	if err != nil || watching == nil {
		t.Fatalf("a whole watcher was refused: %v", err)
	}
	watchPath := filepath.Join(filepath.Dir(path), ".test.watch")
	if _, err := os.Stat(watchPath); err == nil {
		t.Error("a watcher with no folder given wrote one")
	}
}

// the status folder holds the latest picture and the cells as shown,
// and a status that says what they are, counts them, and says stopped
// once watch has quit.
func TestTheStatusFolderIsTheSeam(t *testing.T) {
	control := gomock.NewController(t)
	drew := make(chan Size, 16)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).
		Return(stageOf(control, one, drew), nil)
	path := jobFile(t, stillJob)
	events := make(chan terminal.Event, 1)
	watching := watcherOn(t, path, stager,
		screenOf(control, events, nil, nil), noon(), make(chan time.Time))
	ended := running(t, watching)
	ladder := ladderFor(facts{width: 40, height: 30}, testScreen())
	for range ladder {
		await(t, drew, "a rung")
	}
	folder := filepath.Join(filepath.Dir(path), ".test.watch")
	deadline := time.Now().Add(waitLimit)
	for time.Now().Before(deadline) {
		raw, _ := os.ReadFile(filepath.Join(folder, statusFile))
		var said report
		if json.Unmarshal(raw, &said) == nil &&
			said.Pictures == len(ladder) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	events <- terminal.KeyEvent(pressed(t, "q"))
	await(t, ended, "the end")
	raw, err := os.ReadFile(filepath.Join(folder, statusFile))
	if err != nil {
		t.Fatal(err)
	}
	var said report
	if err := json.Unmarshal(raw, &said); err != nil {
		t.Fatal(err)
	}
	last := ladder[len(ladder)-one].size
	if said.Schema != statusSchema || said.State != stateStopped ||
		said.Job != path || said.Pictures != len(ladder) ||
		said.Width != last.width || said.Rays != last.rays ||
		said.Frames != one || said.Mode != "preview" {
		t.Errorf("the status says %+v", said)
	}
	for _, name := range []string{framePicture, screenPicture} {
		if _, err := os.Stat(filepath.Join(folder, name)); err != nil {
			t.Errorf("%s is not there: %v", name, err)
		}
	}
	entries, _ := os.ReadDir(folder)
	if len(entries) != 3 {
		t.Errorf("the folder holds %d files, a half-written one among "+
			"them", len(entries))
	}
}

// a save of the set the job names stages the scene again, as a save of
// the job does; a save of another file beside it changes nothing.
func TestWatchFollowsTheSet(t *testing.T) {
	control := gomock.NewController(t)
	staged := make(chan struct{}, 8)
	stager := NewMockStager(control)
	stager.EXPECT().Stage(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, job.Job) (Stage, error) {
			staged <- struct{}{}
			return stageOf(control, one, nil), nil
		}).AnyTimes()
	directory := t.TempDir()
	set := filepath.Join(directory, "set.yaml")
	other := filepath.Join(directory, "other.yaml")
	save(t, set, "one")
	save(t, other, "one")
	path := filepath.Join(directory, "test.yaml")
	save(t, path, "$schema: https://github.com/janearc/slayer/schemas/"+
		"job-v1.json\nscene: {set: "+set+"}\n"+
		"film: {width: 40, height: 30, rays: 1}\n")
	ticks := make(chan time.Time)
	now := noon()
	screen := screenOf(control, make(chan terminal.Event), nil, nil)
	watching := watcherOn(t, path, stager,
		screen, now, ticks)
	running(t, watching)
	await(t, staged, "the first staging")
	tick := func() {
		now.pass(PollEvery)
		ticks <- now.Now()
	}
	tick()
	// anti-test of TestWatchFollowsTheSet
	save(t, other, "two, longer")
	tick()
	select {
	case <-staged:
		t.Fatal("a file the job does not name staged the scene again")
	case <-time.After(200 * time.Millisecond):
	}
	save(t, set, "two, longer")
	tick()
	await(t, staged, "staging after the set was saved")
}
