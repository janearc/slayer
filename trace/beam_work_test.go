package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// workHalf is half the length of the fog the work is counted across, in
// metres, and workDensity its density a metre: thin, so that the march is
// never cut short for want of light.
const (
	workHalf    = 10.0
	workDensity = 0.01
)

// askedLamp is an ordinary lamp that counts how often it is asked for its
// light. It shines from straight above, as far as the sun, so that each
// asking that finds light casts a shadow ray.
type askedLamp struct{ asked *int }

// Light is the lamp's one light, and one more asking.
func (lamp askedLamp) Light(_, _ maths.Vec) (maths.Vec, float64,
	maths.Light) {
	*lamp.asked++
	return maths.V(0, 1, 0), math.Inf(1), maths.Grey(1)
}

// work is what one march across the fog costs: how often the ordinary
// lamps were asked in all, and how often the fog's density was asked, by
// the march and by every shadow ray.
type work struct{ lamps, density int }

// workOf is the work of the eye's ray along x through the fog, with so
// many ordinary lamps and, when a beam is given, that beam among them.
func workOf(test *testing.T, ordinary int, beam *emit.Beam) work {
	test.Helper()
	var spent work
	world := thing.NewWorld()
	for range ordinary {
		world.Lamp(askedLamp{asked: &spent.lamps})
	}
	if beam != nil {
		world.Lamp(beam)
	}
	world.AddVolume(thing.Volume{Medium: counted{asked: &spent.density,
		slab: slab{density: workDensity, box: maths.Box{
			Min: maths.V(-workHalf, -crossingHeight, -crossingHeight),
			Max: maths.V(workHalf, crossingHeight, crossingHeight)}}},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	scene := stage(world, 0)
	spent = work{}
	through(scene, crossingRay(workHalf, 0), math.Inf(1))
	return spent
}

// One ordinary lamp, with no beam anywhere, is asked once at each of the
// 48 steps that a crossing is cut into, and the fog is asked at least
// as often as that.
func TestAMarchAsksAnOrdinaryLampOnceAStep(t *testing.T) {
	spent := workOf(t, 1, nil)
	if spent.lamps != acrossCuts {
		t.Errorf("the lamp was asked %d times, want %d", spent.lamps,
			acrossCuts)
	}
	if spent.density < acrossCuts {
		t.Errorf("the fog was asked %d times, want %d or more",
			spent.density, acrossCuts)
	}
}

// Anti: a second ordinary lamp doubles the lamps' askings, 96 against 48,
// and the comparison of work reports it as more.
func TestASecondOrdinaryLampDoublesTheWork(t *testing.T) {
	one, two := workOf(t, 1, nil), workOf(t, 2, nil)
	if two.lamps != 2*one.lamps {
		t.Errorf("two lamps were asked %d times, one lamp %d", two.lamps,
			one.lamps)
	}
	if two == one {
		t.Error("a second lamp left the work unchanged")
	}
}
