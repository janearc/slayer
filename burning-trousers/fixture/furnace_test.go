package fixture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// The white furnace: a white ball in a sky of one radiance everywhere.
// Every point of a convex ball sees half the sky, so a matte white
// ball gives back exactly the sky's radiance and vanishes into it, with
// no bounce between surfaces needed. The rays look at the ball's
// middle, near its edge, and past it at the sky.
func TestAWhiteBallVanishesIntoAWhiteSky(t *testing.T) {
	ball := &thing.Thing{Shape: fake.NewSphere(maths.V(0, 0, 0), 1, nil),
		Material: &thing.Material{Colour: maths.Grey(1)}}
	world := &thing.World{Things: []*thing.Thing{ball},
		Sky: fake.NewSky(maths.Grey(1))}
	eye := maths.V(0, 0, 5)
	looks := map[string]maths.Vec{"the middle": maths.V(0, 0, 0),
		"near the edge": maths.V(0.9, 0, 0), "the sky": maths.V(3, 0, 0)}
	var names []string
	var rays []maths.Ray
	for name, at := range looks {
		names = append(names, name)
		rays = append(rays, maths.Ray{At: eye, Dir: at.Sub(eye).Unit()})
	}
	picture, err := trace.Render(world, maths.Subspace{},
		fake.NewAimed(rays...), 0, len(rays), 1)
	if err != nil {
		t.Fatal(err)
	}
	for x, name := range names {
		if got := float64(picture.At(x, 0).R); math.Abs(got-1) > 1e-3 {
			t.Errorf("%s is %.4f in a sky of 1", name, got)
		}
	}
}
