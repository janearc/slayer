package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/scene/thing"
)

// The beam and its fog in these tests: a column of a quarter of a metre
// in radius and no divergence, standing up from a metre below the ray,
// in a uniform fog the ray crosses square to the beam, along x.
const (
	// crossingSigma is the fog's density, per metre: its extinction, and
	// the scatter of each metre, as the fog's colour is white.
	crossingSigma = 2e-4
	// crossingRadius is the beam's launch radius, in metres.
	crossingRadius = 0.25
	// crossingReach is how far the beam runs, in metres.
	crossingReach = 3.0
	// crossingAlong is how far the ray is above the launch plane, in
	// metres: the beam starts at y = -1 and the ray runs at y = 0.
	crossingAlong = 1.0
	// crossingEven is the share of a lamp's light an even scatterer
	// sends toward the eye: the engine's convention, one half, stated
	// here as an input and not worked out.
	crossingEven = 0.5
	// crossingTolerance is how far the gathered light may be from the
	// closed form, as a share of it. With steps of 0.06 m the sum of a
	// Gaussian 0.25 m wide is exact to exp(-87). Each step takes
	// 1 - exp(-x) where x = sigma * 0.06 = 1.2e-5, which is x less half
	// its square, and keeps the eye's light from its start, which is
	// off by half x: a share of 1.2e-5 at most in all. A thousandth is
	// eighty times that.
	crossingTolerance = 1e-3
	// crossingBefore is how far the eye stands before the fog, in metres.
	crossingBefore = 5.0
	// crossingHeight is how far the fog stands above and below the ray
	// and to each side of it, in metres.
	crossingHeight = 2.0
	// acrossCuts is how many steps the march cuts a crossing into
	// before any refinement.
	acrossCuts = 48
)

// crossingWorld is the beam, east of the middle of the fog by east
// metres and of that radius and peak, in a fog that runs half metres
// each side of the origin, staged.
func crossingWorld(test *testing.T, half, east, radius,
	peak float64) *staged {
	test.Helper()
	beam, err := emit.NewBeam(emit.Lit{At: maths.V(east, -crossingAlong, 0),
		Toward: maths.V(0, 1, 0), Strength: maths.Grey(peak)},
		measure.Metres(radius), measure.Metres(crossingReach),
		measure.Degrees(0))
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	world.Lamp(beam)
	world.AddVolume(thing.Volume{Medium: slab{density: crossingSigma,
		box: maths.Box{
			Min: maths.V(-half, -crossingHeight, -crossingHeight),
			Max: maths.V(half, crossingHeight, crossingHeight)}},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	return stage(world, 0)
}

// crossingRay is the eye's ray along x at the height of the ray and z
// metres to one side, from before the fog.
func crossingRay(half, aside float64) maths.Ray {
	return maths.Ray{At: maths.V(-half-crossingBefore, 0, aside),
		Dir: maths.V(1, 0, 0)}
}

// crossingWanted is the light a march gathers across the beam, in closed
// form. At a point of the ray s metres from the beam's axis the beam's
// light is peak * exp(-2 s^2 / radius^2), kept by the fog for the
// metre along the axis: exp(-sigma * along). The fog scatters even * that
// toward the eye, at sigma a metre, and the eye keeps exp(-sigma * d) of
// it, d being the way through the fog to the beam, half + east. Across
// the beam the Gaussian holds radius * sqrt(pi / 2) metres of its peak,
// so the sum is
// even * peak * sigma * exp(-sigma * (along + d)) * radius * sqrt(pi / 2).
func crossingWanted(half, east, radius, peak float64) float64 {
	return crossingEven * peak * crossingSigma *
		math.Exp(-crossingSigma*(crossingAlong+half+east)) *
		radius * math.Sqrt(math.Pi/2)
}

// gathersAs is whether the light a march gathered is the closed form's,
// to the tolerance.
func gathersAs(got, want float64) bool {
	return math.Abs(got/want-1) <= crossingTolerance
}

// marchGathers is the red a march adds along a ray in a staged scene.
func marchGathers(scene *staged, ray maths.Ray) float64 {
	return through(scene, ray, math.Inf(1)).added.R
}

// crossingShortHalf is half of a crossing short enough that refinement
// reaches the step of a quarter radius: 20 m in 48 cuts is 0.417 m, which
// the step of 0.25 / 4 = 0.0625 m cuts into 7, under the 64 allowed.
const crossingShortHalf = 10.0

// A ray square to a beam, across a crossing short enough that refinement
// reaches a quarter of the beam's radius, gathers the Gaussian's integral,
// with the switch unset and set, and with the beam on a cell's edge and
// not.
func TestARayAcrossAShortCrossingGathersTheGaussianIntegral(t *testing.T) {
	for _, value := range []string{"", "1"} {
		t.Setenv(scrooge.Variable, value)
		for _, east := range []float64{0, 0.8} {
			scene := crossingWorld(t, crossingShortHalf, east,
				crossingRadius, 1)
			got := marchGathers(scene, crossingRay(crossingShortHalf, 0))
			want := crossingWanted(crossingShortHalf, east,
				crossingRadius, 1)
			if !gathersAs(got, want) {
				t.Errorf("switch %q, east %g: gathered %g, want %g "+
					"(%.4g of it)", value, east, got, want, got/want)
			}
		}
	}
}

// Anti: a ray a metre to one side of the beam gathers nothing of it, so
// the comparison with the closed form reports the miss.
func TestARayBesideTheBeamGathersNoneOfTheGaussianIntegral(t *testing.T) {
	scene := crossingWorld(t, crossingShortHalf, 0, crossingRadius, 1)
	got := marchGathers(scene, crossingRay(crossingShortHalf, 1))
	want := crossingWanted(crossingShortHalf, 0, crossingRadius, 1)
	if gathersAs(got, want) {
		t.Errorf("a ray beside the beam gathered %g, the beam's %g", got,
			want)
	}
}
