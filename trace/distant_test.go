package trace

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// a distant sun's shadow reaches any distance: a ball ten kilometres up
// the sun's way from a point on the ground shades it, and a point half
// a kilometre aside is lit.
func TestADistantSunsShadowReachesAnyDistance(t *testing.T) {
	sun, err := emit.NewDistant(maths.V(0, 1, 1), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Lamp(sun)
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 50},
		Motion: resting(maths.V(0, 7071, 7071))})
	scene := stage(world, 0)
	up := maths.V(0, 1, 0)
	shaded := scene.lamplight(maths.Vec{}, up, maths.Grey(1))
	aside := scene.lamplight(maths.V(500, 0, 0), up, maths.Grey(1))
	if shaded != (maths.Light{}) || aside.R <= 0 {
		t.Errorf("under the ball the light is %v, aside %v", shaded,
			aside)
	}
}
