package progress

import (
	"fmt"
	"math"
	"time"
)

// Clock is the time, given to a meter by its caller.
//
//ninja:vow practice NewMeter takes one; time.Now is one
type Clock func() time.Time

// Meter is the state of a job in flight: how many frames, how many rows
// to a frame, what has finished, and when. It reads the time from its clock
// at Begin and at each frame's end, and when it is asked. It is not safe
// for more than one goroutine.
//
//ninja:vow practice NewMeter makes one
type Meter struct {
	now          Clock
	begun        bool
	frames       int
	rowsPerFrame int
	started      time.Time
	frameStarted time.Time
	ended        time.Time
	framesDone   int
	rowsInFrame  int
	spent        time.Duration // the finished frames' durations, summed
}

// NewMeter is a meter that tells time by now. It refuses no clock.
func NewMeter(now Clock) (*Meter, error) {
	if now == nil {
		return nil, fmt.Errorf("progress: a meter needs a clock to " +
			"tell the time by, such as time.Now")
	}
	return &Meter{now: now}, nil
}

// Begin tells the meter the job's shape: frames frames of rowsPerFrame
// rows each, starting now. It refuses a count under one and a job begun
// already.
func (meter *Meter) Begin(frames, rowsPerFrame int) error {
	switch {
	case meter.begun:
		return fmt.Errorf("progress: this meter has begun a job; " +
			"make another meter for the next")
	case frames < fewest || rowsPerFrame < fewest:
		return fmt.Errorf("progress: a job is at least %d frame of "+
			"%d row, not %d of %d", fewest, fewest, frames,
			rowsPerFrame)
	case frames > math.MaxInt/rowsPerFrame:
		return fmt.Errorf("progress: %d frames of %d rows is more "+
			"rows than can be counted", frames, rowsPerFrame)
	}
	meter.begun, meter.frames, meter.rowsPerFrame = true, frames,
		rowsPerFrame
	meter.started = meter.now()
	meter.frameStarted = meter.started
	return nil
}

// Rows tells the meter that count more rows of the current frame are
// finished. It refuses a count below nought, more rows than the frame has
// left, a meter that has not begun and a job that is over.
func (meter *Meter) Rows(count int) error {
	if err := meter.running(); err != nil {
		return err
	}
	if count < none || count > meter.rowsPerFrame-meter.rowsInFrame {
		return fmt.Errorf("progress: %d more rows of a frame that has "+
			"%d of %d done: say between %d and %d", count,
			meter.rowsInFrame, meter.rowsPerFrame, none,
			meter.rowsPerFrame-meter.rowsInFrame)
	}
	meter.rowsInFrame += count
	return nil
}

// FrameDone tells the meter the current frame is finished, whatever rows
// it was told of. It refuses a meter that has not begun and a job that is
// over.
func (meter *Meter) FrameDone() error {
	if err := meter.running(); err != nil {
		return err
	}
	at := meter.now()
	meter.spent += max(at.Sub(meter.frameStarted), none)
	meter.frameStarted = at
	meter.framesDone++
	meter.rowsInFrame = none
	if meter.framesDone == meter.frames {
		meter.ended = at
	}
	return nil
}

// running is nil while the meter has begun and has frames left.
func (meter *Meter) running() error {
	switch {
	case !meter.begun:
		return fmt.Errorf("progress: tell the meter the job's shape " +
			"with Begin first")
	case meter.framesDone == meter.frames:
		return fmt.Errorf("progress: all %d frames are done; there "+
			"is nothing more to tell", meter.frames)
	}
	return nil
}

// Frames is how many frames the job has, nought before Begin.
func (meter *Meter) Frames() int { return meter.frames }

// FramesDone is how many frames are finished.
func (meter *Meter) FramesDone() int { return meter.framesDone }

// Frame is the number, from one, of the frame in flight, and the last
// frame's once all are done. Nought before Begin.
func (meter *Meter) Frame() int {
	return min(meter.framesDone+fewest, meter.frames)
}

// RowsTotal is how many rows the whole job has.
func (meter *Meter) RowsTotal() int { return meter.frames * meter.rowsPerFrame }

// RowsDone is how many rows are finished, the whole of every finished
// frame and what the frame in flight has told.
func (meter *Meter) RowsDone() int {
	return meter.framesDone*meter.rowsPerFrame + meter.rowsInFrame
}

// Share is the part of the job done, from nought to one. It is nought
// before Begin and never falls.
func (meter *Meter) Share() float64 {
	if meter.RowsTotal() == none {
		return none
	}
	return float64(meter.RowsDone()) / float64(meter.RowsTotal())
}

// Elapsed is the time since Begin, nought before it, and fixed at the last
// frame's end once the job is over. It is never negative.
func (meter *Meter) Elapsed() time.Duration {
	switch {
	case !meter.begun:
		return none
	case meter.framesDone == meter.frames:
		return max(meter.ended.Sub(meter.started), none)
	}
	return max(meter.now().Sub(meter.started), none)
}

// Remaining is the time left, and whether it is offered. It is not offered
// until MinElapsed has passed and MinRows rows are done, since before that
// there is nothing to base it on; a finished job has none left. Once a
// frame is finished the time is the mean of the finished frames' durations
// a frame, less what the frame in flight has had; before that it is the
// pace so far in rows. It is never negative.
func (meter *Meter) Remaining() (time.Duration, bool) {
	if !meter.begun {
		return none, false
	}
	if meter.framesDone == meter.frames {
		return none, true
	}
	if meter.Elapsed() < MinElapsed || meter.RowsDone() < MinRows {
		return none, false
	}
	if meter.framesDone > none {
		return meter.byFrames(), true
	}
	left := float64(meter.RowsTotal()-meter.RowsDone()) /
		float64(meter.RowsDone())
	return scaled(meter.Elapsed(), left), true
}

// byFrames is the time left by the finished frames' mean duration.
func (meter *Meter) byFrames() time.Duration {
	mean := meter.spent / time.Duration(meter.framesDone)
	into := max(meter.now().Sub(meter.frameStarted), none)
	current := max(mean-into, none)
	rest := scaled(mean, float64(meter.frames-meter.framesDone-fewest))
	return current + min(rest, math.MaxInt64-current)
}

// scaled is d times by, kept to what a duration can hold.
func scaled(d time.Duration, by float64) time.Duration {
	product := float64(d) * by
	if product >= math.MaxInt64 {
		return math.MaxInt64
	}
	return max(time.Duration(product), none)
}
