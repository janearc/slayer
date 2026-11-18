package geom

import (
	"math"
	"testing"
)

// TestNormalizeScalesToLengthOneAndLeavesZeroAlone checks (3, 0, 4) becomes (0.6, 0, 0.8), and the zero vector stays zero.
func TestNormalizeScalesToLengthOneAndLeavesZeroAlone(t *testing.T) {
	got := Vec3{3, 0, 4}.Normalize()
	if math.Abs(got[0]-0.6) > 1e-12 || got[1] != 0 || math.Abs(got[2]-0.8) > 1e-12 {
		t.Errorf("Normalize of (3, 0, 4) = %v", got)
	}
	if zero := (Vec3{}).Normalize(); zero != (Vec3{}) {
		t.Errorf("Normalize of zero = %v", zero)
	}
}
