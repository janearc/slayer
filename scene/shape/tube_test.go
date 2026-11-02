package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A tube is a shape; inside its wall the distance is less than zero, on the
// wall it is zero, and away from it, in or out, it grows.
func TestTubeDistance(t *testing.T) {
	var _ Shape = Tube{}
	tb := Tube{Radius: 2, Thick: 0.2, Height: 1}
	for _, c := range []struct {
		p    maths.Vec
		want float64
	}{
		{maths.V(2, 0, 0), -0.1},  // in the wall
		{maths.V(2.1, 0, 0), 0},   // on the outside
		{maths.V(0, 0, 1.9), 0},   // on the inside
		{maths.V(0, 0, 1), 0.9},   // inside the ring, in the air
		{maths.V(3, 0, 0), 0.9},   // outside the ring
		{maths.V(2, 1, 0), 0.5},   // above the rim
		{maths.V(0, 0, 0), 1.9},   // on the axis
		{maths.V(2.1, 0.5, 0), 0}, // the rim's corner
		{maths.V(2.1, 1.5, 0), 1}, // above the corner
	} {
		if got := tb.Distance(c.p); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("at %v: %v, want %v", c.p, got, c.want)
		}
	}
}

// The marched normal on the inside of the wall points toward the axis,
// and on the outside away from it.
func TestTubeNormalsFaceTheirSides(t *testing.T) {
	tb := Tube{Radius: 2, Thick: 0.2, Height: 1}
	in := Normal(tb, maths.V(0, 0, 1.9))
	out := Normal(tb, maths.V(0, 0, 2.1))
	if !in.Near(maths.V(0, 0, -1), 1e-4) || !out.Near(maths.V(0, 0, 1), 1e-4) {
		t.Fatalf("inside %v, outside %v", in, out)
	}
}
