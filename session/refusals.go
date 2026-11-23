package session

import "github.com/janearc/slayer/gpu"

// Refusals is everything in frame index's world, as it is staged at the
// job's exact time, that the GPU path would refuse to draw, each with
// where it is in the world and why. It draws nothing and does not stop
// at the first; it is empty when the GPU path would take the frame.
func (session *Session) Refusals(index int) ([]gpu.Refusal, error) {
	world, err := session.inspected(index)
	if err != nil {
		return nil, err
	}
	return gpu.Refusals(world, session.times[index]), nil
}
