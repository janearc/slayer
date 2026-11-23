package thing

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// The counts a ray-origin reference is checked against.
const (
	// rayNone is defined: no amount, below which a length is not positive.
	rayNone = 0
	// rayEitherSign is defined by math.IsInf: a sign of nought tests for
	// infinity of both signs.
	rayEitherSign = 0
)

// RayOriginReference supplies the base length of the existing secondary-ray
// offset heuristic. Its zero value retains the geometry-derived reference.
// Camera-relative scenes can declare a local reference rather than inheriting
// a planet's diameter. Position magnitude still contributes to every offset.
// This is a numerical choice, not a proved intersection-error bound.
type RayOriginReference struct{ metres float64 }

// NewRayOriginReference checks a positive base length without changing the
// existing multiplier or the point-magnitude contribution to the heuristic.
func NewRayOriginReference(length measure.Length) (RayOriginReference, error) {
	value := length.InMetres()
	if !(value > rayNone) || math.IsInf(value, rayEitherSign) {
		return RayOriginReference{}, fmt.Errorf("scene: ray-origin reference must be a finite positive length")
	}
	return RayOriginReference{metres: value}, nil
}

// Metres is zero for the geometry-derived reference, or the declared
// positive base length.
func (reference RayOriginReference) Metres() float64 { return reference.metres }
