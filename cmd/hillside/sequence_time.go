package main

import "time"

// frameTime maps playback frames to scene time. Omitting scene-duration keeps
// the original real-time clock. Camera travel and zoom use this scene clock.
func (plan sequencePlan) frameTime(index int) time.Duration {
	if plan.sceneSpan == 0 {
		offset := time.Duration(index) * time.Second / time.Duration(plan.fps)
		return plan.start + offset
	}
	// Divide before multiplying: a day of nanoseconds times a large frame
	// index would overflow, although the final offset fits in a duration.
	frames, frame := time.Duration(plan.frames), time.Duration(index)
	step, remainder := plan.sceneSpan/frames, plan.sceneSpan%frames
	return plan.start + frame*step + frame*remainder/frames
}
