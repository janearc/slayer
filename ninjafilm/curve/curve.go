// Package curve models a projected positive’s density as a function of
// exposure, using a logistic approximation to a Hurter–Driffield curve.
package curve

import (
	"errors"
	"math"

	"github.com/janearc/slayer/ninjafilm/number"
)

// none is no density and no exposure at all, defined as zero.
const none = 0

// steepest is how many times a logistic's height its slope is at the
// middle, for each unit of the logarithm. It is defined by the logistic:
// the slope there is a quarter of the height, so the height's share of
// the slope is four.
const steepest = 4

// unity is the one in a logistic, swing over one plus a power. It is
// defined by the logistic.
const unity = 1

// errCurve is a curve with no room between its ends, or no slope.
var errCurve = errors.New("ninjafilm: a curve has a least density of " +
	"nought or more, a most density above it, and a slope above nought")

// Curve runs from the most density where no light fell to the least
// where plenty did, with a slope in the middle.
type Curve struct{ least, most, slope float64 }

// New is a curve between two densities with a slope, or why not.
func New(least, most, slope float64) (made Curve, err error) {
	if least < none || !number.Positive(most-least) ||
		!number.Positive(slope) {
		return made, errCurve
	}
	return Curve{least, most, slope}, nil
}

// Density is how dense the positive is where an exposure fell. One is
// the exposure at the middle of the curve. None at all is the most
// density, and less than none is none.
//
// The curve is a logistic in the logarithm of exposure. Steepest at its
// middle, a logistic falls a quarter of its height for each unit there.
// So the four makes the slope at the middle the curve's own slope, in
// density for each tenfold of exposure.
func (tone Curve) Density(exposure float64) float64 {
	swing := tone.most - tone.least
	steep := steepest * tone.slope / (swing * math.Ln10)
	lit := math.Max(none, exposure)
	return tone.least + swing/(unity+math.Pow(lit, steep))
}
