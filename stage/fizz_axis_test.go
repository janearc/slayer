package stage_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestFizzPlacementRotatesExistingColumnWithoutChangingDensity(t *testing.T) {
	old := fizzOf(t)
	oriented := fizzOf(t, "        foot: [0, 0, 0]", "        foot: [0, 0, 0]\n        axis: [0.342, -0.94, 0.03]")
	if old.Medium != oriented.Medium {
		t.Fatal("orientation changed the existing Fizz constructor")
	}
	got := oriented.Motion.At(0).Orientation.Out(maths.Up)
	want := maths.V(.342, .03, .94).Unit()
	if got.Sub(want).Len() > 1e-12 {
		t.Fatalf("axis=%v want=%v", got, want)
	}
	reverse := fizzOf(t, "        foot: [0, 0, 0]", "        foot: [0, 0, 0]\n        axis: [0, 0, -1]")
	if math.Abs(reverse.Motion.At(0).Orientation.Out(maths.Up).Y+1) > 1e-12 {
		t.Fatal("antiparallel axis was left upright")
	}
	if old.Motion.At(0).Orientation != (maths.Orientation{}) {
		t.Fatal("omitted orientation changed legacy placement")
	}
}
