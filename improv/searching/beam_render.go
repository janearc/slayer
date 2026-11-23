package searching

import (
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// drawBeams retains a separate full-resolution primary beam drawer.
// Its scene, camera, time and sample grid match the complementary base.
func (renderer *Renderer) drawBeams(set *set, state State,
	eye maths.Subspace, lens film.Sensor) (*frame.Frame, error) {
	key := state.Site + "-primary-beams"
	drawer := renderer.drawers[key]
	if drawer == nil {
		var err error
		drawer, err = gpu.NewCompositeFrames(gpu.FasterLighting, gpu.PrimaryBeams)
		if err != nil {
			return nil, err
		}
		renderer.drawers[key] = drawer
	}
	plan := renderer.Edit.Plan
	light, _, err := drawer.Draw(set.World,
		uint64(math.Floor(state.At*float64(plan.FPS)))+1,
		eye, lens, time.Duration(state.At*4*float64(time.Second)),
		plan.Width, plan.Height, plan.Rays)
	return light, err
}
