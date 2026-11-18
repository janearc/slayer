package searching

import (
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"reflect"
	"testing"
)

func TestHeardOpeningIsIndependentOfFrameHistory(t *testing.T) {
	p := flowFixture().Plan.FlowStudy
	f, err := newFlow(p)
	if err != nil {
		t.Fatal(err)
	}
	c := &continuousField{reference: f, plan: p, centre: maths.V(-10e6, 1e6, -6e6), fixed: map[*thing.Thing]maths.Subspace{}}
	for _, part := range f.world.Things {
		c.fixed[part] = part.At(0)
	}
	s := &heardStage{world: f.world, cosmos: c}
	s.updateContinuousField(220, false)
	want := []maths.Subspace{}
	for _, part := range f.world.Things {
		want = append(want, part.At(0))
	}
	for _, at := range []float64{220, 234, 203, 220} {
		s.updateContinuousField(at, false)
	}
	for i, part := range f.world.Things {
		if !reflect.DeepEqual(part.At(0), want[i]) {
			t.Fatal("frame history moved reference member", i, part.Material.Name)
		}
	}
}

func TestHeardSkyHasNoArtificialHorizontalHorizon(t *testing.T) {
	s := heardSky{}
	a := s.Light(maths.V(1, 1e-7, 0)).Luminance()
	b := s.Light(maths.V(1, -1e-7, 0)).Luminance()
	// Permit point-star tile quantization, well below the old .01-scale
	// discontinuity caused by the separate lower-hemisphere colour.
	if math.Abs(a-b) > 1e-4 {
		t.Fatal("sky cuts off above the actual terrain", a, b)
	}
}

func TestHeardReviewPlanAllows2KButBoundsWork(t *testing.T) {
	p := goodPlan()
	p.Width, p.Height, p.Rays = 2048, 1152, 4
	if err := p.check(); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"width", "height", "rays"} {
		bad := p
		switch field {
		case "width":
			bad.Width++
		case "height":
			bad.Height++
		case "rays":
			bad.Rays++
		}
		if bad.check() == nil {
			t.Fatal("unbounded review work", field)
		}
	}
}

func TestHeardLaunchersFollowTheirActualOpticalAssembly(t *testing.T) {
	for _, name := range []string{"keck-i", "subaru", "elt"} {
		rig, err := manufacture.MakeGroundRigWithLaunchers(name, "demo")
		if err != nil {
			t.Fatal(err)
		}
		s := &heardStage{world: thing.NewWorld()}
		s.world.Lamp(thing.Sun(maths.YAxis, maths.Grey(1)))
		if err := s.groundRig(name, maths.V(13, 2, -7), &StagePlan{}); err != nil {
			t.Fatal(err)
		}
		if len(rig.Launchers) == 0 || len(s.beams) != len(rig.Launchers) {
			t.Fatal("missing attached hardware", name)
		}
		p := &StagePlan{Aim: [2]float64{-30, 40}, Elevation: [2]float64{-15, 20}}
		for _, u := range []float64{0, .2, .5, 1, .2} {
			if err := s.updateGround(p, State{At: 12}, u); err != nil {
				t.Fatal(err)
			}
			pose := s.objects[2].things[0].At(0)
			for i, exit := range rig.Launchers {
				got := s.beamPose(s.beams[i], p, u)
				if got.Origin.Sub(pose.Out(exit.Origin)).Len() > 1e-8 || got.Orientation.Out(maths.YAxis).Sub(pose.Orientation.Out(exit.Orientation.Out(maths.YAxis))).Len() > 1e-8 {
					t.Fatal("detached beam", name, u)
				}
			}
		}
	}
	rig, err := manufacture.MakeGroundRigWithLaunchers("rubin", "demo")
	if err != nil || len(rig.Launchers) != 0 {
		t.Fatal("survey camera acquired invented laser hardware", err)
	}
}

func TestHeardSpacecraftRotatesAroundItsBoundsCentre(t *testing.T) {
	for _, name := range []string{"hubble", "webb", "spitzer"} {
		s := &heardStage{world: thing.NewWorld()}
		if err := s.factory(name, maths.V(2, 3, 4), 1, 1); err != nil {
			t.Fatal(err)
		}
		obj := s.objects[0]
		if obj.pivot.Len() < .1 {
			t.Fatal("construction foot used as centre", name)
		}
		for _, at := range []float64{174, 176, 180, 185, 174} {
			s.updateSpace(&StagePlan{Kind: "space", Craft: name}, State{At: at}, .5)
			for _, part := range obj.things {
				if part.At(0).Out(obj.pivot).Sub(obj.at).Len() > 1e-8 {
					t.Fatal("centre moves during slew", name, at)
				}
			}
		}
	}
}

func TestHeardCrowdAcceleratesThenHoldsARealPopulation(t *testing.T) {
	s := &heardStage{world: thing.NewWorld()}
	if err := s.makeMultitude(); err != nil {
		t.Fatal(err)
	}
	if len(s.crowd.beings) < 3000 {
		t.Fatal("population remains a small montage")
	}
	previous := 0.
	for _, at := range []float64{280, 290, 300, 308} {
		speed := crowdTravel(at+.01) - crowdTravel(at)
		if speed <= previous {
			t.Fatal("passage does not accelerate", at, speed)
		}
		previous = speed
	}
	s.updateMultitude(320)
	visible := 0
	for _, being := range s.crowd.beings {
		if len(being.parts) == 0 {
			t.Fatal("empty identity")
		}
		if !being.parts[0].part.Hidden() {
			visible++
		}
	}
	if visible < 3000 {
		t.Fatal("population hidden", visible)
	}
	if math.Abs(crowdTravel(330)-crowdTravel(320)) > 1e-9 {
		t.Fatal("population drifts during held consequence")
	}
}
