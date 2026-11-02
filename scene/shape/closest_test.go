package shape

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// the nearest point on a face, worked by hand on the triangle with legs of
// four along x and y. just past A along each edge, by a quarter, the
// nearest is on that edge, a sixteenth of its way along; a unit out from
// either edge, a quarter of its way.
func TestTheNearestPointOnAnEdgeIsOnIt(t *testing.T) {
	a, b, c := maths.V(0, 0, 0), maths.V(4, 0, 0), maths.V(0, 4, 0)
	for _, want := range []struct {
		p maths.Vec
		w [3]float64
	}{
		{maths.V(0.25, -1, 0), [3]float64{0.9375, 0.0625, 0}},
		{maths.V(-1, 0.25, 0), [3]float64{0.9375, 0, 0.0625}},
		{maths.V(1, -1, 0), [3]float64{0.75, 0.25, 0}},
		{maths.V(-1, 1, 0), [3]float64{0.75, 0, 0.25}},
	} {
		if got := closest(want.p, a, b, c); got != want.w {
			t.Errorf("from %v the nearest is %v, want %v", want.p, got, want.w)
		}
	}
}
