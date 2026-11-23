package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestIndependentSurfaceEmissionPreservesClearTransmission(t *testing.T) {
	for _, test := range []struct {
		name        string
		through     float64
		independent bool
		want        float64
	}{
		{"opaque original", 0, false, 1},
		{"opaque independent", 0, true, 1},
		{"clear original", 1, false, 0},
		{"clear independent", 1, true, 1},
		{"half original", .5, false, .5},
		{"half independent", .5, true, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			world := thing.NewWorld()
			world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
				Glow: maths.Grey(1), Through: test.through, Index: 1,
				IndependentEmission: test.independent,
			}})
			from := maths.V(0, 2, 0)
			down := maths.Up.Neg()
			frame, err := Render(world, mediumEye(from, down), film.Pinhole{}, 0, 1, 1)
			if err != nil {
				t.Fatal(err)
			}
			if got := float64(frame.At(0, 0).R); math.Abs(got-test.want) > 1e-6 {
				t.Fatalf("emission %g, want %g", got, test.want)
			}
		})
	}
}
