package session

import (
	"fmt"
	"time"
)

// sequenceAbsent is defined as no selected numbered smoke.
const sequenceAbsent = 0

// sequenceMinimumFrames is defined as one actual frame for a nonempty clip.
const sequenceMinimumFrames = 1

// withCamera samples time relative to the sequence start. Scene time can start
// partway through a fire without starting the camera partway along its path.
func (plan sequencePlan) withCamera(given *asked,
	at time.Duration) error {
	elapsed := at - plan.start
	if plan.camera != nil {
		eye, err := plan.camera.At(elapsed)
		if err != nil {
			return fmt.Errorf("sequence: %w", err)
		}
		given.eye = eye
	}
	if plan.zoom != nil {
		wide, err := plan.zoom.At(elapsed)
		if err != nil {
			return fmt.Errorf("sequence: %w", err)
		}
		given.wide = wide
	}
	return nil
}
