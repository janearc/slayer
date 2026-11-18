package film

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A spot says where it stands, and the most light it sends a point
// a distance away: the brightest channel over the distance squared.
func TestSpotStandsAndAtMostFollowTheInverseSquare(t *testing.T) {
	spot := Spot{At: maths.V(1, 2, 3), Aim: maths.V(0, -1, 0),
		Strength: maths.Light{R: 2, G: 8, B: 4}, Cone: 1}
	if got := spot.Stands(); got != maths.V(1, 2, 3) {
		t.Errorf("the spot stands at %v, want 1 2 3", got)
	}
	for _, each := range []struct{ distance, most float64 }{
		{1, 8}, {2, 2}, {4, 0.5}, {0.5, 32},
	} {
		if got := spot.AtMost(each.distance); math.Abs(got-each.most) > 1e-12 {
			t.Errorf("at %v away the most light is %v, want %v",
				each.distance, got, each.most)
		}
	}
	if got := spot.AtMost(0); !math.IsInf(got, 1) {
		t.Errorf("at the spot itself the most light is %v, want no end", got)
	}
}
