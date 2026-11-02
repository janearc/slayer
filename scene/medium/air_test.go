package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Air is thickest at the surface, a factor of e thinner a scale height up, zero
// below the ground and past its reach, and spans a ray by its top.
func TestAir(t *testing.T) {
	var _ Spanner = Air{}
	a := Air{Radius: 10, Height: 0.5, Thick: 2}
	if d := a.Density(maths.V(10, 0, 0)); d != 2 {
		t.Errorf("at the surface, %v", d)
	}
	if d := a.Density(maths.V(10.5, 0, 0)); math.Abs(d-2/math.E) > 1e-12 {
		t.Errorf("a scale height up, %v", d)
	}
	if d := a.Density(maths.V(9, 0, 0)); d != 0 {
		t.Errorf("underground, %v", d)
	}
	if d := a.Density(maths.V(0, 10+Reach*0.5+0.01, 0)); d != 0 {
		t.Errorf("past the top, %v", d)
	}
	down := maths.Ray{At: maths.V(0, 20, 0), Dir: maths.V(0, -1, 0)}
	enter, _, ok := a.Span(down, 0, math.Inf(1))
	if !ok || math.Abs(enter-(20-13)) > 1e-12 {
		t.Errorf("the air's top is met at %v, %v", enter, ok)
	}
}
