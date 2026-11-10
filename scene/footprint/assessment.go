package footprint

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Assessment reports possible visibility and a projected displacement bound.
// Pixels is not a measured terrain error: the supplied displacement must be
// justified independently. Limited means the view meets the near-plane bound.
type Assessment struct {
	Visible, Limited bool
	Pixels           float64
}

// Assess encloses a world-axis box and a possible displacement in metres.
// The displacement bounds the Euclidean change in world metres. A conservative
// axis norm converts it to a component bound in the supplied camera basis.
// The input box must enclose the actual geometry, including unsampled peaks.
func (view Perspective) Assess(box maths.Box, displacement measure.Length) (Assessment, error) {
	metres := displacement.InMetres()
	if view.near <= 0 || !validBox(box) || !finite(metres) || metres < 0 {
		return Assessment{}, fmt.Errorf("footprint: invalid bounds, displacement or unchecked view")
	}
	if metres > 0 {
		metres = view.axisMargin(metres)
		if !finite(metres) {
			return Assessment{Visible: true, Limited: true, Pixels: math.MaxFloat64}, nil
		}
	}
	local, err := view.bounds(box)
	if err != nil {
		return Assessment{}, err
	}
	if view.outside(local, metres) {
		return Assessment{}, nil
	}
	result := Assessment{Visible: true}
	if metres == 0 {
		return result, nil
	}
	depth := -local.Max.Z
	across := max(math.Abs(local.Min.X), math.Abs(local.Max.X))
	down := max(math.Abs(local.Min.Y), math.Abs(local.Max.Y))
	remaining := exact(depth).sub(exact(metres))
	if remaining.low <= view.near {
		return Assessment{Visible: true, Limited: true, Pixels: math.MaxFloat64}, nil
	}
	// |dx'| <= f*e*(1+|x|/d)/(d-e). Each factor is maximized over the box.
	// Combine column and row displacement, not physical image-plane
	// distance.
	horizontal := view.pixelBound(view.focalAcross, metres, depth, across, remaining)
	vertical := view.pixelBound(view.focalDown, metres, depth, down, remaining)
	result.Pixels = math.Nextafter(math.Hypot(horizontal, vertical), math.Inf(1))
	if !finite(result.Pixels) {
		result.Pixels, result.Limited = math.MaxFloat64, true
	}
	return result, nil
}

// axisMargin uses the L1 norm as an upper bound on each camera axis's norm.
// It stays safe for the small basis roundoff admitted by the constructor.
func (view Perspective) axisMargin(metres float64) float64 {
	orientation := view.eye.Orientation
	if orientation == (maths.Orientation{}) {
		return metres
	}
	bound := 0.0
	for _, axis := range [3]maths.Vec{orientation.X, orientation.Y, orientation.Z} {
		norm := exact(math.Abs(axis.X)).add(exact(math.Abs(axis.Y))).add(exact(math.Abs(axis.Z)))
		bound = max(bound, norm.high)
	}
	return exact(metres).mul(exact(bound)).high
}

// pixelBound evaluates positive factors with outward-rounded intervals.
func (view Perspective) pixelBound(focal, metres, depth, extent float64, remaining interval) float64 {
	scale := exact(focal).mul(exact(metres).div(remaining))
	offAxis := exact(1).add(exact(extent).div(exact(depth)))
	return scale.mul(offAxis).high
}

// validBox permits flat bounds and points but refuses unordered/non-finite
// axes.
func validBox(box maths.Box) bool {
	return finitePoint(box.Min) && finitePoint(box.Max) &&
		box.Min.X <= box.Max.X && box.Min.Y <= box.Max.Y && box.Min.Z <= box.Max.Z
}
