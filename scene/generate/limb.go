package generate

import "math"

// limbPoint is a limb-darkening coefficient at a temperature.
type limbPoint struct{ kelvin, coefficient float64 }

// limbTable is the linear limb-darkening coefficient u of a star's disc
// in visible light, by surface temperature: the Sun's near 0.65, an
// A star's near 0.5, a hot white dwarf's near 0.3. Set by eye from
// memory of Claret's tables, 2026-11-26, to shelve: the coarse trend is
// right, the second digit is not claimed.
var limbTable = []limbPoint{
	{5800, 0.65}, {9500, 0.50}, {15000, 0.38}, {25000, 0.30},
}

// LimbCoefficient is the linear limb-darkening coefficient of a star's
// disc at a surface temperature in kelvin: the table's, joined by
// straight lines in the log of the temperature, and held at its ends
// above and below. It is between nought and one.
func LimbCoefficient(kelvin float64) float64 {
	last := len(limbTable) - 1
	if !(kelvin > limbTable[0].kelvin) {
		return limbTable[0].coefficient
	}
	for index := 1; index <= last; index++ {
		low, high := limbTable[index-1], limbTable[index]
		if kelvin <= high.kelvin {
			along := math.Log(kelvin/low.kelvin) /
				math.Log(high.kelvin/low.kelvin)
			return low.coefficient + along*(high.coefficient-low.coefficient)
		}
	}
	return limbTable[last].coefficient
}
