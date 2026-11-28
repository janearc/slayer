package workspace

import (
	"time"

	"github.com/janearc/slayer/internal/observability"
)

// workingTiming tracks the newest motion that can appear in the next pixels.
type workingTiming struct {
	motion, previousFrame time.Time
}

// workingClock avoids reading the clock when nobody consumes the metrics.
func workingClock() time.Time {
	if observability.Enabled() {
		return time.Now()
	}
	return time.Time{}
}

func (timing *workingTiming) moved(start time.Time) {
	if start.IsZero() {
		return
	}
	if !timing.motion.IsZero() {
		observability.Add("ferruccio_working_motion_coalesced_total", 1)
	}
	timing.motion = start
}

// pixelsReady records an accepted working frame and consumes its input once.
func (timing *workingTiming) pixelsReady(now time.Time) {
	if now.IsZero() || timing.motion.IsZero() {
		return
	}
	observability.Observe("ferruccio_working_motion_to_pixels_seconds", now.Sub(timing.motion).Seconds())
	if !timing.previousFrame.IsZero() {
		observability.Observe("ferruccio_working_motion_frame_interval_seconds", now.Sub(timing.previousFrame).Seconds())
	}
	timing.motion, timing.previousFrame = time.Time{}, now
}
