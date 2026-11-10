// Package radiance models spectral radiance in watts per square metre,
// steradian, and nanometre, including thermal emission through Planck's law.
package radiance

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
)

// Radiance is spectral radiance, held in W m^-2 sr^-1 nm^-1.
type Radiance struct{ watts float64 }

// WattsPerNanometre is a radiance of so many W m^-2 sr^-1 nm^-1.
func WattsPerNanometre(amount float64) Radiance { return Radiance{amount} }

// InWattsPerNanometre is the radiance in W m^-2 sr^-1 nm^-1.
func (glow Radiance) InWattsPerNanometre() float64 { return glow.watts }

// Times is the radiance scaled by a share.
func (glow Radiance) Times(share float64) Radiance {
	return Radiance{glow.watts * share}
}

// none is no heat and no wavelength. Defined: zero.
const none = 0

// Planck is planck's law: the radiance of a black body at a temperature,
// at a wavelength, as celestial's radiation.Planck gives it. Nought or
// less of either, or no number, gives none.
func Planck(heat measure.Temperature, wavelength measure.Length) Radiance {
	kelvin := heat.InKelvin()
	nanometres := radiation.MetresToNm(wavelength.InMetres())
	if !(kelvin > none) || !(nanometres > none) {
		return Radiance{}
	}
	return Radiance{radiation.Planck(nanometres, kelvin)}
}
