package skylight

import "math"

// a vector is a place or a way in metres from the planet's centre, with
// the sun far along plus z.
type vector struct{ x, y, z float64 }

// dot is the vectors' inner product.
func (a vector) dot(b vector) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }

// plus is the vectors' sum.
func (a vector) plus(b vector) vector {
	return vector{a.x + b.x, a.y + b.y, a.z + b.z}
}

// scaled is the vector times a number.
func (a vector) scaled(by float64) vector {
	return vector{a.x * by, a.y * by, a.z * by}
}

// length is the vector's length.
func (a vector) length() float64 { return math.Sqrt(a.dot(a)) }

// unit is the vector at length one.
func (a vector) unit() vector { return a.scaled(one / a.length()) }

// exit is how far a photon at position going along heading, a unit
// vector, travels in its layer before it meets the layer's inner or
// outer sphere, and whether the outer.
//
// The photon is in the layer by the walk's own count, whatever its
// position's rounding says, so a gap that puts it a hair outside the
// layer is taken as nought: it is on that sphere. Each distance comes
// from the root of the quadratic that does not subtract near equals.
func exit(position, heading vector, through *layer) (float64, bool) {
	square := position.dot(position)
	along := position.dot(heading)
	out := outward(along, math.Min(square-through.outerSquared, none))
	in := inward(along, math.Max(square-through.innerSquared, none))
	if in < out {
		return in, false
	}
	return out, true
}

// outward is the distance along a line to a sphere the line starts
// inside: the positive root of t squared plus 2 along t plus gap, where
// gap, the square of the start's radius less the sphere's, is not
// above nought. Going outward the root is written so as not to cancel.
func outward(along, gap float64) float64 {
	root := math.Sqrt(along*along - gap)
	if along > none {
		return -gap / (along + root)
	}
	return root - along
}

// inward is the distance along a line to a sphere the line starts
// outside, with gap not below nought, or endless when the line goes
// away from it or passes it by. The nearer root is written as gap over
// the far one's sum, so as not to cancel at grazing.
func inward(along, gap float64) float64 {
	square := along*along - gap
	if along >= none || square < none {
		return math.Inf(one)
	}
	return gap / (math.Sqrt(square) - along)
}

// chord is how far the straight line from a point at radius from the
// centre, whose way has a projection along on the point's radial, runs
// inside the sphere of radius sphereRadius, on its way and not behind.
// It is nought when the line misses the sphere or only touches it.
func chord(radius, along, sphereRadius float64) float64 {
	gap := (radius - sphereRadius) * (radius + sphereRadius)
	root := math.Sqrt(math.Max(along*along-gap, none))
	return math.Max(root-along, none) - math.Max(-root-along, none)
}

// basis is two unit vectors square to each other and to axis, a unit
// vector, without a branch at either pole: Duff and others, "Building
// an Orthonormal Basis, Revisited", Journal of Computer Graphics
// Techniques 6(1), 2017.
func basis(axis vector) (vector, vector) {
	sign := math.Copysign(one, axis.z)
	scale := -one / (sign + axis.z)
	cross := axis.x * axis.y * scale
	return vector{one + sign*axis.x*axis.x*scale, sign * cross,
			-sign * axis.x},
		vector{cross, sign + axis.y*axis.y*scale, -axis.y}
}

// turned is the way at an angle whose cosine is cosine from axis, a
// unit vector, spun about it by spin of a full turn.
func turned(axis vector, cosine, spin float64) vector {
	first, second := basis(axis)
	sine := math.Sqrt(math.Max(one-cosine*cosine, none))
	across, along := math.Sincos(fullTurn * spin)
	return axis.scaled(cosine).plus(first.scaled(sine * along)).
		plus(second.scaled(sine * across))
}
