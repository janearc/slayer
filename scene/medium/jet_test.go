package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// jetTested is the jet the tests below hold, read from the chosen numbers.
func jetTested(t *testing.T) Jet {
	t.Helper()
	return Jet{Length: picked(t, "jet-length"),
		Base:   unity,
		Spread: picked(t, "jet-spread"),
		Thick:  unity}
}

// jetAt is the point on the jet's axis at a height, at no x and no z.
func jetAt(height float64) maths.Vec {
	return maths.V(nought, height, nought)
}

// A jet is two cones, dense at the heart of the base, the same up and down,
// thinning to zero at the end and soft across; its span pulls in to the gas.
func TestJetIsTwoConesThinningOut(t *testing.T) {
	jet := jetTested(t)
	baseHeight := picked(t, "jet-base-height")
	base := jet.Density(jetAt(baseHeight))
	if base < picked(t, "jet-base-density-least") {
		t.Errorf("the base is %v", base)
	}
	middle := picked(t, "jet-middle-height")
	up, down := jet.Density(jetAt(middle)), jet.Density(jetAt(-middle))
	if up != down || up < picked(t, "jet-middle-density-least") ||
		up > picked(t, "jet-middle-density-most") {
		t.Errorf("halfway up %v, halfway down %v", up, down)
	}
	endHeight := picked(t, "jet-end-height")
	end := jet.Density(jetAt(endHeight))
	if end > picked(t, "jet-end-density-most") {
		t.Errorf("the end is %v", end)
	}
	beyondHeight := picked(t, "jet-beyond-height")
	beyond := jet.Density(jetAt(beyondHeight))
	side := maths.V(picked(t, "jet-side-out"), middle, nought)
	if beyond != nought || jet.Density(side) > picked(t, "jet-side-most") {
		t.Error("gas beyond the jet")
	}
	jetSpansSideOn(t, jet)
}

// jetSpansSideOn checks that, side-on through the middle of the upper cone,
// the span is about the cone's width there, not the box's, and that a ray
// over the jet has none.
func jetSpansSideOn(t *testing.T, jet Jet) {
	t.Helper()
	middle := picked(t, "jet-ray-middle")
	height := picked(t, "jet-middle-height")
	far := picked(t, "jet-ray-far")
	through := maths.Ray{At: maths.V(-middle, height, nought), Dir: maths.XAxis}
	enter, leave, ok := jet.Span(through, nought, far)
	if !ok || leave-enter > picked(t, "jet-span-widest") ||
		enter > middle || leave < middle {
		t.Errorf("the span is %v to %v, %v", enter, leave, ok)
	}
	overHeight := picked(t, "jet-over-height")
	over := maths.Ray{At: maths.V(-middle, overHeight, nought),
		Dir: maths.XAxis}
	if _, _, ok := jet.Span(over, nought, far); ok {
		t.Error("a ray over the jet has a span")
	}
}

// A jet that starts out from the origin is empty before its start.
func TestJetStartsWhereAsked(t *testing.T) {
	jet := jetTested(t)
	jet.From = picked(t, "jet-from")
	before := picked(t, "jet-before-height")
	if jet.Density(jetAt(before)) != nought ||
		jet.Density(jetAt(-before)) != nought {
		t.Error("gas before the start")
	}
	after := jetAt(picked(t, "jet-after-height"))
	if jet.Density(after) <= nought {
		t.Error("no gas after the start")
	}
}

// Knots bunch a jet along its length: with them the density varies up
// the axis, without them it falls smoothly.
func TestKnotsBunchAJet(t *testing.T) {
	smooth := jetTested(t)
	knotted := smooth
	knotted.Knots = picked(t, "jet-knots")
	// the knots' own factor, the knotted density over the smooth one,
	// wanders up the axis; without knots it is one all the way
	var wander float64
	last := float64(-unity)
	top, step := picked(t, "jet-knots-top"), picked(t, "jet-knots-step")
	for height := picked(t, "jet-knots-first"); height < top; height += step {
		factor := knotted.Density(jetAt(height)) / smooth.Density(jetAt(height))
		if last >= nought {
			wander += math.Abs(factor - last)
		}
		last = factor
	}
	if !(wander > picked(t, "jet-knots-wander-least")) {
		t.Errorf("the knots' factor wanders only %.3f up the jet", wander)
	}
}
