package scrooge

import "math"

// FlatFrom is how far along a footprint a surface that stays within reach
// of its mean, above and below, may be taken for the mean itself: the
// distance from which the footprint is as wide as the surface is tall,
// crest to trough. It is Resolve's rule for a thing with nothing to open:
// what is no taller than the footprint is wide is its aggregate, and the
// aggregate of a surface that rises and falls about a plane is the plane.
//
// It is nought for a footprint that starts that wide, and infinite for
// one that never grows to it.
//
// A search for where a ray meets such a surface is by this bounded to the
// part of the ray before FlatFrom; past it the ray meets the mean. A sea
// seen from a tower is waves to a few kilometres and a plane to the
// horizon, and how many kilometres follows the picture's size and not a
// number anyone picked.
func (cone Footprint) FlatFrom(reach float64) float64 {
	tall := reach + reach
	if cone.Width >= tall {
		return 0
	}
	if !(cone.Spread > 0) {
		return math.Inf(1)
	}
	return (tall - cone.Width) / cone.Spread
}
