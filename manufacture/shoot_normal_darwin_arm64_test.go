//go:build darwin && arm64

package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The existing spruce normal's rounding order is part of its recorded mesh.
func TestShootNormalKeepsOriginalArm64Rounding(t *testing.T) {
	from := maths.V(math.Float64frombits(0x3fde98053c4c8ba1), math.Float64frombits(0x3fe0ac681eb0061a), math.Float64frombits(0xbfe6a09e667f3bcd))
	got := shootNormal(from)
	want := [3]uint64{0x3fde98053c4c8b9f, 0x3fe0ac681eb00619, 0xbfe6a09e667f3bcc}
	for i, v := range []float64{got.X, got.Y, got.Z} {
		if bits := math.Float64bits(v); bits != want[i] {
			t.Fatalf("component %d: got %016x, want %016x", i, bits, want[i])
		}
	}
	if got := shootNormal(maths.Vec{}); got != (maths.Vec{}) {
		t.Fatalf("zero: %+v", got)
	}
}

// This point exercises the normal in the foliage construction context, where
// inlining changed its evaluation order in the initial profile refactor.
func TestShootPlanKeepsOriginalArm64NeedleNormal(t *testing.T) {
	plan := planShoot(spruceShootStyle, .025, Sow(0))
	if len(plan.Needles) <= 63 {
		t.Fatal("missing reference needle")
	}
	needle := plan.Needles[63]
	values := []float64{needle.Normal.X, needle.Normal.Y, needle.Normal.Z,
		needle.Across.X, needle.Across.Y, needle.Across.Z}
	want := []uint64{0x3fde98053c4c8b9f, 0x3fe0ac681eb00619, 0xbfe6a09e667f3bcc,
		0x3feb63c052232051, 0xbfdd4069f13ccdff, 0x3fcef4c4e924167e}
	for i, value := range values {
		if got := math.Float64bits(value); got != want[i] {
			t.Errorf("component %d: got %016x, want %016x", i, got, want[i])
		}
	}
}
