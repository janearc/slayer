package live

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
)

// The words and numbers of the tests' jobs and screens, each picked by
// the author of this slice, 2026-11-12, small so a test is quick.
const (
	jobLine = "$schema: https://github.com/janearc/slayer/schemas/" +
		"job-v1.json\nscene: {smoke: 1}\n"
	// a still four by three, and a film of two seconds at four frames
	stillJob = jobLine + "film: {width: 40, height: 30, rays: 1}\n"
	filmJob  = jobLine + "film: {width: 40, height: 30, rays: 1}\n" +
		"fps: 4\nduration: {value: 2, unit: s}\n"
	filmRate   = 4
	filmFrames = 8
	// a screen of 40 by 20 cells, its pixels as a terminal guesses them
	testCols, testRows = 40, 20
	testPixel          = 0.75
	// how long a test waits for the loop before it says nothing came
	waitLimit = 3 * time.Second
)

// testScreen is the tests' terminal's size.
func testScreen() terminal.Screen {
	return terminal.Screen{Cols: testCols, Rows: testRows,
		W: testCols * pair, H: testRows * 3, Pixel: testPixel}
}

// jobFile writes a job into a folder of the test's and gives its path.
func jobFile(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.yaml")
	if err := os.WriteFile(path, []byte(text), fileMode); err != nil {
		t.Fatal(err)
	}
	return path
}

// readJob is a job of the text, or the test stops.
func readJob(t *testing.T, text string) job.Job {
	t.Helper()
	read, err := job.Read(jobFile(t, text))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// clock is a clock the test moves.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

// Now is the clock's time.
func (made *clock) Now() time.Time {
	made.mu.Lock()
	defer made.mu.Unlock()
	return made.now
}

// pass moves the clock on.
func (made *clock) pass(span time.Duration) {
	made.mu.Lock()
	made.now = made.now.Add(span)
	made.mu.Unlock()
}

// lit is a frame of light width by height, every pixel grey.
func lit(width, height int) *frame.Frame {
	made := frame.New(width, height)
	for i := range made.Pix {
		made.Pix[i] = frame.Pixel{R: 0.5, G: 0.5, B: 0.5, Cover: 1}
	}
	return made
}

// drawnOf is a mock drawn frame of light of a size.
func drawnOf(control *gomock.Controller, width, height int) *MockDrawn {
	drawn := NewMockDrawn(control)
	drawn.EXPECT().Light().Return(lit(width, height)).AnyTimes()
	drawn.EXPECT().Overlay(gomock.Any()).Return(nil).AnyTimes()
	return drawn
}

// stageOf is a mock stage of a job's times, drawing light of the size
// each rung asks, the job's own where it asks none. Each frame drawn is
// told on drew.
func stageOf(control *gomock.Controller, times int,
	drew chan<- Size) *MockStage {
	stage := NewMockStage(control)
	moments := make([]time.Duration, times)
	for i := range moments {
		moments[i] = time.Duration(i) * time.Second / filmRate
	}
	stage.EXPECT().Times().Return(moments).AnyTimes()
	stage.EXPECT().Tone().Return(colour.Tone{Exposure: 1, Knee: 0.7,
		Top: 8}).AnyTimes()
	stage.EXPECT().Shape().Return(40, 30).AnyTimes()
	stage.EXPECT().Exposure().Return(2.0).AnyTimes()
	stage.EXPECT().Cameras().Return([]string{"near", "far"}).AnyTimes()
	stage.EXPECT().Frame(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int, size Size) (Drawn,
			error) {
			if drew != nil {
				drew <- size
			}
			width, height := size.Width(), size.Height()
			if width == none {
				width, height = 40, 30
			}
			return drawnOf(control, width, height), nil
		}).AnyTimes()
	stage.EXPECT().Close().Return(nil).AnyTimes()
	return stage
}

// screenOf is a mock screen of the tests' size, its events from events,
// each picture presented told on shown and each foot on footed.
func screenOf(control *gomock.Controller, events chan terminal.Event,
	shown chan<- *terminal.Cells, footed chan<- []string) *MockScreen {
	screen := NewMockScreen(control)
	screen.EXPECT().Events().Return((<-chan terminal.Event)(events)).
		AnyTimes()
	screen.EXPECT().Screen().Return(testScreen()).AnyTimes()
	screen.EXPECT().Present(gomock.Any()).DoAndReturn(
		func(cells *terminal.Cells) error {
			if shown != nil {
				select {
				case shown <- cells:
				default:
				}
			}
			return nil
		}).AnyTimes()
	screen.EXPECT().Foot(gomock.Any()).DoAndReturn(func(lines []string) error {
		if footed != nil {
			select {
			case footed <- lines:
			default:
			}
		}
		return nil
	}).AnyTimes()
	return screen
}

// watcherOn is a watcher of the job file at path, on a mock screen and
// stager, its clock and ticks the test's.
func watcherOn(t *testing.T, path string, stager Stager, screen Screen,
	now *clock, ticks chan time.Time) *Watcher {
	t.Helper()
	watchFolder := filepath.Join(filepath.Dir(path), ".test.watch")
	watching, err := New(Given{Path: path, Read: job.Read, Stager: stager,
		Screen: screen, Now: now.Now, Ticks: ticks,
		Folder: watchFolder})
	if err != nil {
		t.Fatal(err)
	}
	return watching
}

// running runs a watcher until the test ends, and gives what Run said
// on ended.
func running(t *testing.T, watching *Watcher) <-chan error {
	t.Helper()
	ended := make(chan error, one)
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		ended <- watching.Run(ctx)
		close(stopped)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-stopped:
		case <-time.After(waitLimit):
			t.Error("the watcher did not stop")
		}
	})
	return ended
}

// await is the next value on a channel, or the test stops.
func await[Value any](t *testing.T, from <-chan Value, what string) Value {
	t.Helper()
	select {
	case value := <-from:
		return value
	case <-time.After(waitLimit):
		t.Fatalf("no %s came", what)
	}
	var nothing Value
	return nothing
}

// keyOf is a key event of a character.
func keyOf(t *testing.T, char rune) terminal.Event {
	t.Helper()
	var decoder terminal.Decoder
	typed := string(char)
	keys := decoder.Feed([]byte(typed))
	if len(keys) != one {
		t.Fatalf("%q decoded as %d keys", char, len(keys))
	}
	return eventOf(t, keys[none])
}

// eventOf is a terminal event of a key. The terminal package makes
// events only from a terminal, so the test takes one from a decoder
// through a channel the way a Live would send it.
func eventOf(t *testing.T, key terminal.Key) terminal.Event {
	t.Helper()
	return terminal.KeyEvent(key)
}
