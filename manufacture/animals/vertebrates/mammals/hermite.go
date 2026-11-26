package mammals

// Hermite is the cubic through (x0, y0) and (x1, y1) with slopes m0
// and m1 there, at x between them.
func Hermite(x0, x1, y0, y1, m0, m1, x float64) float64 {
	span := x1 - x0
	t := (x - x0) / span
	square, cube := t*t, t*t*t
	return (2*cube-3*square+1)*y0 + (cube-2*square+t)*span*m0 +
		(3*square-2*cube)*y1 + (cube-square)*span*m1
}
