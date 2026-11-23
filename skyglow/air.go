package skyglow

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// flatRise is the difference in height under which two points are taken
// to be at one height, in metres: the mean density along the path is then
// the density there, and no division by the rise is made.
const flatRise = 1e-6

// Air is the profile of a flat layer of air. Its extinction is Extinction
// per metre at its base and thins by a smooth step to nothing at Depth
// above it, the profile of a mist layer in package medium.
type Air struct {
	// Extinction is the extinction per metre at the base.
	Extinction float64
	// Depth is the height above the base, in metres, where the air has
	// thinned to nothing.
	Depth float64
	// Albedo is the share of the extinction that scatters, nought to one.
	Albedo float64
	// Base is the height of the layer's floor, in metres.
	Base float64
}

// density is the extinction per metre at a height, in metres. Below the
// base it is nothing.
func (air Air) density(height float64) float64 {
	above := height - air.Base
	if above < 0 {
		return 0
	}
	return air.Extinction * (1 - maths.Smooth(0, air.Depth, above))
}

// column is the optical depth straight up from the base to a height: the
// integral of density, in closed form. A smooth step's thinning has the
// integral t - t^3 + t^4/2 in t, the height as a share of Depth, since
// the step is 3 t^2 - 2 t^3.
func (air Air) column(height float64) float64 {
	share := math.Min(math.Max((height-air.Base)/air.Depth, 0), 1)
	cube := share * share * share
	return air.Extinction * air.Depth * (share - cube + cube*share/2)
}

// optical is the optical depth of a straight path of a length between two
// heights: the length times the mean density along it, which for a
// profile that depends on height alone is the change in column over the
// change in height.
func (air Air) optical(from, to, length float64) float64 {
	rise := to - from
	if math.Abs(rise) < flatRise {
		return length * air.density(from)
	}
	return length * (air.column(to) - air.column(from)) / rise
}

// Valid says whether the air can be solved: an extinction, a depth and
// an albedo that are numbers in their ranges.
func (air Air) Valid() bool {
	finite := func(value float64) bool {
		return !math.IsNaN(value) && !math.IsInf(value, 0)
	}
	return finite(air.Extinction) && air.Extinction > 0 &&
		finite(air.Depth) && air.Depth > 0 &&
		finite(air.Albedo) && air.Albedo > 0 && air.Albedo <= 1 &&
		finite(air.Base)
}
