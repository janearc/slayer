// Package flame models firelight from hot soot, the fraction of heat emitted as
// radiation, and flame puffing as a function of size.
//
// Kirchhoff's law gives emission as absorptivity times black-body radiance.
// Soot absorptivity is one minus the exponential of negative optical depth. Its
// wavelength dependence favours blue absorption, so the resulting flame is
// bluer than a grey body at the same temperature.
//
// Comments beside parameters distinguish measurements, modelling choices, and
// visual tuning and retain their source information.
package flame

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/radiance"
)

// Visible is how soot's thickness goes with wavelength in the visible:
// as one over the wavelength to this. modelled, hottel and broughton's
// fit to flames (1932), 1.39; a diesel study measured about 1.38.
const Visible = 1.39

// steepest is the most an exponent may be: the visible's, the largest measured.
// zero, a grey body, is the least.
const steepest = Visible

var errFlame = errors.New("flame: a fire that cannot be")

// Soot is a column of glowing soot: how hot it is, how thick it is to
// light at one wavelength, and how its thickness goes with wavelength.
//
// made only by NewSoot.
type Soot struct {
	heat     measure.Temperature
	depth    float64
	at       measure.Length
	exponent float64
}

// NewSoot is soot at a temperature greater than zero, of an optical depth of
// zero or more at a wavelength greater than zero, thick as one over the
// wavelength to an exponent from zero, grey, to Visible.
func NewSoot(heat measure.Temperature, depth float64, at measure.Length,
	exponent float64) (Soot, error) {
	thick := depth >= 0 && !math.IsInf(depth, 0)
	tilted := exponent >= 0 && exponent <= steepest
	if !sane(heat.InKelvin()) || !thick || !sane(at.InMetres()) ||
		!tilted {
		return Soot{}, errFlame
	}
	return Soot{heat: heat, depth: depth, at: at, exponent: exponent}, nil
}

// sane is whether a number is greater than zero and finite.
func sane(value float64) bool { return value > 0 && !math.IsInf(value, 0) }

// Heat is how hot the soot is.
func (soot Soot) Heat() measure.Temperature { return soot.heat }

// Emissivity is how much of a black body the soot is at a wavelength:
// one less e to the minus its optical depth there.
func (soot Soot) Emissivity(wavelength measure.Length) float64 {
	tilt := math.Pow(soot.at.Over(wavelength), soot.exponent)
	return -math.Expm1(-soot.depth * tilt)
}

// Radiance is the soot's light at a wavelength: its emissivity times
// planck's law at its temperature.
func (soot Soot) Radiance(wavelength measure.Length) radiance.Radiance {
	return radiance.Planck(soot.heat, wavelength).
		Times(soot.Emissivity(wavelength))
}

// Seen is the soot's light at every wavelength the 1931 observer has a
// row for, a nanometre apart, as libtheme carries light: in each band,
// watts a square metre a steradian.
func (soot Soot) Seen() bands.Light { return observed(soot.Radiance) }

// Glowing is a black body's light at a temperature, as Seen carries
// soot's: what soot of any depth tends to, and the glow a flame's body
// gives at each point by kirchhoff's law.
func Glowing(heat measure.Temperature) bands.Light {
	return observed(func(wavelength measure.Length) radiance.Radiance {
		return radiance.Planck(heat, wavelength)
	})
}

// observed is a radiance at every wavelength the 1931 observer has a
// row for, a nanometre apart, as light in bands.
func observed(glowAt func(measure.Length) radiance.Radiance) bands.Light {
	lo, hi := swatch.Observed()
	spectrum := make(map[float64]float64, hi-lo+1)
	for nm := lo; nm <= hi; nm++ {
		wavelength := measure.Metres(float64(nm) * nanometre)
		glow := glowAt(wavelength)
		spectrum[float64(nm)] = glow.InWattsPerNanometre()
	}
	return bands.Lines(spectrum)
}

// nanometre is a nanometre in metres
const nanometre = 1e-9

// Radiated is how much of a fire's heat leaves it as light: the heat it
// releases times its radiative fraction, from zero to one.
func Radiated(released measure.Power, fraction float64) (measure.Power,
	error) {
	if !(released.InWatts() >= 0) || math.IsInf(released.InWatts(), 0) ||
		!(fraction >= 0) || fraction > 1 {
		return measure.Power{}, errFlame
	}
	return released.Times(fraction), nil
}

// Period is how long a flame takes to puff once, for how wide it is at
// its base. modelled: cetegen and ahmed's correlation (combustion and
// flame, 1993), a frequency of 1.5 over the square root of the width in
// metres. malalasekera and others (1996) fit 1.68 over a wider range.
func Period(across measure.Length) (time.Duration, error) {
	width := across.InMetres()
	if !sane(width) {
		return 0, errFlame
	}
	const puffs = 1.5
	return time.Duration(math.Sqrt(width) / puffs * float64(time.Second)),
		nil
}
