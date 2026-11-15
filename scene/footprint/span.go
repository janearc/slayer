package footprint

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Span bounds a box's screen extent, separately from Assess's displacement
// error. Pixels is the diagonal of a containing screen rectangle. This can
// select specimen detail; it is not a certificate of geometric approximation
// error, leaf coverage or shading accuracy.
func (view Perspective) Span(box maths.Box) (Assessment, error) {
	if view.near <= 0 || !validBox(box) {
		return Assessment{}, fmt.Errorf("footprint: invalid bounds or unchecked span view")
	}
	local, err := view.bounds(box)
	if err != nil {
		return Assessment{}, err
	}
	if view.outside(local, 0) {
		return Assessment{}, nil
	}
	depth := interval{-local.Max.Z, -local.Min.Z}
	if depth.low <= view.near {
		return Assessment{Visible: true, Limited: true, Pixels: math.MaxFloat64}, nil
	}
	// Positive depth makes each quotient monotone in its numerator. The
	// interval division includes off-axis perspective and depth variation;
	// using only centre distance would miss both. Camera-axis enclosure can
	// make this rectangle wider than the actual projected geometry.
	x := (interval{local.Min.X, local.Max.X}).div(depth).mul(exact(view.focalAcross))
	y := (interval{local.Min.Y, local.Max.Y}).div(depth).mul(exact(view.focalDown))
	width := exact(x.high).sub(exact(x.low)).high
	height := exact(y.high).sub(exact(y.low)).high
	pixels := math.Nextafter(math.Hypot(width, height), math.Inf(1))
	if !finite(pixels) {
		return Assessment{Visible: true, Limited: true, Pixels: math.MaxFloat64}, nil
	}
	return Assessment{Visible: true, Pixels: pixels}, nil
}
