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
	if !(kelvin > 0) {
		return 0
	}
	wave := nm * 1e-9
	return 2 * planck * light * light / math.Pow(wave, 5) /
		math.Expm1(planck*light/(wave*boltzmann*kelvin))
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
	if cleared == 0 || face.hottest == 0 {
		return 0
	}
	kelvin := face.top + (face.hottest-face.top)*cleared
	return cleared * radiance(math.Sqrt(lo*hi), kelvin) * (hi - lo) * 1e-9
}
