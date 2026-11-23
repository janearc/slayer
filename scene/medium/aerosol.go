package medium

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// The three wavelengths the eye's channels meet, in nanometres, are the ones
// the tracer's gas optics and scene/gas name, red, green and blue, and the
// green one is the one the extinction is given at: chosen.wavelengthRed,
// chosen.wavelengthGreen and chosen.wavelengthBlue.
//
// The greatest Angstrom exponent an Aerosol accepts is chosen.exponentMost.
// Measured aerosols run from about nought, for coarse dust and fog, to about
// two, for fine smoke; the half beyond that is room, and past it the law is no
// longer a fit.

// DefaultExponent is the Angstrom exponent of city haze when a set says
// none: 1.3, a value in the 1.0 to 1.8 range continental aerosol is
// measured to have. Picked, not measured for any one place.
const DefaultExponent = 1.3

// Aerosol is a flat layer of haze whose extinction depends on wavelength
// as Angstrom's law has it: proportional to the wavelength to the power
// of minus the exponent. Its density is the layer's, which is the density
// at the reference wavelength, 550 nm; Tint says how many times that
// each of the eye's three channels meets.
//
// A layer of fine particles, haze, thins faster toward the red than a
// fog's coarse droplets do, which is why distant lights look redder and
// the air near a lamp looks bluer.
type Aerosol struct {
	*Layer
	// Exponent is the Angstrom exponent: nought is grey, as fog is.
	Exponent float64
	// absorb is a gas mixed through the layer that takes light and
	// scatters none, by channel, as a share of the layer's density: a
	// nitrogen dioxide that takes blue. Nought is none.
	absorb maths.Light
}

// NewAbsorbingAerosol is a layer of haze of an Angstrom exponent with a gas
// mixed through it, as NewAerosol, whose absorption by channel is the
// share of the layer's density it adds, each part from nought.
func NewAbsorbingAerosol(layer *Layer, exponent float64,
	absorb maths.Light) (*Aerosol, error) {
	aerosol, err := NewAerosol(layer, exponent)
	if err != nil {
		return nil, err
	}
	if !(absorb.R >= nought && absorb.G >= nought && absorb.B >= nought) ||
		!weatherFinite(absorb.R+absorb.G+absorb.B) {
		return nil, errors.New("medium: a gas in an aerosol absorbs " +
			"from nought, by a finite share")
	}
	aerosol.absorb = absorb
	return aerosol, nil
}

// Albedo is the share of what each channel meets that is scattered, for
// particles that scatter the share of it the albedo says: the particles'
// share of the channel's extinction, the gas's taken out.
func (aerosol *Aerosol) Albedo(particles float64) maths.Light {
	tint := aerosol.Tint()
	return maths.RGB(
		particles*aerosol.relative(chosen.wavelengthRed)/tint.R,
		particles*aerosol.relative(chosen.wavelengthGreen)/tint.G,
		particles*aerosol.relative(chosen.wavelengthBlue)/tint.B)
}

// NewAerosol is a layer of haze of an Angstrom exponent, which MUST be
// from nought to the most an aerosol is measured to have.
func NewAerosol(layer *Layer, exponent float64) (*Aerosol, error) {
	if layer == nil {
		return nil, errors.New("medium: an aerosol needs a layer")
	}
	if !(exponent >= nought && exponent <= chosen.exponentMost) {
		return nil, errors.New("medium: an Angstrom exponent is from " +
			"nought to two and a half")
	}
	return &Aerosol{Layer: layer, Exponent: exponent}, nil
}

// Tint is how many times its density each of the eye's channels meets:
// the wavelength over the reference one, to the power of minus the
// exponent. Green is one.
func (aerosol *Aerosol) Tint() maths.Light {
	return maths.RGB(aerosol.relative(chosen.wavelengthRed),
		aerosol.relative(chosen.wavelengthGreen),
		aerosol.relative(chosen.wavelengthBlue)).Add(aerosol.absorb)
}

// relative is the extinction at a wavelength, in nanometres, as a share of
// the extinction at the reference wavelength.
func (aerosol *Aerosol) relative(wavelength float64) float64 {
	return math.Pow(wavelength/chosen.wavelengthGreen, -aerosol.Exponent)
}

// Tick is the aerosol at a time. The layer changes only its noise phase,
// so the copy shares the layer's bounds and ground and keeps the exponent.
func (aerosol *Aerosol) Tick(at time.Duration) Medium {
	ticked := *aerosol.Layer
	ticked.seconds = at.Seconds()
	return &Aerosol{Layer: &ticked, Exponent: aerosol.Exponent,
		absorb: aerosol.absorb}
}
