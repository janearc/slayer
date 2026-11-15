package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// goldenRelative and goldenAbsolute are how far a channel may be from
// its golden value: a part in a hundred thousand of it, or a ten
// millionth outright.
//
// arm64 fuses a multiply and an add, so reordering an expression moves
// a number's last bit. That is not a change of picture; a part in a
// thousand is.
const (
	goldenRelative = 1e-5
	goldenAbsolute = 1e-7
)

// lookingAt is an eye at a place, looking at a target.
func lookingAt(from, target maths.Vec) maths.Subspace {
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, target)}
}

// tenBack is the eye most of these tests look from: ten units up z,
// looking at the middle of the world.
func tenBack() maths.Subspace {
	return lookingAt(maths.V(0, 0, 10), maths.Vec{})
}

// same is whether a channel is the value wanted of it: equal, as two
// infinite depths are, or within a share of it, or within the golden
// absolute tolerance. A NaN is never the same as anything, itself
// included.
func same(got, want, share float64) bool {
	if got == want {
		return true
	}
	gap := math.Abs(got - want)
	return gap <= goldenAbsolute || gap <= share*math.Abs(want)
}

// apart is how far a channel is from the value wanted of it: zero when it is
// the same to a share, and infinitely far when either is a NaN or only one is
// infinite.
func apart(got, want, share float64) float64 {
	if same(got, want, share) {
		return 0
	}
	gap := math.Abs(got - want)
	if math.IsNaN(gap) {
		return math.Inf(1)
	}
	return gap
}
