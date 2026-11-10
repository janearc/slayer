package flame

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestFieldBoundsAreFiniteAndEncloseEachPlacedBody(t *testing.T) {
	large := campfire
	large.Radius, large.Height = .8, 2
	cells := []Ignition{{At: maths.V(-4, 0, 1), Fire: campfire}, {At: maths.V(3, 2, -5), Fire: large}}
	field, err := NewField(cells, 1)
	if err != nil {
		t.Fatal(err)
	}
	bounds := field.Bounds()
	for _, v := range []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z, bounds.Max.X, bounds.Max.Y, bounds.Max.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatal("composite bounds contain a nonfinite coordinate")
		}
	}
	if !(bounds.Max.X > bounds.Min.X && bounds.Max.Y > bounds.Min.Y && bounds.Max.Z > bounds.Min.Z) {
		t.Fatal("composite bounds are empty")
	}
	expected := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)), Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, cell := range cells {
		body, err := NewBody(cell.Fire, 1)
		if err != nil {
			t.Fatal(err)
		}
		box := body.Bounds()
		for _, x := range []float64{box.Min.X, box.Max.X} {
			for _, y := range []float64{box.Min.Y, box.Max.Y} {
				for _, z := range []float64{box.Min.Z, box.Max.Z} {
					p := maths.V(x, y, z).Add(cell.At)
					if p.X < bounds.Min.X || p.X > bounds.Max.X || p.Y < bounds.Min.Y || p.Y > bounds.Max.Y || p.Z < bounds.Min.Z || p.Z > bounds.Max.Z {
						t.Fatalf("placed body corner %v lies outside %v", p, bounds)
					}
					expected.Min = maths.V(math.Min(expected.Min.X, p.X), math.Min(expected.Min.Y, p.Y), math.Min(expected.Min.Z, p.Z))
					expected.Max = maths.V(math.Max(expected.Max.X, p.X), math.Max(expected.Max.Y, p.Y), math.Max(expected.Max.Z, p.Z))
				}
			}
		}
	}
	if bounds != expected {
		t.Fatalf("bounds retain space beyond all component corners: %v want %v", bounds, expected)
	}
}
