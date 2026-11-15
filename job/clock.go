package job

import (
	"math"
	"time"
)

// Clock is the job's time: a still's one moment, or a film's rate and
// span, and the scene's time at each of its frames.
type Clock struct {
	still bool
	at    *quantity
	rate  int
	// playing is the film's duration in seconds, as written: the frame
	// count is worked from it and not from a rounded duration.
	playing float64
	start   *quantity
	stretch *quantity
}

// Clock is the job's clock.
func (job Job) Clock() Clock {
	wire := job.wire
	clock := Clock{still: !wire.film(), at: wire.At, rate: wire.FPS,
		start: wire.Start, stretch: wire.Stretch}
	if wire.Duration != nil {
		clock.playing = wire.Duration.Value
	}
	return clock
}

// Times is the scene's time at each of the job's frames, in order.
func (job Job) Times() []time.Duration { return job.Clock().Times() }

// Still is whether the job is a still: it says no rate and no span.
func (clock Clock) Still() bool { return clock.still }

// At is a still's time, and whether the job said it.
func (clock Clock) At() (time.Duration, bool) {
	return spanOf(clock.at)
}

// Rate is a film's frames a second, from the job or its preset, and
// whether there is one; a still has none.
func (clock Clock) Rate() (int, bool) { return said(clock.rate) }

// Start is the scene's time at a film's first frame, and whether the job
// said it; unsaid, it is zero.
func (clock Clock) Start() (time.Duration, bool) {
	return spanOf(clock.start)
}

// Duration is how long a film plays; a still's is zero.
func (clock Clock) Duration() time.Duration {
	return quantity{Value: clock.playing}.duration()
}

// SceneDuration is how much scene time a film spans, stretched over its
// duration, and whether the job said it; unsaid, scene time runs with
// the film.
func (clock Clock) SceneDuration() (time.Duration, bool) {
	return spanOf(clock.stretch)
}

// Playback maps elapsed scene time back to film time. Audio and its cues
// run on this clock, even when an hour of flight plays in thirty seconds.
// The caller subtracts its scene's actual starting instant first.
func (clock Clock) Playback(elapsed time.Duration) time.Duration {
	span, stretched := clock.SceneDuration()
	if !stretched || span == 0 {
		return elapsed
	}
	return time.Duration(elapsed.Seconds() / span.Seconds() *
		clock.playing * float64(time.Second))
}

// spanOf is a quantity as a duration, and whether it was said.
func spanOf(amount *quantity) (time.Duration, bool) {
	if amount == nil {
		return nothing, false
	}
	return amount.duration(), true
}

// Frames is how many frames the job draws: one for a still, and for a
// film its duration times its rate, rounded, and never fewer than one.
func (clock Clock) Frames() int {
	if clock.still {
		return one
	}
	wanted := clock.playing * float64(clock.rate)
	return max(one, int(math.Round(wanted)))
}

// Times is the scene's time at each frame, in order. A still's one frame
// is at its at, or zero where it says none. A film's frame i is at start
// plus i over the rate, or, with a scene duration, at its share of it.
func (clock Clock) Times() []time.Duration {
	if clock.still {
		at, _ := clock.At()
		return []time.Duration{at}
	}
	times := make([]time.Duration, clock.Frames())
	for index := range times {
		times[index] = clock.frameTime(index)
	}
	return times
}

// frameTime is frame index's scene time, as a sequence plan's frameTime
// works it out, so a lifted plan's frames fall where they always did.
func (clock Clock) frameTime(index int) time.Duration {
	start, _ := clock.Start()
	span, stretched := clock.SceneDuration()
	if !stretched {
		offset := time.Duration(index) * time.Second /
			time.Duration(clock.rate)
		return start + offset
	}
	// divide before multiplying: a day of nanoseconds times a large
	// frame index would overflow, though the final offset fits.
	frames, frame := time.Duration(clock.Frames()), time.Duration(index)
	step, remainder := span/frames, span%frames
	return start + frame*step + frame*remainder/frames
}
