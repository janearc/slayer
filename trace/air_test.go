package trace

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// a sky that is one colour in every direction
type even maths.Light

func (sky even) Light(maths.Vec) maths.Light { return maths.Light(sky) }

// glowing is a world of one floor that gives off light of 1 and needs no
// lamp, so what reaches the eye is only the floor and the air.
func glowing() *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)}})
	return world
}

// straightDown is a ray straight down from a height: the floor is
// that far away.
func straightDown(height float64) maths.Ray {
	return maths.Ray{At: maths.V(0, height, 0), Dir: maths.V(0, -1, 0)}
}

// Through haze of density k, a thing d away is exp(-k d) itself and the rest
// the haze.
func TestHazeThinsWithDistance(t *testing.T) {
	world := glowing()
	world.Haze = thing.Haze{Density: 0.1, Colour: maths.Grey(0.25)}
	for _, away := range []float64{1, 5, 20} {
		got := lone(world, straightDown(away), time.Duration(0)).R
		thinned := math.Exp(-0.1 * away)
		want := thinned + 0.25*(1-thinned)
		if !within(float64(got), want, 1e-6) {
			t.Errorf("%v away: %v, wanted %v", away, got, want)
		}
	}
}

// Haze from the sky fades the floor into the sky, and a ray that meets
// nothing is the sky, however thick the air.
func TestHazeFromTheSkyFadesIntoIt(t *testing.T) {
	world := glowing()
	world.Sky = even(maths.Grey(0.5))
	world.Haze = thing.Haze{Density: 0.1, FromSky: true}
	near := lone(world, straightDown(1), 0).R
	far := lone(world, straightDown(50), 0).R
	if !(near > far && far > 0.5) {
		t.Errorf("near %v, far %v: the far floor should be nearer 0.5",
			near, far)
	}
	up := lone(world, maths.Ray{At: maths.V(0, 1, 0),
		Dir: maths.V(0, 1, 0)}, 0)
	if !within(float64(up.R), 0.5, 1e-6) ||
		!math.IsInf(float64(up.Depth), 1) {
		t.Errorf("looking up: %v", up)
	}
}

// No sky and clear air is what the tracer has always done: nothing is black.
func TestNoSkyIsBlack(t *testing.T) {
	up := lone(glowing(), maths.Ray{At: maths.V(0, 1, 0),
		Dir: maths.V(0, 1, 0)}, 0)
	if up.R != 0 || up.G != 0 || up.B != 0 {
		t.Errorf("an empty sky gave %v", up)
	}
}
