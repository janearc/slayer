package airs

import (
	"math"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
)

// profile is the 1976 standard atmosphere, celestial's, sampled at the
// heights a stack of shells is cut at: the pressure in pascals and the
// density in kilograms a cubic metre at each one.
type profile struct {
	edges    []float64
	pressure []float64
	density  []float64
}

// standardAt is the standard atmosphere at each of the heights given, in
// metres. It gives the error celestial gives for a height above its top.
func standardAt(edges []float64) (profile, error) {
	sampled := profile{edges: edges}
	for _, height := range edges {
		air, err := atmosphere.Standard(measure.Metres(height))
		if err != nil {
			return profile{}, err
		}
		sampled.pressure = append(sampled.pressure,
			air.Pressure().InPascals())
		sampled.density = append(sampled.density,
			air.Density().InKilogramsPerCubicMetre())
	}
	return sampled, nil
}

// shells is how many shells the profile has: one between each two edges.
func (sampled profile) shells() int {
	return len(sampled.edges) - 1
}

// mass is the mass of the air in a shell, a square metre, in kilograms.
// The density is taken as falling by a constant factor from the shell's
// bottom to its top, as the density sampled at the two ends says, so
// that a shell holds the mass an isothermal one would, and the shells
// together hold the column's.
func (sampled profile) mass(shell int) float64 {
	bottom, top := sampled.density[shell], sampled.density[shell+1]
	thickness := sampled.edges[shell+1] - sampled.edges[shell]
	return (bottom - top) * thickness / math.Log(bottom/top)
}
