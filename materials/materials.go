// Package materials describes elements, formulas, bulk materials, mixtures, and
// particles. Constructors validate inputs and expose the resulting values
// through methods. See doc/materials.md for the design.
package materials

import (
	"fmt"
	"math"
	"sort"
)

// weights are the elements this package knows, by symbol, at their
// standard atomic weights; a symbol not here is refused. plutonium has
// no stable form, and is at its longest-lived, 244. lead's is its
// atomic number, 82, over the ratio of number to mass, 0.39575, that
// NIST's table of material constants gives it (xray.go), to the four
// figures the table's ratio supports.
var weights = map[string]float64{
	"H": 1.008, "He": 4.0026, "C": 12.011, "N": 14.007, "O": 15.999,
	"Ne": 20.180, "Na": 22.990, "Mg": 24.305, "Al": 26.982, "Si": 28.085,
	"S": 32.06, "Ar": 39.948, "Fe": 55.845, "Ni": 58.693, "Th": 232.04,
	"U": 238.03, "Pu": 244.06, "Ag": 107.87, "Au": 196.97, "Be": 9.0122,
	"Ti": 47.867, "Pb": 207.2,
	// CIAAW standard atomic weights, read 2026-11-15:
	// https://www.ciaaw.org/gallium.htm and /arsenic.htm.
	"Ga": 69.723, "As": 74.921595,
	// CIAAW copper.htm and manganese.htm, read 2026-11-17.
	"Cu": 63.546, "Mn": 54.938043,
}

// within is how far from one a set of shares may sum, for the rounding
// of numbers written by hand
const within = 1e-9

// formula is elements in proportion: water ice is two of H to one of O
type formula struct {
	parts map[string]float64
}

// NewFormula is a formula from element symbols and how many of each. It
// refuses no elements, a symbol it does not know, and a count that is
// not a positive number.
func NewFormula(parts map[string]float64) (formula, error) {
	if len(parts) == 0 {
		return formula{}, refuse("a formula of no elements")
	}
	kept := make(map[string]float64, len(parts))
	for symbol, count := range parts {
		if _, known := weights[symbol]; !known {
			return formula{}, refuse("no element %q", symbol)
		}
		if !positive(count) {
			return formula{}, refuse("%s counted %v times",
				symbol, count)
		}
		kept[symbol] = count
	}
	return formula{parts: kept}, nil
}

// Mass is the formula's mass in atomic mass units, added up in the
// symbols' order so it is the same sum every time
func (made formula) Mass() float64 {
	symbols := make([]string, 0, len(made.parts))
	for symbol := range made.parts {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	sum := 0.0
	for _, symbol := range symbols {
		sum += weights[symbol] * made.parts[symbol]
	}
	return sum
}

// Material is a formula with cited bulk properties: density in grams a
// cubic centimetre and optical readings by band. Derived appearance optics
// are distinguished by their source's Estimated method.
//
// A material made of a formula and a
// density has no optics yet, and Optics answers optics never made; a
// material made of a formula and its optics has no density cited yet,
// and Density answers nought until a source for one is found; a
// material made of a formula alone says what it is made of, with
// neither its density nor its optics cited yet.
//
// AtDensity gives any of
// them a density with its source, and Attenuating its mass attenuation
// coefficients, for x-rays and gamma rays.
type Material struct {
	name    string
	made    formula
	density float64
	weighed Source
	optics  Optics
	stops   Attenuations
}

// NewMaterial is a named material of a formula at a bulk density. It
// refuses no name, a formula never made, and a density that is not a
// positive number.
func NewMaterial(name string, made formula,
	density float64) (Material, error) {
	if name == "" {
		return Material{}, refuse("a material with no name")
	}
	if len(made.parts) == 0 {
		return Material{}, refuse("%s has no formula", name)
	}
	if !positive(density) {
		return Material{}, refuse("%s at density %v",
			name, density)
	}
	return Material{name: name, made: made, density: density}, nil
}

// NewSubstance is a named material that says what it is made of and
// nothing more; its density and optics are not yet cited, so Density
// answers nought and Optics answers optics never made. It refuses no
// name and a formula never made.
func NewSubstance(name string, made formula) (Material, error) {
	if name == "" {
		return Material{}, refuse("a material with no name")
	}
	if len(made.parts) == 0 {
		return Material{}, refuse("%s has no formula", name)
	}
	return Material{name: name, made: made}, nil
}

// newOpticalMaterial is a named material of a formula with its optics,
// and no bulk density cited for it: Density answers nought until a
// source for one is found. It refuses no name, a formula never made, and
// optics never made.
func newOpticalMaterial(name string, made formula,
	sight Optics) (Material, error) {
	if name == "" {
		return Material{}, refuse("a material with no name")
	}
	if len(made.parts) == 0 {
		return Material{}, refuse("%s has no formula", name)
	}
	if len(sight.bands) == 0 {
		return Material{}, refuse("%s has no optics", name)
	}
	return Material{name: name, made: made, optics: sight}, nil
}

// Name is the material's name
func (kind Material) Name() string { return kind.name }

// Density is the material's bulk density in grams a cubic centimetre;
// nought when no density is cited for it
func (kind Material) Density() float64 { return kind.density }

// Optics is the material's refractive index by band, optics never made
// when it has none
func (kind Material) Optics() Optics { return kind.optics }

// fractions are shares of materials that sum to one. Only NewFractions
// makes them, so a mix that does not sum cannot exist.
type fractions struct {
	kinds  []Material
	shares []float64
}

// NewFractions is materials and their shares, in the same order. It refuses no
// materials, a count of shares unlike the count of materials, a share less than
// zero or not a number, a material never made, and shares that do not sum to
// one.
func NewFractions(kinds []Material, shares []float64) (fractions, error) {
	if len(kinds) == 0 {
		return fractions{}, refuse("fractions of nothing")
	}
	if len(kinds) != len(shares) {
		return fractions{}, refuse("%d materials, %d shares",
			len(kinds), len(shares))
	}
	sum := 0.0
	for i, share := range shares {
		if !(share >= 0) || math.IsInf(share, 0) {
			return fractions{}, refuse("a share of %v", share)
		}
		if kinds[i].name == "" {
			return fractions{}, refuse("a material never made")
		}
		sum += share
	}
	if math.Abs(sum-1) > within {
		return fractions{}, refuse("shares sum to %v", sum)
	}
	return fractions{kinds: append([]Material(nil), kinds...),
		shares: append([]float64(nil), shares...)}, nil
}

// Of is the share of the named material in the mix, zero when it is not there
func (mix fractions) Of(name string) float64 {
	total := 0.0
	for i, kind := range mix.kinds {
		if kind.name == name {
			total += mix.shares[i]
		}
	}
	return total
}

// particle is a grain: a size and what it is made of
type particle struct {
	size float64
	made fractions
}

// NewParticle is a grain of a size, in micrometres, made of a mix. It
// refuses a size that is not a positive number and a mix never made.
func NewParticle(size float64, made fractions) (particle, error) {
	if !positive(size) {
		return particle{}, refuse("a grain of size %v", size)
	}
	if len(made.kinds) == 0 {
		return particle{}, refuse("a grain made of nothing")
	}
	return particle{size: size, made: made}, nil
}

// Size is the grain's size in micrometres
func (grain particle) Size() float64 { return grain.size }

// Made is what the grain is made of
func (grain particle) Made() fractions { return grain.made }

// refuse is a constructor's error, saying what it would not make
func refuse(format string, about ...any) error {
	return fmt.Errorf("materials: "+format, about...)
}

// positive is whether a number is greater than zero and finite
func positive(number float64) bool {
	return number > 0 && !math.IsInf(number, 0)
}
