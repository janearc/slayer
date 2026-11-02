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

// wide is the half width at the end, bend included.
func (t Tail) wide() float64 {
	return t.Base/2 + t.Length*t.Spread + math.Abs(t.Curve)*t.Length
}

// Bounds is the box the tail fits in.
func (t Tail) Bounds() maths.Box {
	w := t.wide()
	return maths.Box{Min: maths.V(-w, 0, -w), Max: maths.V(w, t.Length, w)}
}

// Span is where a ray is in the tail: its slab and cylinder, probed in
// to the gas.
func (t Tail) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	shifted := maths.Ray{At: r.At.Add(maths.V(0, -t.Length/2, 0)),
		Dir: r.Dir}
	enter, leave, ok := slab(t.Length/2, shifted, near, far)
	if !ok {
		return 0, 0, false
	}
	enter, leave, ok = cylinder(t.wide(), r, enter, leave)
	if !ok {
		return 0, 0, false
	}
	finest := t.Base / 2
	if finest <= 0 {
		finest = t.Length * t.Spread / 8
	}
	return probed(t, r, enter, leave, finest)
}

// Density is the stream's, thinning out and soft across.
func (t Tail) Density(p maths.Vec) float64 {
	if t.Length <= 0 || p.Y < 0 || p.Y >= t.Length {
		return 0
	}
	y := p.Y
	w := t.Base/2 + y*t.Spread
	if w <= 0 {
		return 0
	}
	// the stream's middle bends toward x as it goes; past two and a
	// half widths there is nothing, so a span can find the edge
	bend := t.Curve * y * y / t.Length
	d := math.Hypot(p.X-bend, p.Z) / w
	if d > 2.5 {
		return 0
	}
	return t.Thick * (1 - y/t.Length) * math.Exp(-2*d*d)
}
