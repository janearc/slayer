package searching

import (
	"context"
	"fmt"
	"image"
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/render/colour"
)

// Renderer retains one scene and GPU drawer per set. Camera changes never
// rebuild the manufactory catalogue, and arrays share antenna prototypes.
type Renderer struct {
	Edit    *Edit
	sets    map[string]*Set
	drawers map[string]*gpu.Frames
}

// NewRenderer takes no GPU resources until an actual frame is requested.
func NewRenderer(edit *Edit) *Renderer {
	return &Renderer{Edit: edit, sets: map[string]*Set{}, drawers: map[string]*gpu.Frames{}}
}

// Close releases every retained drawer. A preview cannot leave native
// resources attached to a package-level singleton between runs.
func (renderer *Renderer) Close() {
	for _, drawer := range renderer.drawers {
		drawer.Close()
	}
	renderer.drawers = map[string]*gpu.Frames{}
}

// Frame draws a real scene at a song instant. Site override is for the
// representative stills only; an empty override obeys the checked edit.
func (renderer *Renderer) Frame(ctx context.Context, at float64,
	override string) (*image.RGBA, State, error) {
	if err := ctx.Err(); err != nil {
		return nil, State{}, err
	}
	state := renderer.Edit.At(at)
	if override != "" {
		state.Site = override
	}
	if state.Site != "hawaii" && state.Site != "atacama" &&
		state.Site != "radio-hawaii" && state.Site != "radio-atacama" &&
		state.Site != "blazar" {
		return nil, state, fmt.Errorf("unknown shoot site %q", state.Site)
	}
	set, ok := renderer.sets[state.Site]
	if !ok {
		var err error
		set, err = NewSet(renderer.Edit.Plan, state.Site)
		if err != nil {
			return nil, state, err
		}
		renderer.sets[state.Site] = set
	}
	drawer, ok := renderer.drawers[state.Site]
	if !ok {
		var err error
		drawer, err = gpu.NewFrames(gpu.FasterLighting)
		if err != nil {
			return nil, state, err
		}
		renderer.drawers[state.Site] = drawer
	}
	plan := renderer.Edit.Plan
	eye, wide, err := set.Update(plan, state)
	if err != nil {
		return nil, state, err
	}
	lens, err := film.PinholeOf(wide * math.Pi / 180)
	if err != nil {
		return nil, state, err
	}
	frame, _, err := drawer.Draw(set.World, uint64(math.Floor(at*float64(plan.FPS)))+1,
		eye, lens, time.Duration(at*4*float64(time.Second)), plan.Width, plan.Height, plan.Rays)
	if err != nil {
		return nil, state, err
	}
	picture := colour.Tone{Exposure: plan.Exposure, Knee: 0.5, Top: 12, Bloom: plan.Bloom}.Develop(frame)
	grade(picture, state.Saturation)
	return picture, state, nil
}

// grade drains a little display saturation on beat four, as an edit choice.
// The blazar views separately change authored component emission before
// tracing; neither operation measures radiation in physical bands.
func grade(picture *image.RGBA, saturation float64) {
	for index := 0; index < len(picture.Pix); index += 4 {
		r, g, b := float64(picture.Pix[index]), float64(picture.Pix[index+1]), float64(picture.Pix[index+2])
		grey := (r + g + b) / 3
		for channel, value := range []float64{r, g, b} {
			picture.Pix[index+channel] = uint8(math.Max(0, math.Min(255, grey+(value-grey)*saturation)))
		}
	}
}
