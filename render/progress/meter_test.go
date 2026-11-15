package progress_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/render/progress"
	"pgregory.net/rapid"
)

// what the meter tests count in. each is picked by the test author,
// 2026-11-12, as a job small enough to follow by hand.
const (
	someFrames = 4  // picked: frames in the worked job
	someRows   = 10 // picked: rows a frame in the worked job
)

// ticking is a clock the test moves by hand.
type ticking struct{ at time.Time }

func (clock *ticking) now() time.Time { return clock.at }

func (clock *ticking) pass(d time.Duration) { clock.at = clock.at.Add(d) }

// failer is what the helpers need of a test: both testing.T and rapid.T.
type failer interface {
	Helper()
	Fatal(args ...any)
}

// begun is a meter on a ticking clock, for a job of frames by rows.
func begun(t failer, frames, rows int) (*progress.Meter, *ticking) {
	t.Helper()
	clock := &ticking{at: origin}
	meter, err := progress.NewMeter(clock.now)
	if err != nil {
		t.Fatal(err)
	}
	if err := meter.Begin(frames, rows); err != nil {
		t.Fatal(err)
	}
	return meter, clock
}

func mustNil(t failer, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// anti-test: no meter without a clock.
func TestAMeterNeedsAClock(t *testing.T) {
	if meter, err := progress.NewMeter(nil); err == nil || meter != nil {
		t.Errorf("made %v, %v", meter, err)
	}
}

// anti-test: a job has frames and rows, and is begun once.
func TestAJobIsRefusedWhenItHasNoShapeOrBeginsTwice(t *testing.T) {
	for what, shape := range map[string][2]int{
		"no frames":       {0, 4},
		"no rows":         {4, 0},
		"negative frames": {-1, 4},
		"too many rows":   {math.MaxInt, 2},
	} {
		meter, _ := progress.NewMeter(time.Now)
		if err := meter.Begin(shape[0], shape[1]); err == nil {
			t.Errorf("%s: begun", what)
		}
		if meter.Frames() != 0 {
			t.Errorf("%s: a refused job left %d frames", what,
				meter.Frames())
		}
	}
	meter, _ := begun(t, someFrames, someRows)
	if err := meter.Begin(1, 1); err == nil {
		t.Error("a meter began a second job")
	}
}

// rows and frames that finish move the share by hand: 4 frames of 10 rows
// is 40 rows; 5 rows is 1/8, and the first frame ended at 6 rows is 10 of
// 40, a quarter.
func TestRowsAndFramesMoveTheShare(t *testing.T) {
	meter, _ := begun(t, someFrames, someRows)
	if meter.Share() != 0 || meter.RowsTotal() != 40 || meter.Frame() != 1 {
		t.Fatalf("at the start: share %v of %d rows, frame %d",
			meter.Share(), meter.RowsTotal(), meter.Frame())
	}
	mustNil(t, meter.Rows(5))
	if meter.Share() != 0.125 || meter.RowsDone() != 5 {
		t.Errorf("after 5 rows: share %v, rows %d", meter.Share(),
			meter.RowsDone())
	}
	mustNil(t, meter.Rows(1))
	mustNil(t, meter.FrameDone())
	if meter.Share() != 0.25 || meter.Frame() != 2 || meter.FramesDone() != 1 {
		t.Errorf("after a frame: share %v frame %d done %d",
			meter.Share(), meter.Frame(), meter.FramesDone())
	}
}

// the last frame's end makes the share one, and the frame stays the last.
func TestTheLastFrameMakesTheShareOne(t *testing.T) {
	meter, _ := begun(t, 2, 3)
	mustNil(t, meter.FrameDone())
	mustNil(t, meter.FrameDone())
	if meter.Share() != 1 || meter.Frame() != 2 || meter.RowsDone() != 6 {
		t.Errorf("share %v frame %d rows %d", meter.Share(), meter.Frame(),
			meter.RowsDone())
	}
}

// anti-test: nothing is taken that is out of order or out of range, and a
// refusal changes nothing.
func TestAMeterRefusesWhatCannotHaveHappened(t *testing.T) {
	cold, _ := progress.NewMeter(time.Now)
	if cold.Rows(1) == nil || cold.FrameDone() == nil {
		t.Error("a meter that had not begun took news")
	}
	meter, _ := begun(t, 1, someRows)
	mustNil(t, meter.Rows(3))
	for what, count := range map[string]int{"negative": -1,
		"more than the frame has left": someRows - 2} {
		if meter.Rows(count) == nil {
			t.Errorf("%s: taken", what)
		}
	}
	if meter.RowsDone() != 3 {
		t.Errorf("refusals moved the rows to %d", meter.RowsDone())
	}
	mustNil(t, meter.FrameDone())
	if meter.Rows(1) == nil || meter.FrameDone() == nil {
		t.Error("a finished job took more news")
	}
}

// elapsed counts from Begin, and stops at the last frame's end.
// anti-test: a clock that runs backwards does not make it negative.
func TestElapsedRunsFromBeginAndStopsAtTheEnd(t *testing.T) {
	meter, clock := begun(t, 1, someRows)
	clock.pass(3 * time.Second)
	if meter.Elapsed() != 3*time.Second {
		t.Errorf("elapsed %v", meter.Elapsed())
	}
	clock.pass(-5 * time.Second)
	if meter.Elapsed() != 0 {
		t.Errorf("a clock gone back gave %v", meter.Elapsed())
	}
	clock.pass(7 * time.Second)
	mustNil(t, meter.FrameDone())
	clock.pass(time.Hour)
	if meter.Elapsed() != 5*time.Second {
		t.Errorf("a finished job's elapsed moved to %v", meter.Elapsed())
	}
}

// the time left is not offered until both the minimum time and the minimum
// rows are there. a job told nothing, a job with time and no rows, and a job
// with rows and no time each offer none.
func TestRemainingIsNotOfferedWithoutSomethingToBaseItOn(t *testing.T) {
	cold, _ := progress.NewMeter(time.Now)
	if _, offered := cold.Remaining(); offered {
		t.Error("a meter told nothing offered a time left")
	}
	meter, clock := begun(t, 1, progress.MinRows*4)
	clock.pass(progress.MinElapsed * 5)
	if _, offered := meter.Remaining(); offered {
		t.Error("time and no rows offered a time left")
	}
	mustNil(t, meter.Rows(progress.MinRows))
	if _, offered := meter.Remaining(); !offered {
		t.Error("time and rows offered none")
	}
	short, shortClock := begun(t, 1, progress.MinRows*4)
	mustNil(t, short.Rows(progress.MinRows))
	shortClock.pass(progress.MinElapsed - 1)
	if _, offered := short.Remaining(); offered {
		t.Error("rows and under the minimum time offered a time left")
	}
}

// before a frame is finished the time left is the pace in rows: 8 rows of
// 32 in 4 s leaves 24 rows, 12 s.
func TestRemainingBeforeAFrameIsThePaceInRows(t *testing.T) {
	meter, clock := begun(t, 1, progress.MinRows*4)
	clock.pass(4 * time.Second)
	mustNil(t, meter.Rows(progress.MinRows))
	left, offered := meter.Remaining()
	if !offered || left != 12*time.Second {
		t.Errorf("left %v, offered %v; want 12s", left, offered)
	}
}

// once a frame is finished its duration is used, not the rows: two frames of
// 10 s done in 20 s leave two more, 20 s, and 4 s into the third leaves
// 6 s of it and 10 s of the fourth, 16 s.
func TestRemainingUsesFinishedFramesDurations(t *testing.T) {
	meter, clock := begun(t, 4, progress.MinRows)
	for range 2 {
		clock.pass(10 * time.Second)
		mustNil(t, meter.FrameDone())
	}
	left, _ := meter.Remaining()
	if left != 20*time.Second {
		t.Errorf("at a frame's start: left %v, want 20s", left)
	}
	clock.pass(4 * time.Second)
	left, _ = meter.Remaining()
	if left != 16*time.Second {
		t.Errorf("4s into a frame: left %v, want 16s", left)
	}
}

// anti-test: a frame that runs past the mean leaves its share at nought,
// never below, and a finished job is offered with none left.
func TestRemainingIsNeverNegative(t *testing.T) {
	meter, clock := begun(t, 2, progress.MinRows)
	clock.pass(2 * time.Second)
	mustNil(t, meter.FrameDone())
	clock.pass(30 * time.Second)
	left, offered := meter.Remaining()
	if !offered || left != 0 {
		t.Errorf("a frame far past the mean: %v, %v; want 0", left,
			offered)
	}
	mustNil(t, meter.FrameDone())
	left, offered = meter.Remaining()
	if !offered || left != 0 {
		t.Errorf("a finished job: %v, %v; want 0 and offered", left,
			offered)
	}
}

// any job, any order of news: the share never falls, the time left is never
// negative, the elapsed time is never negative, and the frame stays in range.
func TestPropertyShareNeverFallsAndRemainingNeverNegative(t *testing.T) {
	rapid.Check(t, func(test *rapid.T) {
		frames := rapid.IntRange(1, 6).Draw(test, "frames")
		rows := rapid.IntRange(1, 20).Draw(test, "rows")
		meter, clock := begun(test, frames, rows)
		last := meter.Share()
		for step := 0; step < 80 && meter.FramesDone() < frames; step++ {
			clock.pass(time.Duration(rapid.IntRange(-2000, 5000).
				Draw(test, "millis")) * time.Millisecond)
			if rapid.Bool().Draw(test, "frame ends") {
				mustNil(test, meter.FrameDone())
			} else {
				left := rows - (meter.RowsDone() - meter.FramesDone()*rows)
				mustNil(test, meter.Rows(rapid.IntRange(0, left).
					Draw(test, "rows done")))
			}
			if got := meter.Share(); got < last || got > 1 {
				test.Fatalf("share went %v to %v", last, got)
			} else {
				last = got
			}
			if left, _ := meter.Remaining(); left < 0 {
				test.Fatalf("remaining %v", left)
			}
			if meter.Elapsed() < 0 || meter.Frame() > frames {
				test.Fatalf("elapsed %v frame %d", meter.Elapsed(),
					meter.Frame())
			}
		}
	})
}

// a meter told nothing, whatever the clock says, offers no time left.
func TestPropertyAMeterToldNothingOffersNoRemaining(t *testing.T) {
	rapid.Check(t, func(test *rapid.T) {
		at := time.Unix(rapid.Int64Range(0, 1<<40).Draw(test, "at"), 0)
		meter, err := progress.NewMeter(func() time.Time { return at })
		mustNil(test, err)
		if left, offered := meter.Remaining(); offered || left != 0 {
			test.Fatalf("offered %v, %v", left, offered)
		}
		if meter.Share() != 0 || meter.Elapsed() != 0 {
			test.Fatalf("share %v elapsed %v", meter.Share(),
				meter.Elapsed())
		}
	})
}

// a job so long that the time left would not fit a duration is held at the
// most a duration holds, not wrapped round to a negative.
func TestRemainingTooLongToHoldIsHeldAtTheMost(t *testing.T) {
	meter, clock := begun(t, math.MaxInt/progress.MinRows, progress.MinRows)
	clock.pass(time.Hour)
	mustNil(t, meter.FrameDone())
	left, offered := meter.Remaining()
	if !offered || left != time.Duration(math.MaxInt64) {
		t.Errorf("left %v, offered %v", left, offered)
	}
}
