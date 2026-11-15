package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// Sums done by hand for the wood's rings, each at a place chosen so that
// one slip in the arithmetic of a ring moves it from one wood to the
// other. A ring a unit wide is early wood until three quarters through,
// late wood from 0.83 to 0.92, and early again by its end.

// A ring half as wide has twice as many rings in the same distance:
// 0.45 out is nine tenths through its first ring, and late wood.
func TestANarrowerRingComesSooner(t *testing.T) {
	narrow := trueOak()
	narrow.Ring = 0.5
	if got := narrow.At(maths.V(0, 0.45, 0)); !near(got, dark) {
		t.Errorf("0.45 out in rings half a unit wide is %v, not late", got)
	}
	// the anti-test: in rings a unit wide the same place is early wood
	if got := trueOak().At(maths.V(0, 0.45, 0)); !near(got, pale) {
		t.Errorf("0.45 out in rings a unit wide is %v, not early", got)
	}
}

// The second ring is as the first: nine tenths through it is late wood.
// The anti-test: three tenths through it is early, so the second ring
// is a ring and not all late.
func TestTheSecondRingIsAsTheFirst(t *testing.T) {
	if got := trueOak().At(maths.V(0, 1.9, 0)); !near(got, dark) {
		t.Errorf("nine tenths through the second ring is %v, not late", got)
	}
	if got := trueOak().At(maths.V(0, 1.3, 0)); !near(got, pale) {
		t.Errorf("three tenths through the second ring is %v, not early", got)
	}
}

// Late wood ends with its year: at the very end of a ring the wood has
// turned back to early. The anti-test: a tenth of a ring before its
// end it has not, and is late still.
func TestLateWoodEndsWithItsYear(t *testing.T) {
	if got := trueOak().At(maths.V(0, 0.9999999, 0)); !near(got, pale) {
		t.Errorf("at the very end of a ring the wood is %v, not early", got)
	}
	if got := trueOak().At(maths.V(0, 0.9, 0)); !near(got, dark) {
		t.Errorf("a tenth of a ring before its end the wood is %v, not late", got)
	}
}

// Half way up the turn from early to late the wood is half of each: the
// turn starts three quarters through a ring and takes 0.08 of it, so its
// middle is at 0.79. The anti-test: where the turn starts, at 0.75, the
// wood is early and no part late.
func TestHalfWayUpTheTurnIsHalfOfEach(t *testing.T) {
	half := pale.Scale(0.5).Add(dark.Scale(0.5))
	if got := trueOak().At(maths.V(0, 0.79, 0)); !near(got, half) {
		t.Errorf("half way up the turn the wood is %v, want %v", got, half)
	}
	if got := trueOak().At(maths.V(0, 0.75, 0)); !near(got, pale) {
		t.Errorf("where the turn starts the wood is %v, not early", got)
	}
}

// A wander pushes a ring outward where its wave is above nought. One
// unit along the grain the wave is sin 0.7 + 0.35 sin 2.9, which is
// 0.644 + 0.084 = 0.728; with a wander of a half, a place half a ring
// out counts as 0.5 + 0.364 = 0.864 through, and is late wood. Half a
// unit along, the wave is sin 0.35 + 0.35 sin 1.45 = 0.343 + 0.347 =
// 0.690, and 0.53 out counts as 0.875 through: late wood too.
func TestAWanderPushesARingOutward(t *testing.T) {
	wandering := trueOak()
	wandering.Wander = 0.5
	if got := wandering.At(maths.V(1, 0.5, 0)); !near(got, dark) {
		t.Errorf("one unit along, half a ring out in a wandering wood is %v, "+
			"not late", got)
	}
	if got := wandering.At(maths.V(0.5, 0.53, 0)); !near(got, dark) {
		t.Errorf("half a unit along, 0.53 out in a wandering wood is %v, "+
			"not late", got)
	}
	// the anti-test: with no wander both places are early wood
	if got := trueOak().At(maths.V(1, 0.5, 0)); !near(got, pale) {
		t.Errorf("half a ring out in a wood with no wander is %v, not early", got)
	}
}
