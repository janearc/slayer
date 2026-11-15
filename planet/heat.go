package planet

import (
	"fmt"
	"math"
)

// planck and light are, with boltzmann in gases.go, the constants of
// planck's law, si, exact by the si's definition since 2019.
const (
	planck = 6.62607015e-34
	light  = 299792458.0
)

// radiance is planck's law: the light a black body at a temperature in
// kelvin gives off at a wavelength in nanometres, watts a square metre
// a steradian a metre of wavelength.
func radiance(nm, kelvin float64) float64 {
	return thermal(nm).radiance(kelvin)
}

// a warmth is planck's law at a wavelength, less the temperature: what it
// takes of the wavelength alone, and the width of the band the wavelength
// is the middle of, in nanometres.
type warmth struct{ scale, cool, width float64 }

// thermal is the warmth at a wavelength in nanometres, in a band of no width.
func thermal(nm float64) warmth {
	wave := nm * 1e-9
	return warmth{scale: 2 * planck * light * light / math.Pow(wave, 5),
		cool: wave * boltzmann}
}

// warmthOf is the warmth at the middle of a band from lo to hi nanometres.
func warmthOf(lo, hi float64) warmth {
	made := thermal(math.Sqrt(lo * hi))
	made.width = hi - lo
	return made
}

// radiance is planck's law at the warmth's wavelength, at a temperature in
// kelvin.
func (heat warmth) radiance(kelvin float64) float64 {
	if !(kelvin > 0) {
		return 0
	}
	return heat.scale / math.Expm1(planck*light/(heat.cool*kelvin))
}

// Heated is the body with heat under its clouds: the temperature of its
// cloud tops in kelvin, and the hottest a clearing shows, where a vortex
// opens all the way down toward the radon.
//
// a clearing glows as a black body at a temperature between the two, by how
// deep it opens, as jupiter's five-micron hot spots show its warm depths
// through its clearings.
func (body Body) Heated(top, hottest float64) (Body, error) {
	if !(top > 0) || !(hottest >= top) {
		return body, fmt.Errorf("cloud tops at %g K and clearings at %g K",
			top, hottest)
	}
	body.top, body.hottest = top, hottest
	return body, nil
}

// glowed is the light a clearing so deep, from zero to one, gives off in a band
// from lo to hi nanometres, watts a square metre a steradian: a black body at
// the temperature that deep, across the band.
func (face *Face) glowed(cleared, lo, hi float64) float64 {
	return face.glowedBy(cleared, warmthOf(lo, hi))
}

// glowedBy is glowed for a band's warmth.
func (face *Face) glowedBy(cleared float64, heat warmth) float64 {
	if cleared == 0 || face.hottest == 0 {
		return 0
	}
	kelvin := face.top + (face.hottest-face.top)*cleared
	return cleared * heat.radiance(kelvin) * heat.width * 1e-9
}
