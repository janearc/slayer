package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestSphericalMediumSpansContainPositiveDistanceOnly(t *testing.T) {
	puff := Puff{Radius: 1, Thick: 1}
	direct := maths.Ray{At: maths.V(0, 0, 3), Dir: maths.ZAxis.Neg()}
	tangent := maths.Ray{At: maths.V(1, 0, 3), Dir: maths.ZAxis.Neg()}
	for _, row := range []struct {
		ray       maths.Ray
		near, far float64
	}{
		{tangent, 0, 10}, {direct, 2.5, 2.5}, {direct, 0, 2}, {direct, 4, 10},
	} {
		if _, _, ok := puff.Span(row.ray, row.near, row.far); ok {
			t.Fatalf("zero-length span accepted: %+v", row)
		}
	}
	if enter, leave, ok := puff.Span(direct, 2, 4); !ok || enter != 2 || leave != 4 {
		t.Fatalf("positive clipped span %g..%g, %t", enter, leave, ok)
	}
}
