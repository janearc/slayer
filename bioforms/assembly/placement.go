package assembly

import (
	"math"

	"github.com/janearc/slayer/maths"
)

func finiteNumber(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func finitePoint(p maths.Vec) bool {
	return finiteNumber(p.X) && finiteNumber(p.Y) && finiteNumber(p.Z)
}

// validPlacement accepts a finite uniform scale, with zero as the scene
// convention for unit scale. Tolerance leaves room for composed rotations.
func validPlacement(placement maths.Subspace) bool {
	if !finitePoint(placement.Origin) || !finiteNumber(placement.Scale) {
		return false
	}
	basis := placement.Orientation
	if basis == (maths.Orientation{}) {
		return true
	}
	const tolerance = 1e-8
	return math.Abs(basis.X.Len()-1) <= tolerance &&
		math.Abs(basis.Y.Len()-1) <= tolerance &&
		math.Abs(basis.X.Dot(basis.Y)) <= tolerance &&
		basis.Z.Near(basis.X.Cross(basis.Y), tolerance)
}
