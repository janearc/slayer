package shape

import "testing"

func TestCoincidentEdgeHasNoOwnership(t *testing.T) {
	for _, point := range [][2]float64{{0, 0}, {3, -7}} {
		if side(0, point[0], point[1], point[0], point[1]) != 0 {
			t.Fatalf("zero-length edge at %v acquired a side", point)
		}
	}
}
