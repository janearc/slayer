package searching

import (
	"context"
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
)

// flowFrame draws the opt-in reveal study on Metal through the same guarded
// output command and original-song clock as the observatory proof.
func (renderer *Renderer) flowFrame(ctx context.Context, state State) (
	*Exposure, State, error) {
	if err := ctx.Err(); err != nil {
		return nil, state, err
	}
	state.Site, state.Chapter = "flow-study", "authored flow study"
	state.Spectrum = "authored visible emission"
	state.Camera, state.Aim, state.Flash, state.Saturation = 0, 0, 0, 1
	state.SceneSeconds = state.At
	if state.At >= renderer.Edit.FlowFold {
		state.Chapter = "fold"
	}
	if state.At >= renderer.Edit.FlowAnswer {
		state.Chapter = "answer"
	}
	if renderer.flow == nil {
		set, err := newFlow(renderer.Edit.Plan.FlowStudy)
		if err != nil {
			return nil, state, err
		}
		renderer.flow = set
	}
	drawer := renderer.drawers[state.Site]
	if drawer == nil {
		var err error
		drawer, err = gpu.NewFrames(gpu.Software)
		if err != nil {
			return nil, state, err
		}
		renderer.drawers[state.Site] = drawer
	}
	plan := renderer.Edit.Plan
	renderer.flow.update(renderer.Edit, state.Beat)
	eye, aim := renderer.Edit.flowCamera(state.At)
	lens, err := film.PinholeOf(plan.FlowStudy.Wide * maths.Degree)
	if err != nil {
		return nil, state, err
	}
	frame, _, err := drawer.Draw(renderer.flow.world,
		uint64(math.Floor(state.At*float64(plan.FPS)))+1,
		maths.Subspace{Origin: eye, Orientation: maths.LookAt(eye, aim)},
		lens, time.Duration(state.At*float64(time.Second)),
		plan.Width, plan.Height, plan.Rays)
	if err != nil {
		return nil, state, err
	}
	return &Exposure{linear: frame, tone: colour.Tone{Exposure: plan.Exposure,
		Knee: 0.5, Top: 12, Bloom: plan.Bloom}}, state, nil
}
