package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// boxOnlyMedium has no tighter Span method: the box is its entire support.
// Its box reaches edge from the origin on every axis, and its density is
// density everywhere.
type boxOnlyMedium struct{ edge, density float64 }

// Bounds is the cube that reaches edge from the origin on every axis.
func (only boxOnlyMedium) Bounds() maths.Box {
	return maths.Box{
		Min: maths.V(-only.edge, -only.edge, -only.edge),
		Max: maths.V(only.edge, only.edge, only.edge),
	}
}

// Density is the one density the medium has, wherever it is asked.
func (only boxOnlyMedium) Density(maths.Vec) float64 { return only.density }

// TestWispyUsesTheBoxWhenItsMediumHasNoSpan holds that a ray down the axis
// of a box-only medium enters and leaves at the faces of its box.
func TestWispyUsesTheBoxWhenItsMediumHasNoSpan(t *testing.T) {
	only := boxOnlyMedium{
		edge:    picked(t, "box-span-edge"),
		density: picked(t, "box-span-density"),
	}
	wispy := Wispy{Of: only}
	start := maths.V(nought, nought, picked(t, "box-span-ray-start"))
	ray := maths.Ray{At: start, Dir: maths.V(nought, nought, -unity)}
	far := picked(t, "box-span-far")
	enter, leave, hit := wispy.Span(ray, nought, far)
	wantEnter := picked(t, "box-span-enter")
	wantLeave := picked(t, "box-span-leave")
	if !hit || enter != wantEnter || leave != wantLeave {
		t.Fatalf("box fallback: %g %g %v", enter, leave, hit)
	}
}
