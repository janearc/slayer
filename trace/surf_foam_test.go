package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestSurfFoamLeavesWaterWithNoFoamAlone(t *testing.T) {
	open, err := arid.NewSea(oceantex.Sea{Wind: 7, Fetch: 500000}, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	sum := maths.Light{R: 0.1, G: 0.2, B: 0.3}
	at := maths.V(1, 0, 1)
	for name, target := range map[string]*thing.Thing{
		"a sea with no surf":   {Shape: open},
		"a shape with no foam": {Shape: shape.Sphere{Radius: 1}},
	} {
		got := (&staged{}).surfFoam(sum, at, at, maths.V(0, 1, 0), look{},
			target)
		if got != sum {
			t.Errorf("%s: the light %v became %v", name, sum, got)
		}
	}
}

// frothy is a flat water that says the same foam everywhere: how much of
// its surface is foam, and what the foam reflects.
type frothy struct {
	shape.Plane
	coverage    float64
	reflectance maths.Light
}

// Foam is the one coverage and reflectance, wherever it is asked.
func (sea frothy) Foam(maths.Vec) (float64, maths.Light) {
	return sea.coverage, sea.reflectance
}

// foamWhite is what the foam of these tests reflects, a different share
// of each channel.
var foamWhite = maths.RGB(0.8, 0.6, 0.4)

// foamLit is a scene of a sun of two, 0.6 square to a level surface, and
// a sky of a half everywhere, with nothing in their way.
func foamLit() *staged {
	world := thing.NewWorld()
	along := maths.V(0, 0.6, 0.8)
	light := maths.Grey(2)
	world.Lamp(thing.Sun(along, light))
	world.Sky = even(maths.Grey(0.5))
	return stage(world, 0)
}

// foamed is the light surfFoam makes of the water's own, at a level
// point of a sea with so much foam on it.
func foamed(water maths.Light, coverage float64) maths.Light {
	sea := &thing.Thing{Shape: frothy{coverage: coverage,
		reflectance: foamWhite}}
	return foamLit().surfFoam(water, maths.Vec{}, maths.Vec{}, maths.Up,
		look{}, sea)
}

// Foam is laid over the water by its share of the surface: that share of
// the light is a matte surface of the foam's reflectance under the lamps
// and the sky, and the rest is the water's own.
//
// A matte surface sends its colour over pi times the sun's irradiance
// by the cosine it lands at, 2 times 0.6 over pi, and its colour times
// an even sky's radiance, a half: 1.2/pi + 0.5 = 0.8820 of the foam's
// reflectance. A quarter foam over water of 0.1, 0.2 and 0.3 is then
// 0.75 of the water and 0.25 of 0.8820 times 0.8, 0.6 and 0.4.
func TestSurfFoamLaysItsShareOfMatteLightOverTheWater(t *testing.T) {
	water := maths.RGB(0.1, 0.2, 0.3)
	matte := 1.2/math.Pi + 0.5
	want := water.Scale(0.75).Add(foamWhite.Scale(0.25 * matte))
	lightIs(t, "a quarter foam", foamed(water, 0.25), want, 1e-12)
}

// Anti: no foam leaves the water's light as it is, to the bit, and so
// does a coverage that is no number; and all foam leaves none of the
// water's, however bright the water was.
func TestSurfFoamOfNoCoverKeepsTheWaterAndOfAllCoverKeepsNone(t *testing.T) {
	water := maths.RGB(0.1, 0.2, 0.3)
	for _, none := range []float64{0, -0.5, math.NaN()} {
		if got := foamed(water, none); got != water {
			t.Errorf("foam over %v of the water made %v of %v", none, got,
				water)
		}
	}
	matte := 1.2/math.Pi + 0.5
	allFoam := foamed(maths.Grey(1000), 1)
	lightIs(t, "all foam over bright water", allFoam,
		foamWhite.Scale(matte), 1e-12)
}
