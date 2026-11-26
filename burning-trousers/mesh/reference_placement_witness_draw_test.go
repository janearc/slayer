//go:build draw

package mesh_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestReferenceFramingEnclosesPlacedShapeCorners(t *testing.T) {
	things := []thing.Thing{
		{Shape: shape.Tube{Radius: .5, Thick: .5, Height: 2}, Motion: thing.Still(maths.Subspace{Origin: maths.V(8, 2, -3), Orientation: maths.About(maths.ZAxis, math.Pi/3), Scale: .25})},
		{Shape: shape.Sphere{Radius: 1}, Motion: thing.Still(maths.Subspace{Origin: maths.V(-4, 0, 3), Scale: 2})},
	}
	box := around(things)
	for _, item := range things {
		b := item.Shape.Bounds()
		for _, x := range []float64{b.Min.X, b.Max.X} {
			for _, y := range []float64{b.Min.Y, b.Max.Y} {
				for _, z := range []float64{b.Min.Z, b.Max.Z} {
					p := item.At(0).Out(maths.V(x, y, z))
					if p.X < box.Min.X || p.X > box.Max.X || p.Y < box.Min.Y || p.Y > box.Max.Y || p.Z < box.Min.Z || p.Z > box.Max.Z {
						t.Fatalf("placed corner %v lies outside frame %+v", p, box)
					}
				}
			}
		}
	}
}
