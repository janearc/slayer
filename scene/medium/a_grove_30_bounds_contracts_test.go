package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestGroveIncludesItsPrescribedFloorAndCanopyTop(t *testing.T) {
	for _, row := range []struct{ base, height, terrain, y float64 }{
		{0, 1, 0, 1},
		// The prescribed floor clips a column whose terrain continues below it.
		{1, 2, -1, 0},
	} {
		field, err := NewGrove(GroveWeather{Extinction: .008, Depth: 8, Coverage: 1, GroundFloor: 0},
			[]Canopy{{At: maths.V(0, row.base, 0), Radius: 2, Height: row.height}},
			func(x, z float64) float64 { return row.terrain })
		if err != nil {
			t.Fatal(err)
		}
		// Both points are one metre above terrain. Smoothstep at 1/8 is 11/256.
		const want = .008 * 245.0 / 256
		if got := field.Density(maths.V(0, row.y, 0)); !(math.Abs(got-want) <= 1e-15) {
			t.Fatalf("field boundary at y=%g: density %.17g, want %.17g", row.y, got, want)
		}
	}
}
