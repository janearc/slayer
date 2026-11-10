package fuelbed_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/fuelbed"
)

// Reaction is also a public scalar helper. An inactive state does not need
// an exponent: preserve ambient zero even for a zero activation barrier,
// and keep an empty cell inert when its temperature is unavailable.
func TestInactiveReactionDoesNotEvaluateAnUndefinedExponent(t *testing.T) {
	for _, test := range []struct {
		name       string
		cell       fuelbed.Cell
		activation float64
	}{
		{"ambient without activation barrier", fuelbed.Cell{Fuel: 1}, 0},
		{"empty with missing temperature", fuelbed.Cell{Excess: math.NaN()}, 550},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := fuelbed.Reaction(test.cell, test.activation); got != 0 {
				t.Fatalf("inactive reaction = %v, want zero", got)
			}
		})
	}
	if got := fuelbed.Reaction(fuelbed.Cell{Excess: 1, Fuel: 1}, 0); got != 1 {
		t.Fatalf("active barrier-free reaction = %v, want one", got)
	}
}
