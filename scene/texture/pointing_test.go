package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestPointingIsOneUnitLong: a point along one axis points one unit along
// it. Each length here is a power of two, so each want is exact: eight
// along minus x is one along minus x, a half along y is one along y.
func TestPointingIsOneUnitLong(t *testing.T) {
	for _, each := range []struct{ point, want maths.Vec }{
		{maths.V(0, 0, 2), maths.V(0, 0, 1)},
		{maths.V(-8, 0, 0), maths.V(-1, 0, 0)},
		{maths.V(0, 0.5, 0), maths.V(0, 1, 0)},
	} {
		if got := pointing(each.point); got != each.want {
			t.Errorf("%v points %v; want %v", each.point, got, each.want)
		}
	}
}

// TestPointingLeavesNoLengthAlone is its anti-test: a point of no length
// has no direction, and is given back as it is and not divided by nothing.
func TestPointingLeavesNoLengthAlone(t *testing.T) {
	if got := pointing(maths.Vec{}); got != (maths.Vec{}) {
		t.Errorf("no length points %v; want it left as it is", got)
	}
}
