package rings

import (
	"math"
	"testing"
)

// TestARayThatNeverSettlesIsGivenUpAsCaptured checks that a ray that neither
// leaves nor comes from the hole is stopped after its allowance of steps and
// taken as captured. An angle that is not a number gives such a ray: its
// position is never a number, so no test of it ends the march.
func TestARayThatNeverSettlesIsGivenUpAsCaptured(t *testing.T) {
	path := smokeLens(t).Follow(math.NaN())
	if !path.captured {
		t.Error("the stuck ray was not taken as captured")
	}
	if got := len(path.Points()); got != steps+1 {
		t.Errorf("the ray took %d points, want the eye and %d steps", got, steps)
	}
}
