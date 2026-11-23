package oceantex

import (
	"errors"
	"math"
)

// Every number below is read from sources.yaml by the name in quotes.
var (
	swellExponent = num("swell.spread.exponent")
	swellLow      = num("swell.slope.low")
	swellHigh     = num("swell.slope.high")
	swellSteps    = num("swell.slope.steps")

	// swellNormal is the constant that makes the swell's spread sum to
	// one over a full turn.
	swellNormal = spreadNormal(swellExponent)
)

// slopeMoment is the order of the spectral moment that is the mean-square
// slope in deep water: m4, the integral of omega^4 times the spectrum
// [Longuet-Higgins 1957].
const slopeMoment = 4

// Swell is a train of long waves that a distant wind raised and that
// have run out of it: a significant height, a peak period and the
// direction they travel toward. Its spectrum is JONSWAP's shape with the
// same peak enhancement as a wind sea's, scaled so that four times the
// root of its variance is its height, and spread about its direction
// as cos^(2s)(theta/2).
//
// A Swell is made by NewSwell. Its height is the height in deep water;
// the sea it sits in carries it through its own depth, as it does its
// wind sea.
type Swell struct {
	height, period, toward float64
	// scale is the factor that gives the shape the height's variance.
	scale float64
}

// NewSwell is the swell of a significant height in metres, a peak period
// in seconds, and a direction toward which it travels, in radians from
// east, counterclockwise. It refuses a height or period that is not
// finite and above nought, and a direction that is not finite.
func NewSwell(height, period, toward float64) (*Swell, error) {
	if !finite(height) || !(height > 0) || !finite(period) ||
		!(period > 0) || !finite(toward) {
		return nil, errors.New("oceantex: a swell has a finite height and " +
			"period above nought, and a finite direction")
	}
	made := &Swell{height: height, period: period, toward: toward}
	peak := made.peak()
	shape := logSum(sumLow*peak, sumHigh*peak, sumSteps,
		func(omega float64) float64 { return swellShape(omega, peak) })
	root := height / heightTimes
	made.scale = root * root / shape
	return made, nil
}

// Height is the swell's significant height in metres.
func (w Swell) Height() float64 { return w.height }

// Period is the swell's peak period in seconds.
func (w Swell) Period() float64 { return w.period }

// Toward is the direction the swell travels toward, in radians from
// east, counterclockwise.
func (w Swell) Toward() float64 { return w.toward }

// peak is the swell's peak angular frequency, radians a second.
func (w Swell) peak() float64 { return 2 * math.Pi / w.period }

// swellShape is JONSWAP's shape about a peak: the spectrum without its
// scale. It is the wind sea's, with the same width, decay and
// enhancement, so a swell and a wind sea of one peak differ only in scale.
func swellShape(omega, peak float64) float64 {
	width := widthBelow
	if omega > peak {
		width = widthAbove
	}
	r := math.Exp(-(omega - peak) * (omega - peak) /
		(2 * width * width * peak * peak))
	return math.Exp(-peakDecay*math.Pow(peak/omega, rolloff)) *
		math.Pow(peakGamma, r) / math.Pow(omega, tailPower)
}

// frequency is the swell's variance of surface height per unit angular
// frequency, in deep water, in square metres seconds a radian.
func (w Swell) frequency(omega float64) float64 {
	if omega <= 0 {
		return 0
	}
	return w.scale * swellShape(omega, w.peak())
}

// spreadNormal is the constant that makes cos^(2s)(theta/2) sum to one
// over a full turn: Gamma(s+1) over 2 root pi Gamma(s+1/2).
func spreadNormal(exponent float64) float64 {
	top, _ := math.Lgamma(exponent + 1)
	bottom, _ := math.Lgamma(exponent + 0.5)
	return math.Exp(top-bottom) / (2 * math.Sqrt(math.Pi))
}

// swellSpread is how the swell's energy is shared among directions: the
// share per radian at angle theta from where it travels toward, Goda's
// cos^(2s)(theta/2). Over a full turn it sums to one.
func swellSpread(theta float64) float64 {
	return swellNormal * math.Pow(math.Cos(theta/2), 2*swellExponent)
}

// logSum is the integral of a density over angular frequency from low to
// high, in even steps in the logarithm of frequency, where d(omega) is
// omega d(log omega).
func logSum(low, high, steps float64, density func(float64) float64) float64 {
	lo, hi := math.Log(low), math.Log(high)
	var sum float64
	for i := 0; i < int(steps); i++ {
		omega := math.Exp(lo + (float64(i)+0.5)*(hi-lo)/steps)
		sum += density(omega) * omega * (hi - lo) / steps
	}
	return sum
}

// swellVariance is the swell's variance of surface height in this sea, in
// square metres: its spectrum with the sea's depth factor, summed as
// the wind sea's is.
func (s Sea) swellVariance() float64 {
	peak := s.Swell.peak()
	return logSum(sumLow*peak, sumHigh*peak, sumSteps,
		func(omega float64) float64 {
			return s.Swell.frequency(omega) * s.shallow(omega)
		})
}

// slope is the swell's own mean-square slope, both ways across the
// surface together: m4 over g squared, which is the sum of k squared
// times the spectrum in deep water [Longuet-Higgins 1957]. It is summed
// from swell.slope.low to swell.slope.high peak frequencies in
// swell.slope.steps even steps in the logarithm of frequency.
func (w Swell) slope() float64 {
	peak := w.peak()
	return logSum(swellLow*peak, swellHigh*peak, swellSteps,
		func(omega float64) float64 {
			return w.frequency(omega) * math.Pow(omega, slopeMoment)
		}) / (gravity * gravity)
}

// swellWavenumber is the swell's spectrum by wave vector, in metres to
// the fourth, carried across by the sea's own dispersion and depth as
// the wind sea's is.
func (s Sea) swellWavenumber(kx, ky float64) float64 {
	k := math.Hypot(kx, ky)
	if k == 0 {
		return 0
	}
	omega, slope := s.turning(k)
	theta := math.Remainder(math.Atan2(ky, kx)-s.Swell.toward, 2*math.Pi)
	return s.Swell.frequency(omega) * s.shallow(omega) *
		swellSpread(theta) * slope / k
}
