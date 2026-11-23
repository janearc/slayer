package searching

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/named"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// ShotPlan inserts an existing scene or an explicitly labelled missing shot.
// From and Until are song cues; scene time and camera movement are authored.
type ShotPlan struct {
	Stage     *stagePlan     `yaml:"stage,omitempty"`
	From      string         `yaml:"from"`
	Until     string         `yaml:"until"`
	Scene     string         `yaml:"scene"`
	With      map[string]any `yaml:"with,omitempty"`
	Caption   string         `yaml:"caption"`
	Wide      float64        `yaml:"wide-degrees"`
	TimeScale float64        `yaml:"time-scale"`
	Orbit     float64        `yaml:"orbit-degrees"`
}

// checkShots resolves the cut list and refuses overlaps before building sets.
func (edit *Edit) checkShots() error {
	edit.ShotSpans = nil
	if edit.Plan.FullCut && edit.Plan.FlowStudy != nil && edit.Plan.CatalogueStudy != nil &&
		(edit.CatalogueEnd > edit.FlowStart || edit.FlowAnswer >= edit.Coda) {
		return fmt.Errorf("catalogue, flow and coda overlap")
	}
	if len(edit.Plan.Shots) == 0 {
		return nil
	}
	if !edit.Plan.FullCut || len(edit.Plan.Shots) > 64 {
		return fmt.Errorf("shots require a bounded full cut")
	}
	previous := -1.0
	for _, shot := range edit.Plan.Shots {
		from, a := edit.cue(shot.From)
		until, b := edit.cue(shot.Until)
		if a != nil || b != nil || from < previous || until <= from || until > edit.Grid.Score.Duration ||
			shot.Caption == "" || !bounded(shot.TimeScale, 0, 100) || !bounded(shot.Orbit, -360, 360) {
			return fmt.Errorf("invalid or overlapping shot %q", shot.Caption)
		}
		if shot.Scene == "heard" {
			if shot.Stage == nil {
				return fmt.Errorf("heard shot needs a stage")
			}
			if err := shot.Stage.check(); err != nil {
				return err
			}
		} else if shot.Scene != "slate" {
			scene, err := named.Get(shot.Scene)
			if err != nil {
				return err
			}
			if !scene.Draws(named.GPU) || !bounded(shot.Wide, 10, 120) {
				return fmt.Errorf("shot needs a GPU scene and valid lens")
			}
			if err := scene.Check(shot.With); err != nil {
				return err
			}
		}
		if edit.Plan.CatalogueStudy != nil && from < edit.CatalogueEnd && until > edit.CatalogueStart ||
			edit.Plan.FlowStudy != nil && from < edit.Coda && until > edit.FlowStart {
			return fmt.Errorf("shot overlaps catalogue or flow study")
		}
		edit.ShotSpans = append(edit.ShotSpans, [2]float64{from, until})
		previous = until
	}
	// The v3 stage cut must never silently fall through to the historical
	// generic opening. Account for the catalogue interval explicitly.
	for _, shot := range edit.Plan.Shots {
		if shot.Stage != nil {
			for frame := 0; frame < edit.Frames(); frame++ {
				at := float64(frame) / float64(edit.Plan.FPS)
				if edit.shot(at) < 0 && !(edit.Plan.CatalogueStudy != nil && at >= edit.CatalogueStart && at < edit.CatalogueEnd) {
					return fmt.Errorf("heard cut has uncovered frame %d", frame)
				}
			}
			break
		}
	}
	return nil
}

// shot selects a half-open interval so a cut belongs to its next shot.
func (edit *Edit) shot(at float64) int {
	for index, span := range edit.ShotSpans {
		if at >= span[0] && at < span[1] {
			return index
		}
	}
	return -1
}

// ShotAt exposes the same cut used by Frame to the companion and shot list.
func (edit *Edit) ShotAt(at float64) *ShotPlan {
	index := edit.shot(at)
	if index < 0 {
		return nil
	}
	return &edit.Plan.Shots[index]
}

// namedSet retains one built scene, its camera and original development.
type namedSet struct {
	built *named.Built
	scene *named.Scene
}

// keepOnly releases GPU resources at full-cut transitions. Distant sets do
// not accumulate in memory while the camera is somewhere else.
func (renderer *Renderer) keepOnly(site string) {
	if !renderer.Edit.Plan.FullCut || renderer.active == site {
		return
	}
	renderer.Close()
	renderer.sets = map[string]*set{}
	renderer.flow, renderer.stars, renderer.named = nil, nil, nil
	renderer.active = site
	runtime.GC()
}

// shotFrame draws a named scene at the same song instant as its companion.
func (renderer *Renderer) shotFrame(ctx context.Context, state State, index int) (*Exposure, State, error) {
	if err := ctx.Err(); err != nil {
		return nil, state, err
	}
	plan := renderer.Edit.Plan
	shot := plan.Shots[index]
	span := renderer.Edit.ShotSpans[index]
	state.Site, state.Chapter, state.Spectrum = shot.Scene, shot.Caption, "authored scene"
	state.SceneSeconds = (state.At - span[0]) * shot.TimeScale
	if shot.Scene == "heard" {
		return renderer.stageFrame(ctx, state, index)
	}
	if shot.Scene == "slate" {
		picture := image.NewRGBA(image.Rect(0, 0, plan.Width, plan.Height))
		for pixel := 3; pixel < len(picture.Pix); pixel += 4 {
			picture.Pix[pixel] = 255
		}
		label := font.Drawer{Dst: picture, Src: image.NewUniform(color.RGBA{220, 211, 192, 255}), Face: basicfont.Face7x13,
			Dot: fixed.P((plan.Width-len(shot.Caption)*7)/2, plan.Height/2)}
		label.DrawString(shot.Caption)
		return &Exposure{picture: picture}, state, nil
	}
	if renderer.named == nil {
		scene, err := named.Get(shot.Scene)
		if err != nil {
			return nil, state, err
		}
		built, err := scene.Build(shot.With, nil)
		if err != nil {
			return nil, state, err
		}
		renderer.named = &namedSet{built: built, scene: scene}
	}
	drawer := renderer.drawers[shot.Scene]
	if drawer == nil {
		var err error
		drawer, err = gpu.NewFrames(gpu.FasterLighting)
		if err != nil {
			return nil, state, err
		}
		renderer.drawers[shot.Scene] = drawer
	}
	at := time.Duration(state.SceneSeconds * float64(time.Second))
	renderer.named.built.Set(at)
	eye := renderer.named.built.Eye()
	turn := maths.About(maths.YAxis, shot.Orbit*maths.Degree*smooth((state.At-span[0])/(span[1]-span[0])))
	eye.Origin = turn.Out(eye.Origin)
	eye.Orientation = maths.LookAt(eye.Origin, maths.Vec{})
	lens, err := film.PinholeOf(shot.Wide * maths.Degree)
	if err != nil {
		return nil, state, err
	}
	frame, _, err := drawer.Draw(renderer.named.built.World(), uint64(math.Floor(state.At*float64(plan.FPS)))+1,
		eye, lens, at, plan.Width, plan.Height, plan.Rays)
	if err != nil {
		return nil, state, err
	}
	tone := renderer.named.scene.Defaults().Tone()
	if tone.Exposure == 0 {
		tone.Exposure = plan.Exposure
	}
	tone.Bloom = plan.Bloom
	return &Exposure{linear: frame, tone: tone}, state, nil
}
