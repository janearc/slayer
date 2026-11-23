package planet

import (
	"math"

	"github.com/janearc/celestial/si"
)

// boltzmann is joules a kelvin a molecule, as celestial's si package holds
// it: exact since 2019.
const boltzmann = si.Boltzmann

// squareMicrometre is a square nanometre's share of a square micrometre,
// defined: a nanometre is a thousandth of a micrometre, twice over.
const squareMicrometre = 1e-6

// A resonance is one term of a sellmeier fit: its numerator, and the pole,
// in square micrometres, its denominator is taken from.
type resonance struct{ strength, pole float64 }

// sellmeierFit is a gas's refractive index less one as a sum: a constant,
// then each resonance's numerator over its pole less one over the
// wavelength in micrometres, squared.
type sellmeierFit struct {
	constant   float64
	resonances []resonance
}

// at is the fit's index less one at a wavelength in nanometres.
func (fit sellmeierFit) at(nm float64) float64 {
	inverse := whole / (nm * nm * squareMicrometre)
	sum := fit.constant
	for _, term := range fit.resonances {
		sum += term.strength / (term.pole - inverse)
	}
	return sum
}

// sellmeier is a gas's refractive index less one at a wavelength in
// nanometres, from the refractiveindex.info database's formula 6: a
// constant, then for each resonance a numerator over a pole less one over
// the wavelength in micrometres, squared.
func sellmeier(constant float64,
	resonances ...resonance) func(nm float64) float64 {
	return sellmeierFit{constant: constant, resonances: resonances}.at
}

// Rayleigh's cross-section goes as twenty-four pi cubed over the number
// squared and the wavelength to the fourth.
const (
	// rayleighPrefactor is defined as the twenty-four of the equation.
	rayleighPrefactor = 24
	// rayleighPiPower is defined as the power of pi in the equation: three.
	rayleighPiPower = 3
	// rayleighWavePower is defined as the power of the wavelength in the
	// equation: four.
	rayleighWavePower = 4
)

// crossSection is a gas's rayleigh cross-section by its refractive index
// less one, the number of molecules a cubic metre it was measured at, and
// its king factor.
type crossSection struct {
	index        func(nm float64) float64
	number, king float64
}

// at is the cross-section at a wavelength in nanometres, square metres a
// molecule:
//
//	sigma = 24 pi^3 / (n^2 wavelength^4) ((m^2 - 1) / (m^2 + 2))^2 fk
//
// he and others' equation 1 (acp-21-14927), m the index and n the
// number at which it was measured.
func (gas crossSection) at(nm float64) float64 {
	bent := whole + gas.index(nm)
	lorentz := (bent*bent - whole) / (bent*bent + double)
	metres := nm * nanometre
	return rayleighPrefactor * math.Pow(math.Pi, rayleighPiPower) /
		(gas.number * gas.number * math.Pow(metres, rayleighWavePower)) *
		lorentz * lorentz * gas.king
}

// scatters is a gas's rayleigh cross-section at a wavelength in
// nanometres, square metres a molecule, from its refractive index less
// one at a number of molecules a cubic metre and its king factor.
func scatters(index func(nm float64) float64, number,
	king float64) func(nm float64) float64 {
	return crossSection{index: index, number: number, king: king}.at
}

// standardPressure is the standard atmosphere in pascals: defined.
const standardPressure = 101325.0

// iceTemperature is zero celsius in kelvin: defined by the scale.
const iceTemperature = 273.15

// standard is molecules a cubic metre at zero celsius and 101,325 pascals,
// where bideau-mehu and the others' indices hold.
var standard = standardPressure / (boltzmann * iceTemperature)

const (
	// atomKing is defined as the king factor of an atom: one, by convention
	// (acp-21-14927, table 1).
	atomKing = 1.0
	// methaneKing is defined as methane's king factor: one, as the same
	// table gives it.
	methaneKing = 1.0
)

// the noble gases' indices: bideau-mehu, guern, abjean and
// johannin-gilles 1981, as the refractiveindex.info database gives them
// (rii-ne-, rii-kr- and rii-xe-bideau-mehu); their terms are in
// picks.yaml.
//
// their fit joins the authors' own 140 to 254 nm to older, longer
// measurements. neon holds to 546 nm and krypton and xenon to 623 nm;
// past that it is the fit carried on, modelled.
var (
	neonIndex = sellmeier(nothing,
		resonance{pick("neon-strength-1"), pick("neon-pole-1")},
		resonance{pick("neon-strength-2"), pick("neon-pole-2")})
	kryptonIndex = sellmeier(nothing,
		resonance{pick("krypton-strength-1"), pick("krypton-pole-1")},
		resonance{pick("krypton-strength-2"), pick("krypton-pole-2")},
		resonance{pick("krypton-strength-3"), pick("krypton-pole-3")})
	xenonIndex = sellmeier(nothing,
		resonance{pick("xenon-strength-1"), pick("xenon-pole-1")},
		resonance{pick("xenon-strength-2"), pick("xenon-pole-2")},
		resonance{pick("xenon-strength-3"), pick("xenon-pole-3")})
)

// nanometresPerCentimetre is defined: ten million nanometres make a
// centimetre.
const nanometresPerCentimetre = 1e7

// methaneUnit is the unit methane's index less one is given in: defined,
// the fit's number times 1e-8.
const methaneUnit = 1e-8

// methaneIndex is methane's refractive index less one: he and the
// others' table 1 (acp-21-14927), after sneep and ubachs 2005. times
// 1e8 it is a constant plus a numerator over a pole less the wavenumber
// squared, in reciprocal centimetres; its terms are in picks.yaml.
//
// it holds for 250 to 650 nm, at 288.15 k and 101,325 pascals, and
// 2.546899e19 molecules a cubic centimetre. its king factor is one.
func methaneIndex(nm float64) float64 {
	wavenumber := nanometresPerCentimetre / nm
	return (methaneConstant + methaneStrength/(methanePole-
		wavenumber*wavenumber)) * methaneUnit
}

// methaneConstant, methaneStrength and methanePole are the terms of
// methane's fit, from picks.yaml.
var (
	methaneConstant = pick("methane-index-constant")
	methaneStrength = pick("methane-index-strength")
	methanePole     = pick("methane-index-pole")
)

// methaneNumber is the number at which methane's index holds, a cubic
// metre: theirs.
var methaneNumber = pick("methane-index-number")

// the molar masses, kilograms a mole, from picks.yaml.
var (
	neonMolar    = pick("neon-molar-mass")
	kryptonMolar = pick("krypton-molar-mass")
	xenonMolar   = pick("xenon-molar-mass")
	methaneMolar = pick("methane-molar-mass")
)

// Neon is neon, by its scattering.
func Neon() Gas {
	return Gas{name: "neon", molar: neonMolar,
		section: scatters(neonIndex, standard, atomKing)}
}

// Krypton is krypton, by its scattering.
func Krypton() Gas {
	return Gas{name: "krypton", molar: kryptonMolar,
		section: scatters(kryptonIndex, standard, atomKing)}
}

// Xenon is xenon, by its scattering.
func Xenon() Gas {
	return Gas{name: "xenon", molar: xenonMolar,
		section: scatters(xenonIndex, standard, atomKing)}
}

// Methane is methane, by its scattering, absorbing as given.
func Methane(absorbing func(nm float64) float64) Gas {
	return Gas{name: "methane", molar: methaneMolar,
		section:   scatters(methaneIndex, methaneNumber, methaneKing),
		absorbing: absorbing}
}
