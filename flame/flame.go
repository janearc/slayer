// Package flame models firelight from hot soot and flame puffing as a
// function of size.
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

var errFlame = errors.New("flame: a fire that cannot be")

// sane is whether a number is greater than zero and finite.
func sane(value float64) bool { return value > 0 && !math.IsInf(value, 0) }

// blackBody is a black body's light at a temperature, in each band,
// watts a square metre a steradian: what soot of any depth tends to, and
// the glow a flame's body gives at each point by kirchhoff's law.
func blackBody(heat measure.Temperature) bands.Light {
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

// puffPeriod is how long a flame takes to puff once, for how wide it is at
// its base. modelled: cetegen and ahmed's correlation (combustion and
// flame, 1993), a frequency of 1.5 over the square root of the width in
// metres. malalasekera and others (1996) fit 1.68 over a wider range.
func puffPeriod(across measure.Length) (time.Duration, error) {
	width := across.InMetres()
	if !sane(width) {
		return 0, errFlame
	}
	const puffs = 1.5
	return time.Duration(math.Sqrt(width) / puffs * float64(time.Second)),
		nil
}
