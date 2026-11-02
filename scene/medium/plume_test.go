package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// A plume is dense at the column's foot, has its umbrella's dome
// halfway up at most of the width, is empty under the dome, and stops
// at its height.
func TestPlumeIsAColumnUnderAnUmbrella(t *testing.T) {
	p := Plume{Height: 10, Width: 20, Thick: 1}
	if d := p.Density(maths.V(0, 1, 0)); d < 0.9 {
		t.Errorf("the column's foot is %v", d)
	}
	if d := p.Density(maths.V(8.66, 5, 0)); d < 0.3 {
		t.Errorf("the dome halfway up is %v", d)
	}
	if d := p.Density(maths.V(3, 5, 0)); d > 0.01 {
		t.Errorf("inside the umbrella is %v", d)
	}
	if p.Density(maths.V(0, 11, 0)) != 0 || p.Density(maths.V(0, -1, 0)) != 0 {
		t.Error("the plume goes beyond its height")
	}
	if b := p.Bounds(); b.Min != maths.V(-10, 0, -10) || b.Max != maths.V(10, 10, 10) {
		t.Errorf("bounds %v", b)
	}
}
