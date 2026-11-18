package airs

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/research"
)

// Constants of nature and of the units, none of them a pick.
const (
	// avogadro is molecules in a mole, exact since 2019 (codata 2018).
	avogadro = 6.02214076e23 // defined: the SI value
	// boltzmann is joules a kelvin a molecule, exact since 2019
	// (codata 2018). With the pressure and temperature the index was
	// measured at, it gives the molecules a cubic metre there, the ideal
	// gas law's: Loschmidt's constant, 2.6868e25, as codata has it.
	boltzmann = 1.380649e-23 // defined: the SI value
	// metresPerNanometre turns a wavelength into metres.
	metresPerNanometre = 1e-9 // defined: the unit
	// squareNanometresPerSquareMicrometre turns the inverse square of a
	// wavelength in nanometres into one in micrometres.
	squareNanometresPerSquareMicrometre = 1e6 // defined: the unit
)

// The parts of Rayleigh's cross section, sigma = 24 pi^3 / (lambda^4
// N^2) ((n^2 - 1) / (n^2 + 2))^2 F, Bodhaine and others 1999, equation 23.
const (
	rayleighScale = 24 // defined: Rayleigh's 24 pi^3
	lorentzShift  = 2  // defined: the Lorentz-Lorenz n^2 + 2
	powerOfPi     = 3  // defined: Rayleigh's pi cubed
	powerOfLength = 4  // defined: Rayleigh's wavelength to the fourth
)

// kingCarbonDioxide is the name of carbon dioxide's King factor in
// celestial's research data: 1.15, Bodhaine and others 1999, equation
// 23, as celestial's research-derived-data.yml has it.
const kingCarbonDioxide = "king-co2"

// index is carbon dioxide's refractive index at a wavelength in
// nanometres, from the sum of terms the data holds (Bideau-Mehu and
// others 1973). It refuses a wavelength outside the range the sum holds
// over.
func (index refractionData) index(nanometres float64) (float64, error) {
	if !(nanometres >= index.Shortest && nanometres <= index.Longest) {
		return 0, fmt.Errorf("airs: %g nm is outside the %g to %g nm the "+
			"carbon dioxide index holds over", nanometres, index.Shortest,
			index.Longest)
	}
	inverse := squareNanometresPerSquareMicrometre / (nanometres * nanometres)
	sum := index.Constant
	for _, term := range index.Terms {
		sum += term[0] / (term[1] - inverse)
	}
	return 1 + sum, nil
}

// molecules is the molecules a cubic metre the index was measured at.
func (index refractionData) molecules() float64 {
	return index.Pressure / (boltzmann * index.Temperature)
}

// carbonDioxideCross is carbon dioxide's Rayleigh cross section at a
// wavelength in nanometres, in square metres a molecule.
func (gas gasData) carbonDioxideCross(nanometres float64) (float64, error) {
	index, err := gas.Refraction.index(nanometres)
	if err != nil {
		return 0, err
	}
	square := index * index
	lorentz := (square - 1) / (square + lorentzShift)
	metres := nanometres * metresPerNanometre
	number := gas.Refraction.molecules()
	return rayleighScale * math.Pow(math.Pi, powerOfPi) * lorentz * lorentz *
		research.Value(kingCarbonDioxide) /
		(math.Pow(metres, powerOfLength) * number * number), nil
}

// standInCross is the cross section celestial gives dry air, at a
// wavelength in nanometres, in square metres a molecule: what stands in
// for nitrogen's, which was not reached. It differs from nitrogen's by a
// few percent, and nitrogen is 3.5 percent of the gas.
func (gas gasData) standInCross(nanometres float64) (float64, error) {
	air, err := atmosphere.DryAir(gas.StandIn.Value)
	if err != nil {
		return 0, err
	}
	area, err := air.CrossSection(measure.Metres(nanometres * metresPerNanometre))
	if err != nil {
		return 0, err
	}
	return area.InSquareMetres(), nil
}

// meanCross is the cross section of the gas's average molecule at a
// wavelength in nanometres: the shares by volume of the two.
func (gas gasData) meanCross(nanometres float64) (float64, error) {
	dioxide, err := gas.carbonDioxideCross(nanometres)
	if err != nil {
		return 0, err
	}
	nitrogen, err := gas.standInCross(nanometres)
	if err != nil {
		return 0, err
	}
	return gas.CarbonDioxide.Value*dioxide + gas.Nitrogen.Value*nitrogen, nil
}

// molarMass is the gas's mean molar mass, in kilograms a mole.
func (gas gasData) molarMass() float64 {
	return gas.CarbonDioxide.Value*gas.CarbonDioxideMass.Value +
		gas.Nitrogen.Value*gas.NitrogenMass.Value
}

// columnMass is the mass of the gas above the ground, to the top of the
// air, a square metre, in kilograms: the density falls by the scale
// height.
func (gas gasData) columnMass() float64 {
	return gas.SurfaceDensity.Value * gas.ScaleHeight.Value *
		(1 - math.Exp(-gas.Top.Value/gas.ScaleHeight.Value))
}

// meanDensity is the density, in kilograms a cubic metre, of a shell
// between two heights: the mass in it over its thickness, so that the
// shells together hold the column's whole mass.
func (gas gasData) meanDensity(bottom, top float64) float64 {
	height := gas.ScaleHeight.Value
	return gas.SurfaceDensity.Value * height *
		(math.Exp(-bottom/height) - math.Exp(-top/height)) / (top - bottom)
}

// scattering is the gas's Rayleigh scattering a metre in a shell between
// two heights, from the cross section of its average molecule.
func (gas gasData) scattering(bottom, top, cross float64) float64 {
	molecules := gas.meanDensity(bottom, top) / gas.molarMass() * avogadro
	return molecules * cross
}
