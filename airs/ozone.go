package airs

import (
	"fmt"
	"math"
)

// Units of the ozone's data, none of them a pick.
const (
	// pascalsPerHectopascal turns the layers' pressures into pascals.
	pascalsPerHectopascal = 100.0 // defined: the prefix
	// squareMetresPerSquareCentimetre turns the cross sections, which a
	// source prints in square centimetres, into square metres.
	squareMetresPerSquareCentimetre = 1e-4 // defined: the unit
	// dobsonThickness is the thickness in metres of a Dobson unit of
	// gas: a hundredth of a millimetre at standard conditions.
	dobsonThickness = 1e-5 // defined: the Dobson unit
	// standardPressure is in pascals, the standard atmosphere's, the
	// pressure a Dobson unit is measured at.
	standardPressure = 101325.0 // defined: the standard atmosphere
	// standardTemperature is in kelvin, the freezing point of water, the
	// temperature a Dobson unit is measured at.
	standardTemperature = 273.15 // defined: 0 degrees Celsius
	// moleculesPerDobson is the molecules in a square metre of one
	// Dobson unit, by the ideal gas law: 2.687e20, Loschmidt's number
	// in a column a hundredth of a millimetre thick.
	moleculesPerDobson = dobsonThickness * standardPressure /
		(boltzmann * standardTemperature)
)

// cross is the ozone's absorption cross section at a wavelength in
// nanometres, in square metres a molecule: the table's, which is in
// square centimetres, straight between points in its logarithm. It
// refuses a wavelength outside the table's.
func (ozone ozoneData) cross(nanometres float64) (float64, error) {
	points := ozone.CrossSection.Points
	shortest, longest := points[0][0], points[len(points)-1][0]
	if !(nanometres >= shortest && nanometres <= longest) {
		return 0, fmt.Errorf("airs: %g nm is outside the %g to %g nm the "+
			"ozone cross sections hold over", nanometres, shortest, longest)
	}
	index, fraction := ozone.CrossSection.locate(nanometres)
	low, high := points[index][1], points[index+1][1]
	return low * math.Pow(high/low, fraction) *
		squareMetresPerSquareCentimetre, nil
}

// amount is the ozone in a shell, in Dobson units, between the pressures
// at its bottom and top in pascals: each layer of the source's gives the
// share of its own that lies in the shell, by pressure, which is by the
// mass of air, as the layers are of one mixing ratio.
func (ozone ozoneData) amount(bottom, top float64) float64 {
	total := 0.0
	for _, layer := range ozone.Layers {
		layerBottom := layer.Bottom * pascalsPerHectopascal
		layerTop := layer.Top * pascalsPerHectopascal
		overlap := math.Max(0, math.Min(bottom, layerBottom)-
			math.Max(top, layerTop))
		total += layer.Amount * overlap / (layerBottom - layerTop)
	}
	return total
}
