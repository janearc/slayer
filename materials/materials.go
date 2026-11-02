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
// no stable form, and is at its longest-lived, 244.
var weights = map[string]float64{
	"H": 1.008, "He": 4.0026, "C": 12.011, "N": 14.007, "O": 15.999,
	"Ne": 20.180, "Na": 22.990, "Mg": 24.305, "Si": 28.085, "S": 32.06,
	"Ar": 39.948, "Fe": 55.845, "Ni": 58.693, "Th": 232.04, "U": 238.03,
	"Pu": 244.06,
}

// within is how far from one a set of shares may sum, for the rounding
// of numbers written by hand
const within = 1e-9

// Formula is elements in proportion: water ice is two of H to one of O
type Formula struct {
	parts map[string]float64
}

// NewFormula is a formula from element symbols and how many of each. It
// refuses no elements, a symbol it does not know, and a count that is
// not a positive number.
func NewFormula(parts map[string]float64) (Formula, error) {
	if len(parts) == 0 {
		return Formula{}, refuse("a formula of no elements")
	}
	kept := make(map[string]float64, len(parts))
	for symbol, count := range parts {
		if _, known := weights[symbol]; !known {
			return Formula{}, refuse("no element %q", symbol)
		}
		if !positive(count) {
			return Formula{}, refuse("%s counted %v times",
				symbol, count)
		}
		kept[symbol] = count
	}
	return Formula{parts: kept}, nil
}

// Mass is the formula's mass in atomic mass units, added up in the
// symbols' order so it is the same sum every time
func (made Formula) Mass() float64 {
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

// Material is a formula and what is measured of it in bulk: for now
// its density in grams a cubic centimetre; its refractive index by band
// comes next
type Material struct {
	name    string
	made    Formula
	density float64
}

// NewMaterial is a named material of a formula at a bulk density. It
// refuses no name, a formula never made, and a density that is not a
// positive number.
func NewMaterial(name string, made Formula,
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

// Name is the material's name
func (kind Material) Name() string { return kind.name }

// Density is the material's bulk density in grams a cubic centimetre
func (kind Material) Density() float64 { return kind.density }

// Fractions are shares of materials that sum to one. Only NewFractions
// makes them, so a mix that does not sum cannot exist.
type Fractions struct {
	kinds  []Material
	shares []float64
}

// NewFractions is materials and their shares, in the same order. It refuses no
// materials, a count of shares unlike the count of materials, a share less than
// zero or not a number, a material never made, and shares that do not sum to
// one.
func NewFractions(kinds []Material, shares []float64) (Fractions, error) {
	if len(kinds) == 0 {
		return Fractions{}, refuse("fractions of nothing")
	}
	if len(kinds) != len(shares) {
		return Fractions{}, refuse("%d materials, %d shares",
			len(kinds), len(shares))
	}
	sum := 0.0
	for i, share := range shares {
		if !(share >= 0) || math.IsInf(share, 0) {
			return Fractions{}, refuse("a share of %v", share)
		}
		if kinds[i].name == "" {
			return Fractions{}, refuse("a material never made")
		}
		sum += share
	}
	if math.Abs(sum-1) > within {
		return Fractions{}, refuse("shares sum to %v", sum)
	}
	return Fractions{kinds: append([]Material(nil), kinds...),
		shares: append([]float64(nil), shares...)}, nil
}

// Of is the share of the named material in the mix, zero when it is not there
func (mix Fractions) Of(name string) float64 {
	total := 0.0
	for i, kind := range mix.kinds {
		if kind.name == name {
			total += mix.shares[i]
		}
	}
	return total
}

// Particle is a grain: a size and what it is made of
type Particle struct {
	size float64
	made Fractions
}

// NewParticle is a grain of a size, in micrometres, made of a mix. It
// refuses a size that is not a positive number and a mix never made.
func NewParticle(size float64, made Fractions) (Particle, error) {
	if !positive(size) {
		return Particle{}, refuse("a grain of size %v", size)
	}
	if len(made.kinds) == 0 {
		return Particle{}, refuse("a grain made of nothing")
	}
	return Particle{size: size, made: made}, nil
}

// Size is the grain's size in micrometres
func (grain Particle) Size() float64 { return grain.size }

// Made is what the grain is made of
func (grain Particle) Made() Fractions { return grain.made }

// refuse is a constructor's error, saying what it would not make
func refuse(format string, about ...any) error {
	return fmt.Errorf("materials: "+format, about...)
}

// positive is whether a number is greater than zero and finite
func positive(number float64) bool {
	return number > 0 && !math.IsInf(number, 0)
}
