// Package lights computes physical spectral power for lamps, stars, lasers, and
// other sources, in watts per libtheme band from gamma rays to radio.
//
// libtheme’s visible black-body swatches describe relative colour over the CIE
// 1931 observer’s 360–830 nm range. This package supplies absolute power,
// including infrared and other radiation outside that range, while libtheme’s
// bands.Light retains band boundaries, storage, and spectral operations.
//
// Only the physical band-power calculation belongs here. Colour conversion
// still belongs to libtheme. Its current bands.Light.Swatch implementation
// samples every whole nanometre, making it impractical for a black body with
// power in radio bands billions of nanometres wide.
package lights

import (
	"errors"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/ninjafilm/maths"
)

// none is no power at all. It is defined as zero.
const none = 0

// pair is the two of the middle of a band, half way between its edges.
// It is defined as two.
const pair = 2

// inclusive is one band on, so that both the first band and the last of
// a run are in it. It is defined as one.
const inclusive = 1

var errDark = errors.New("ninjafilm: a light's spectrum has no power in " +
	"any band, so there is nothing to scale to its watts")

// Planck is the spectrum of a black body at a temperature. It is the
// share of its power in every band celestial names, from the shortest
// gamma ray to the longest radio wave. It comes from Planck's law. Only
// the shape is meant. Emitters scale it to their watts.
//
//ninja:vow practice slayer's lamps take their colour here; tests only so far
func Planck(heat maths.Temperature) bands.Light {
	spectrum := map[float64]float64{}
	shortest, _ := radiation.Gamma().Edges()
	_, longest := radiation.Radio().Edges()
	first, last := bands.Of(shortest), bands.Of(longest)
	for offset := range int(last-first) + inclusive {
		band := first + bands.Band(offset)
		short, long := band.Edges()
		if given := heat.Exitance(short, long); given > none {
			spectrum[(short+long)/pair] = given
		}
	}
	return bands.Lines(spectrum)
}

// Emitter is something that gives off light: how many watts it gives,
// in each band.
//
//ninja:vow practice slayer's lamps hold one; tests only so far
type Emitter struct {
	light bands.Light
}

// Point is the simplest emitter: one that gives the same light every
// way, from a point. Its spectrum says what colour it is. Its power says
// how much there is. The spectrum is scaled so that its bands add up to
// the power. It refuses a spectrum with no power in it.
//
//ninja:vow practice slayer's lamps are made here; tests only so far
func Point(spectrum bands.Light, power maths.Power) (made Emitter,
	err error) {
	total := totalOf(spectrum)
	if !(total > none) {
		return made, errDark
	}
	return Emitter{spectrum.Scale(power.Watts() / total)}, nil
}

// Light is the light the emitter gives off, in watts a band.
//
//ninja:vow practice slayer's lamps give their light here; tests only so far
func (emitter Emitter) Light() bands.Light {
	return emitter.light
}

// totalOf is all the power in a light, every band added up. This is band
// work, and it is libtheme's: bands.Light has no total of its own yet.
// It is kept private here, read through libtheme's Span and Power.
//
// So nothing comes to depend on ninjafilm for it. It goes the day
// libtheme gives a light its total.
func totalOf(light bands.Light) float64 {
	first, last := light.Span()
	var total float64
	for offset := range int(last-first) + inclusive {
		total += light.Power(first + bands.Band(offset))
	}
	return total
}
