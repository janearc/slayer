package searching

import (
	"context"
	"fmt"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"time"
)

// StagePlan describes one v3 proof shot. Positions use local metres for
// ground scenes and explicitly authored display units for respondents.
type StagePlan struct {
	Identity   int                 `yaml:"identity,omitempty" json:"identity,omitempty"`
	Kind       string              `yaml:"kind"`
	Eye        [3]float64          `yaml:"eye"`
	EndEye     [3]float64          `yaml:"end-eye"`
	Look       [3]float64          `yaml:"look"`
	Wide       float64             `yaml:"wide"`
	Exposure   float64             `yaml:"exposure"`
	Aim        [2]float64          `yaml:"aim"`
	Elevation  [2]float64          `yaml:"elevation"`
	Sun        [2]float64          `yaml:"sun"`
	SkyHours   [2]float64          `yaml:"sky-hours"`
	Interior   bool                `yaml:"interior"`
	Terrain    string              `yaml:"terrain,omitempty"`
	Craft      string              `yaml:"craft,omitempty"`
	Starlight  bool                `yaml:"starlight,omitempty"`
	Catalogue  *CataloguePlan      `yaml:"catalogue,omitempty"`
	Burst      bool                `yaml:"burst,omitempty"`
	Plunge     bool                `yaml:"plunge,omitempty"`
	Anatomy    *AnatomyPlan        `yaml:"anatomy,omitempty" json:"anatomy,omitempty"`
	Population []AuthoredBeingPlan `yaml:"population,omitempty" json:"population,omitempty"`
	// ShareAnatomyPrototypes defaults to true for authored individual and
	// multitude stages. False builds separate but identical octahedron and
	// sail meshes; materials, placements and motion remain per component.
	ShareAnatomyPrototypes *bool `yaml:"share-anatomy-prototypes,omitempty" json:"share-anatomy-prototypes,omitempty"`
}

// check rejects a plan with an unknown kind or out-of-range values.
func (p *StagePlan) check() error {
	switch p.Kind {
	case "individual", "authored-multitude", "owl", "kecks", "rubin", "subaru", "elt", "griffith", "compact-radio", "spread-radio", "zappy", "fold", "lantern", "occulter", "conversation", "accumulate", "together", "withdraw", "space", "milky-way", "galaxies", "laniakea", "huge-fold", "answer", "consequence", "listening", "continuous-field", "multitude":
	default:
		return fmt.Errorf("unknown heard stage %q", p.Kind)
	}
	if p.Identity < 0 || p.Identity > 1000000 {
		return fmt.Errorf("invalid individual identity")
	}
	if p.Kind == "space" && p.Craft != "" && p.Craft != "hubble" && p.Craft != "webb" && p.Craft != "spitzer" {
		return fmt.Errorf("unknown isolated spacecraft")
	}
	if p.Kind == "continuous-field" && p.Catalogue == nil {
		return fmt.Errorf("continuous field needs pinned nearby catalogue")
	}
	if p.ShareAnatomyPrototypes != nil {
		if p.Kind != "authored-multitude" && !(p.Kind == "individual" && p.Anatomy != nil) {
			return fmt.Errorf("share-anatomy-prototypes is only valid for authored individual or authored-multitude stages")
		}
	}
	if p.Anatomy != nil {
		if p.Kind != "individual" {
			return fmt.Errorf("authored anatomy is only valid for individual stages")
		}
		if err := p.Anatomy.check(); err != nil {
			return err
		}
	}
	if p.Kind == "authored-multitude" {
		if p.Anatomy != nil {
			return fmt.Errorf("authored multitude uses population members, not one anatomy")
		}
		if err := checkAuthoredPopulation(p.Population); err != nil {
			return err
		}
	} else if len(p.Population) != 0 {
		return fmt.Errorf("population is only valid for authored-multitude stages")
	}
	if !bounded(p.Wide, 10, 120) || !bounded(p.Exposure, .01, 100) {
		return fmt.Errorf("invalid stage lens or exposure")
	}
	for _, v := range append(append(p.Eye[:], p.EndEye[:]...), p.Look[:]...) {
		if !bounded(v, -1e6, 1e6) {
			return fmt.Errorf("invalid stage camera")
		}
	}
	for _, pair := range [][2]float64{p.Aim, p.Elevation, p.Sun, p.SkyHours} {
		for _, v := range pair {
			if !bounded(v, -360, 360) {
				return fmt.Errorf("invalid stage motion")
			}
		}
	}
	if p.Eye == p.Look || p.EndEye == p.Look {
		return fmt.Errorf("stage eye coincides with target")
	}
	return nil
}

type stageObject struct {
	volumes     []*thing.Volume
	volumeGlows []maths.Light
	glows       []maths.Light
	locals      []maths.Subspace
	things      []*thing.Thing
	at          maths.Vec
	pivot       maths.Vec
	role        int
	scale       float64
	motions     []thing.Motion
}
type heardStage struct {
	world        *thing.World
	objects      []stageObject
	layers       []stageLayer
	beams        []stageBeam
	ground       func(float64, float64) float64
	summitGround *summitSurface
	summit       bool
	marks        []*thing.Thing
	spectrum     [][]float64
	cosmos       *continuousField
	crowd        *multitude
	burstLamps   []int
}

// stageFrame draws one frame of a heard stage shot.
func (r *Renderer) stageFrame(ctx context.Context, state State, index int) (*Exposure, State, error) {
	p := r.Edit.Plan.Shots[index].Stage
	span := r.Edit.ShotSpans[index]
	u := math.Max(0, math.Min(1, (state.At-span[0])/(span[1]-span[0])))
	state.Site = p.Kind
	state.Aim = mix(p.Aim, smooth(math.Min(1, u/.62)))
	state.Camera = vector(p.Eye).Scale(1 - smooth(u)).Add(vector(p.EndEye).Scale(smooth(u))).Len()
	if p.Kind == "continuous-field" {
		state.Camera = fieldDistance(state.At)
	}
	if r.stage == nil {
		var err error
		r.stage, err = newHeardStage(p, r.Edit.Plan.Music)
		if err != nil {
			return nil, state, err
		}
	}
	if p.Kind == "listening" {
		return &Exposure{picture: r.stage.listening(r.Edit.Plan.Width, r.Edit.Plan.Height, state.At)}, state, nil
	}
	eye, err := r.stage.update(p, state, u)
	if err != nil {
		return nil, state, err
	}
	drawer := r.drawers["heard"]
	if drawer == nil {
		if len(r.stage.beams) > 0 {
			drawer, err = gpu.NewCompositeFrames(gpu.FasterLighting, gpu.WithoutPrimaryBeams)
		} else {
			drawer, err = gpu.NewFrames(gpu.FasterLighting)
		}
		if err != nil {
			return nil, state, err
		}
		r.drawers["heard"] = drawer
	}
	lens, err := film.PinholeOf(p.Wide * maths.Degree)
	if err != nil {
		return nil, state, err
	}
	if err = ctx.Err(); err != nil {
		return nil, state, err
	}
	frame, _, err := drawer.Draw(r.stage.world, uint64(state.At*float64(r.Edit.Plan.FPS))+1, eye, lens, time.Duration(state.At*float64(time.Second)), r.Edit.Plan.Width, r.Edit.Plan.Height, r.Edit.Plan.Rays)
	if err != nil {
		return nil, state, err
	}
	if len(r.stage.beams) > 0 {
		beams := r.drawers["heard-beams"]
		if beams == nil {
			beams, err = gpu.NewCompositeFrames(gpu.FasterLighting, gpu.PrimaryBeams)
			if err != nil {
				return nil, state, err
			}
			r.drawers["heard-beams"] = beams
		}
		light, _, drawErr := beams.Draw(r.stage.world, uint64(state.At*float64(r.Edit.Plan.FPS))+1, eye, lens, time.Duration(state.At*float64(time.Second)), r.Edit.Plan.Width, r.Edit.Plan.Height, r.Edit.Plan.Rays)
		if drawErr != nil {
			return nil, state, drawErr
		}
		frame, err = mergeBeamLight(frame, light)
		if err != nil {
			return nil, state, err
		}
	}
	exposure := p.Exposure
	if p.Kind == "consequence" {
		exposure *= math.Max(.000001, 1-smooth((state.At-(r.Edit.Grid.Score.Duration-3))/3))
	}
	if p.Kind == "continuous-field" || p.Kind == "multitude" {
		exposure *= math.Max(.000001, 1-smooth((state.At-(r.Edit.Grid.Score.Duration-3))/3))
	}
	return &Exposure{linear: frame, tone: colour.Tone{Exposure: exposure, Knee: .7, Top: 8, Bloom: r.Edit.Plan.Bloom}}, state, nil
}

// newHeardStage builds the scene for one heard stage plan.
func newHeardStage(p *StagePlan, music string) (*heardStage, error) {
	s := &heardStage{world: thing.NewWorld()}
	if p.Kind == "multitude" {
		return s, s.makeMultitude()
	}
	if p.Kind == "continuous-field" {
		return s, s.makeContinuousField(p)
	}
	switch p.Kind {
	case "listening":
		return s, s.readSpectrum(music)
	case "owl", "kecks", "rubin", "subaru", "elt", "griffith", "compact-radio", "spread-radio":
		return s, s.makeGround(p)
	default:
		return s, s.makeSpace(p)
	}
}

// update moves the stage to its state at progress u and returns the eye.
func (s *heardStage) update(p *StagePlan, state State, u float64) (maths.Subspace, error) {
	if s.cosmos != nil {
		return s.updateContinuousField(state.At, p.Plunge), nil
	}
	if s.crowd != nil {
		eye := s.updateMultitude(state.At)
		if p.Kind != "individual" && p.Kind != "authored-multitude" {
			return eye, nil
		}
	}
	from := vector(p.Eye).Scale(1 - smooth(u)).Add(vector(p.EndEye).Scale(smooth(u)))
	look := vector(p.Look)
	if s.ground != nil {
		if err := s.updateGround(p, state, u); err != nil {
			return maths.Subspace{}, err
		}
	} else {
		s.updateSpace(p, state, u)
	}
	return maths.NewSubspace(from, maths.LookAt(from, look), 1), nil
}

// mix returns the point at progress u between the two values of pair.
func mix(pair [2]float64, u float64) float64 { return pair[0] + (pair[1]-pair[0])*u }
