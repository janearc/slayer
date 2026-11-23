package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The scene of a beam behind a slab: the eye's ray runs east along x,
// through a slab that takes light and gives none, from -30 m to -20 m,
// and then through a mist from -10 m to 10 m, where a beam stands up from
// a metre below the ray at x = 0.
const (
	// behindPeak is the beam's peak strength, in every channel: a
	// beam is far brighter than a thousand times white.
	behindPeak = 1e10
	// behindMist is the mist's density, per metre.
	behindMist = 0.002
	// behindHalf is half the mist's length, in metres.
	behindHalf = 10.0
	// behindSlabNear and behindSlabFar are where the slab stands on x.
	behindSlabNear, behindSlabFar = -30.0, -20.0
	// behindEyeAt is where the ray starts, short of the slab, on x.
	behindEyeAt = -40.0
	// behindFloor is the least light a display can show: one level of
	// eight bits of white, 1 / 255 = 0.0039.
	behindFloor = 1.0 / 255
	// behindTolerance is how far the unobstructed beam may be from its
	// closed form, as a share: the steps of 0.06 m in a mist of 0.002 are
	// 1.2e-4 each, a share of 1.2e-4 at most in all, and a thousandth is
	// eight times that.
	behindTolerance = 1e-3
)

// behindScene is the beam in the mist behind a slab of that optical
// depth, staged. The slab is white-less: its colour is nought, so it
// gives no scatter, only takes light.
func behindScene(test *testing.T, depth float64) *staged {
	test.Helper()
	beam, err := emit.NewBeam(emit.Lit{At: maths.V(0, -crossingAlong, 0),
		Toward: maths.V(0, 1, 0), Strength: maths.Grey(behindPeak)},
		measure.Metres(crossingRadius), measure.Metres(crossingReach),
		measure.Degrees(0))
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	world.Lamp(beam)
	world.AddVolume(thing.Volume{Medium: slab{
		density: depth / (behindSlabFar - behindSlabNear),
		box: maths.Box{
			Min: maths.V(behindSlabNear, -crossingHeight, -crossingHeight),
			Max: maths.V(behindSlabFar, crossingHeight, crossingHeight)}},
		Material: &thing.Material{Colour: maths.Light{}}})
	world.AddVolume(thing.Volume{Medium: slab{density: behindMist,
		box: maths.Box{
			Min: maths.V(-behindHalf, -crossingHeight, -crossingHeight),
			Max: maths.V(behindHalf, crossingHeight, crossingHeight)}},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	return stage(world, 0)
}

// behindRay is the eye's ray, from short of the slab, along x.
func behindRay() maths.Ray {
	return maths.Ray{At: maths.V(behindEyeAt, 0, 0), Dir: maths.V(1, 0, 0)}
}

// behindWanted is the light the beam gives with the slab out of the way,
// in closed form as crossingWanted works it, with the mist's density and
// length, and then kept by the slab: exp(-depth).
func behindWanted(depth float64) float64 {
	return crossingEven * behindPeak * behindMist *
		math.Exp(-behindMist*(crossingAlong+behindHalf)) *
		crossingRadius * math.Sqrt(math.Pi/2) * math.Exp(-depth)
}

// seen is whether light is enough to be seen.
func seen(light float64) bool { return light >= behindFloor }

// With no slab the beam in the mist gives its closed form, which is
// 0.5 * 1e10 * 0.002 * 0.313 * exp(-0.022) = 3.1e6, far over the floor.
func TestABeamInMistWithNoSlabGivesItsClosedForm(t *testing.T) {
	got := marchGathers(behindScene(t, 0), behindRay())
	want := behindWanted(0)
	if math.Abs(got/want-1) > behindTolerance {
		t.Errorf("the beam gave %g, want %g", got, want)
	}
	if !seen(got) {
		t.Errorf("the beam gave %g, under the floor %g", got, behindFloor)
	}
}

// Anti: behind a slab of optical depth 80 the beam draws nought, to the
// floor: it is 3.1e6 * exp(-80) = 5.5e-29 of white, which no display
// shows, so the check that the beam is seen reports it unseen.
func TestABeamBehindAnOpaqueSlabIsNotSeen(t *testing.T) {
	got := marchGathers(behindScene(t, 80), behindRay())
	if seen(got) {
		t.Errorf("a beam behind a slab of depth 80 gave %g", got)
	}
}
