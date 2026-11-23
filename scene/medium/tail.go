package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Tail is a stream of gas or dust from a body away from its sun: from the
// origin up y to Length, Base across at the origin and spreading by Spread of
// the distance out, bent toward x by Curve of the distance squared over the
// length, as a dust tail lags along the orbit.
//
// It is Thick at the head and thins to zero at the end, gaussian across. A
// comet's dust tail is broad and curved, its ion tail narrow and straight; a
// planet too near its star streams the same way.
type Tail struct {
	Length, Base, Spread float64
	Curve                float64
	Thick                float64
}

// wide is how far from the y axis the tail's gas reaches: the bend at the end,
// and the cut of the gaussian, chosen.skirtWidths half widths of the widest
// the tail gets. Past it the density is exactly zero.
func (tail Tail) wide() float64 {
	head := tail.Base / double
	widest := math.Max(head, head+tail.Length*tail.Spread)
	return math.Abs(tail.Curve)*tail.Length + chosen.skirtWidths*widest
}

// Bounds is the box the tail fits in: all its gas, and nothing else is in
// reach of the density.
func (tail Tail) Bounds() maths.Box {
	across := tail.wide()
	return corners(maths.V(-across, nought, -across),
		maths.V(across, tail.Length, across))
}

// Span is where a ray is in the tail: its slab and cylinder, probed in
// to the gas.
func (tail Tail) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	shifted := maths.Ray{At: ray.At.Add(maths.V(nought,
		-tail.Length/double, nought)), Dir: ray.Dir}
	enter, leave, hit := slab(tail.Length/double, shifted, near, far)
	if !hit {
		return nought, nought, false
	}
	enter, leave, hit = cylinder(tail.wide(), ray, enter, leave)
	if !hit {
		return nought, nought, false
	}
	finest := tail.Base / double
	if finest <= nought {
		finest = tail.Length * tail.Spread / chosen.streamEighth
	}
	return probed(tail, ray, enter, leave, finest)
}

// Density is the stream's, thinning out and soft across.
func (tail Tail) Density(point maths.Vec) float64 {
	if tail.Length <= nought || point.Y < nought || point.Y >= tail.Length {
		return nought
	}
	along := point.Y
	width := tail.Base/double + along*tail.Spread
	if width <= nought {
		return nought
	}
	// the stream's middle bends toward x as it goes
	bend := tail.Curve * along * along / tail.Length
	across := math.Hypot(point.X-bend, point.Z) / width
	if across > chosen.skirtWidths {
		return nought
	}
	return tail.Thick * (unity - along/tail.Length) *
		math.Exp(-chosen.streamBell*across*across)
}
