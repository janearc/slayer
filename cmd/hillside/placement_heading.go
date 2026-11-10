package main

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// placementZero is defined as no angle or no seed index.
const placementZero = 0

// placementTurnSteps is defined as tenth-degrees in one full turn.
const placementTurnSteps = 3600

// fullPlacementTurn is defined as two half-turns.
const fullPlacementTurn = 2

// placementTurn preserves seeded poses unless a named placement says otherwise.
// Optical recipe angles already describe their local assembly.
func (stage *staging) placementTurn(kind, name string,
	seed, index uint64,
) float64 {
	if name != "" {
		for _, object := range stage.recipes.Things() {
			if object.Name() == name {
				if angle, said := object.Heading(); said {
					return -angle * maths.Degree
				}
				break
			}
		}
	}
	if kind == "optical-observatory" {
		return placementZero
	}
	ordinal := int(index)
	picked := stir(seed, ordinal) % placementTurnSteps
	return float64(picked) / placementTurnSteps * fullPlacementTurn * math.Pi
}
