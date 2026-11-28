package framing

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/camera"
)

// Path fits the subject at two or more strictly increasing scene times and
// joins the samples with linear camera moves. The path's time starts at zero
// at the first sample. Its up stays fixed, and intermediate poses are not fit.
func (tracking *tracker) Path(times []time.Duration) (*camera.Path, error) {
	if len(times) < 2 {
		return nil, fmt.Errorf("framing path: need at least two scene times")
	}
	first, err := tracking.At(times[0])
	if err != nil {
		return nil, err
	}
	previous := first
	moves := make([]camera.Move, 0, len(times)-1)
	for index, at := range times[1:] {
		between := at - times[index]
		if at <= times[index] || between <= 0 {
			return nil, fmt.Errorf("framing path: scene time %s must follow %s within the duration range", at, times[index])
		}
		next, err := tracking.At(at)
		if err != nil {
			return nil, err
		}
		if next.up.Sub(first.up).Len() > orientationTolerance {
			return nil, fmt.Errorf("framing path: camera up changes at scene time %s", at)
		}
		move, err := camera.Timed(previous.eye.Origin, next.eye.Origin, between, "linear")
		if err == nil {
			move, err = move.Looking(previous.target, next.target)
		}
		if err != nil {
			return nil, fmt.Errorf("framing path: ending at scene time %s: %w", at, err)
		}
		moves = append(moves, move)
		previous = next
	}
	return camera.New(moves, first.target, first.up)
}
