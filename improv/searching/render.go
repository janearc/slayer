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
	sets    map[string]*set
	drawers map[string]*gpu.Frames
	flow    *flowSet
	stars   *catalogueSet
	named   *namedSet
	active  string
	stage   *heardStage
}

// NewRenderer takes no GPU resources until an actual frame is requested.
func NewRenderer(edit *Edit) *Renderer {
	return &Renderer{Edit: edit, sets: map[string]*set{}, drawers: map[string]*gpu.Frames{}}
}

// Close releases every retained drawer. A preview cannot leave native
// resources attached to a package-level singleton between runs.
func (renderer *Renderer) Close() {
	for _, drawer := range renderer.drawers {
		drawer.Close()
	}
	renderer.drawers = map[string]*gpu.Frames{}
	renderer.sets = map[string]*set{}
	renderer.flow, renderer.stars, renderer.named = nil, nil, nil
	renderer.stage = nil
	renderer.active = ""
}

// Frame draws a real scene at a song instant. Site override is for the
// representative stills only; an empty override obeys the checked edit.
func (renderer *Renderer) Frame(ctx context.Context, at float64,
	override string) (*image.RGBA, State, error) {
	exposure, state, err := renderer.Expose(ctx, at, override)
	if err != nil {
		return nil, state, err
	}
	return exposure.Develop(), state, nil
}

// Expose traces one frame, leaving display work to its consumer. Calls on
// a renderer remain sequential: its next scene update must follow the last
// trace, but need not wait for that exposure's development or file output.
func (renderer *Renderer) Expose(ctx context.Context, at float64,
	override string) (*Exposure, State, error) {
	if err := ctx.Err(); err != nil {
		return nil, State{}, err
	}
	state := renderer.Edit.At(at)
	if renderer.Edit.Plan.FullCut && override == "" && at < renderer.Edit.Grid.Score.Beats.Times[0] {
		picture := image.NewRGBA(image.Rect(0, 0, renderer.Edit.Plan.Width, renderer.Edit.Plan.Height))
		for pixel := 3; pixel < len(picture.Pix); pixel += 4 {
			picture.Pix[pixel] = 255
		}
		state.Site, state.Chapter = "slate", "before heard."
		return &Exposure{picture: picture}, state, nil
	}
	if override != "" {
		state.Site = override
	}
	if renderer.Edit.Plan.FullCut && override == "" {
		if shot := renderer.Edit.shot(at); shot >= 0 {
			renderer.keepOnly(fmt.Sprintf("shot-%d", shot))
			return renderer.shotFrame(ctx, state, shot)
		}
	}
	if renderer.Edit.Plan.CatalogueStudy != nil && override == "" &&
		(!renderer.Edit.Plan.FullCut || at >= renderer.Edit.CatalogueStart && at < renderer.Edit.CatalogueEnd) {
		renderer.keepOnly("catalogue")
		return renderer.catalogueFrame(ctx, state)
	}
	if renderer.Edit.FlowAt(at) && override == "" {
		renderer.keepOnly("flow")
		return renderer.flowFrame(ctx, state)
	}
	if renderer.Edit.Plan.FullCut {
		renderer.keepOnly(state.Site)
	}
	if state.Site != "hawaii" && state.Site != "atacama" &&
		state.Site != "radio-hawaii" && state.Site != "radio-atacama" &&
		state.Site != "blazar" {
		return nil, state, fmt.Errorf("unknown shoot site %q", state.Site)
	}
	set, ok := renderer.sets[state.Site]
	if !ok {
		var err error
		set, err = newSet(renderer.Edit.Plan, state.Site)
		if err != nil {
			return nil, state, err
		}
		renderer.sets[state.Site] = set
	}
	drawer, ok := renderer.drawers[state.Site]
	if !ok {
		var err error
		if len(set.Lasers) > 0 {
			drawer, err = gpu.NewCompositeFrames(gpu.FasterLighting, gpu.WithoutPrimaryBeams)
		} else {
			drawer, err = gpu.NewFrames(gpu.FasterLighting)
		}
		if err != nil {
			return nil, state, err
		}
		renderer.drawers[state.Site] = drawer
	}
	plan := renderer.Edit.Plan
	eye, wide, err := set.update(plan, state)
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
	if len(set.Lasers) > 0 {
		beams, err := renderer.drawBeams(set, state, eye, lens)
		if err != nil {
			return nil, state, err
		}
		frame, err = mergeBeamLight(frame, beams)
		if err != nil {
			return nil, state, err
		}
	}
	return &Exposure{linear: frame,
		tone:   colour.Tone{Exposure: plan.Exposure, Knee: 0.5, Top: 12, Bloom: plan.Bloom},
		graded: true, saturation: state.Saturation}, state, nil
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
