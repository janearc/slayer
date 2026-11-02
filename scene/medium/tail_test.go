package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// A tail streams up from its head, dense at the head and thinning to
// the end, and a curved one bends toward x as it goes.
func TestTailStreamsAndBends(t *testing.T) {
	straight := Tail{Length: 10, Base: 1, Spread: 0.1, Thick: 1}
	if d := straight.Density(maths.V(0, 0.1, 0)); d < 0.98 {
		t.Errorf("the head is %v", d)
	}
	if d := straight.Density(maths.V(0, 9.9, 0)); d > 0.02 {
		t.Errorf("the end is %v", d)
	}
	if straight.Density(maths.V(0, -0.1, 0)) != 0 || straight.Density(maths.V(0, 11, 0)) != 0 {
		t.Error("gas beyond the tail")
	}
	bent := straight
	bent.Curve = 0.3
	// at the end the middle has moved three units toward x
	if on, off := bent.Density(maths.V(3*0.98, 9.8, 0)), bent.Density(maths.V(0, 9.8, 0)); !(on > 2*off) {
		t.Errorf("at the end the bent tail is %v on its bend and %v on the axis", on, off)
	}
	enter, leave, ok := bent.Span(maths.Ray{At: maths.V(-20, 5, 0), Dir: maths.XAxis}, 0, 40)
	if !ok || enter > 20 || leave < 20 || leave-enter > 8 {
		t.Errorf("the span across the middle is %v to %v, %v", enter, leave, ok)
	}
}
