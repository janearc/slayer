package trace

import (
	"github.com/janearc/slayer/maths"
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
