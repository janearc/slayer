package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// tailTested is the straight tail the test below holds, read from the chosen
// numbers.
func tailTested(t *testing.T) Tail {
	t.Helper()
	return Tail{Length: picked(t, "tail-length"),
		Base:   unity,
		Spread: picked(t, "tail-spread"),
		Thick:  unity}
}

// tailPoint is the point on the tail's axis at a height, at no x and no z.
func tailPoint(height float64) maths.Vec {
	return maths.V(nought, height, nought)
}

// A tail streams up from its head, dense at the head and thinning to
// the end, and a curved one bends toward x as it goes.
func TestTailStreamsAndBends(t *testing.T) {
	straight := tailTested(t)
	headHeight := picked(t, "tail-head-height")
	head := straight.Density(tailPoint(headHeight))
	if head < picked(t, "tail-head-density-least") {
		t.Errorf("the head is %v", head)
	}
	endHeight := picked(t, "tail-end-height")
	end := straight.Density(tailPoint(endHeight))
	if end > picked(t, "tail-end-density-most") {
		t.Errorf("the end is %v", end)
	}
	below := straight.Density(tailPoint(-headHeight))
	beyondHeight := picked(t, "tail-beyond-height")
	beyond := straight.Density(tailPoint(beyondHeight))
	if below != nought || beyond != nought {
		t.Error("gas beyond the tail")
	}
	tailBends(t, straight)
}

// tailBends checks a curved tail is denser on its bend than on its axis at
// the end, and that a ray across its middle spans a thin band.
func tailBends(t *testing.T, straight Tail) {
	t.Helper()
	bent := straight
	bent.Curve = picked(t, "tail-curve")
	// at the end the middle has moved three units toward x
	height := picked(t, "tail-bend-height")
	bend := picked(t, "tail-bend-shift") * picked(t, "tail-bend-share")
	on := bent.Density(maths.V(bend, height, nought))
	off := bent.Density(maths.V(nought, height, nought))
	if !(on > double*off) {
		t.Errorf("at the end the bent tail is %v on its bend and %v on the axis",
			on, off)
	}
	middle := picked(t, "tail-ray-middle")
	rayHeight := picked(t, "tail-ray-height")
	across := maths.Ray{At: maths.V(-middle, rayHeight, nought),
		Dir: maths.XAxis}
	enter, leave, ok := bent.Span(across, nought, picked(t, "tail-ray-far"))
	if !ok || enter > middle || leave < middle ||
		leave-enter > picked(t, "tail-span-widest") {
		t.Errorf("the span across the middle is %v to %v, %v", enter, leave, ok)
	}
}
