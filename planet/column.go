package planet

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/si"
)

// avogadro is molecules in a mole, as celestial's si package holds it:
// exact by the si's own definition since 2019.
const avogadro = si.Avogadro

// tintBlue and tintRed are the wavelengths, in nanometres, a tint's two
// values are set at, from picks.yaml.
var (
	tintBlue = pick("tint-blue-wavelength")
	tintRed  = pick("tint-red-wavelength")
)

// shareTolerance is how far the shares of a layer's gases may sum from one
// and still be taken as summing to it, so a sum rounded off is not refused.
var shareTolerance = pick("layer-share-tolerance")

// A Gas is a gas by what light does to it: its scattering cross-section
// at a wavelength in nanometres, square metres a molecule, and its molar
// mass, kilograms a mole.
type Gas struct {
	name      string
	section   func(nm float64) float64
	molar     float64
	absorbing func(nm float64) float64
}

// NewGas is a gas from its cross-section and molar mass, refused if the mass is
// not greater than zero or there is no cross-section.
func NewGas(name string, section func(nm float64) float64,
	molar float64) (Gas, error) {
	if !(molar > nothing) {
		return Gas{}, fmt.Errorf("gas %s: a molar mass of %g kg", name,
			molar)
	}
	if section == nil {
		return Gas{}, fmt.Errorf("gas %s: no scattering cross-section", name)
	}
	return Gas{name: name, section: section, molar: molar}, nil
}

// Absorbing is the gas that also absorbs, by a cross-section in square
// metres a molecule at a wavelength in nanometres.
func (gas Gas) Absorbing(section func(nm float64) float64) Gas {
	gas.absorbing = section
	return gas
}

// A Haze is droplets hanging in a layer: their optical depth through the
// whole layer, the share of what they turn aside that they scatter, and
// how far forward they lean. its depth is the same at every wavelength,
// until the droplets' own optics are read; its albedo may be tinted.
type Haze struct {
	depth, lean float64
	albedo      tint
}

// NewHaze is a grey haze, refused outside what a haze can be.
func NewHaze(depth, albedo, lean float64) (Haze, error) {
	if !(depth >= nothing) || !(albedo >= nothing && albedo <= whole) ||
		!(lean > -whole && lean < whole) {
		return Haze{}, fmt.Errorf("a haze of depth %g, albedo %g and "+
			"lean %g", depth, albedo, lean)
	}
	return Haze{depth: depth, lean: lean, albedo: tint{albedo, albedo}},
		nil
}

// Tinted is the haze with its albedo at 400 nm set apart from the one it
// was made with, which holds at 700 nm.
func (haze Haze) Tinted(blue float64) (Haze, error) {
	made, err := haze.albedo.blued(blue)
	haze.albedo = made
	return haze, err
}

// a tint is an albedo that changes across the visible: its value at 400
// nm and at 700 nm, straight between them in wavelength, held past
// either end. the same two is grey.
type tint struct{ blue, red float64 }

// blued is the tint with its value at 400 nm set, refused outside zero to one.
func (made tint) blued(blue float64) (tint, error) {
	if !(blue >= nothing && blue <= whole) {
		return made, fmt.Errorf("an albedo of %g in the blue", blue)
	}
	made.blue = blue
	return made, nil
}

// at is the tint at a wavelength in nanometres.
func (made tint) at(nm float64) float64 {
	share := math.Max(nothing, math.Min(whole,
		(nm-tintBlue)/(tintRed-tintBlue)))
	return made.blue + share*(made.red-made.blue)
}

// a part is one gas and its share of a layer by number.
type part struct {
	gas   Gas
	share float64
}

// A Layer is the air between two pressures, in pascals, with its gases
// by share of the molecules and its haze.
type Layer struct {
	top, bottom float64
	parts       []part
	haze        Haze
}

// NewLayer is a layer from its pressures, its gases and their shares, and its
// haze. shares must be greater than zero and sum to one.
func NewLayer(top, bottom float64, gases []Gas, shares []float64,
	haze Haze) (Layer, error) {
	if !(top >= nothing) || !(bottom > top) || len(gases) != len(shares) ||
		len(gases) == nothing {
		return Layer{}, fmt.Errorf("a layer from %g to %g pascals with "+
			"%d gases and %d shares", top, bottom, len(gases), len(shares))
	}
	var sum float64
	parts := make([]part, len(gases))
	for i, gas := range gases {
		if !(shares[i] > nothing) {
			return Layer{}, fmt.Errorf("%s at a share of %g", gas.name,
				shares[i])
		}
		sum += shares[i]
		parts[i] = part{gas: gas, share: shares[i]}
	}
	if math.Abs(sum-whole) > shareTolerance {
		return Layer{}, fmt.Errorf("shares summing to %g, not one", sum)
	}
	return Layer{top: top, bottom: bottom, parts: parts, haze: haze}, nil
}

// molecules is how many molecules stand over a square metre between the
// layer's pressures, under a gravity in metres a second squared: the
// weight of the air over its mean molecule's mass.
func (layer Layer) molecules(gravity float64) float64 {
	var molar float64
	for _, each := range layer.parts {
		molar += each.share * each.gas.molar
	}
	return (layer.bottom - layer.top) / gravity / (molar / avogadro)
}

// optics is the layer at a wavelength, under a gravity.
func (layer Layer) optics(nm, gravity float64) optics {
	count := layer.molecules(gravity)
	made := optics{haze: layer.haze.depth,
		hazeAlbedo: layer.haze.albedo.at(nm), hazeLean: layer.haze.lean}
	for _, each := range layer.parts {
		made.scatter += count * each.share * each.gas.section(nm)
		if each.gas.absorbing != nil {
			made.absorb += count * each.share * each.gas.absorbing(nm)
		}
	}
	return made
}
