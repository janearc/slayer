package named

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// the gap wall's marble is looked at on a grid of points in a plane, so
// many apart, across a crowd of cities: picked by the author of this
// package, 2026-11-12, to cross several crowds and many cities
const (
	marbleSteps = 400
	marbleStep  = 0.37
)

// The marble lights cities where its crowds are, and is dark between:
// across a stretch of wall some points are lit and some are not, and a
// lit point is lit warm, its red over its blue.
func TestTheMarbleHasCitiesAndDarkBetween(t *testing.T) {
	stone := marble{seed: orbitalSeed}
	lit, dark := nothing, nothing
	for i := range marbleSteps {
		for j := range marbleSteps {
			point := maths.V(float64(i)*marbleStep, float64(j)*marbleStep,
				orbitalRadius)
			light := stone.At(point)
			if light == (maths.Light{}) {
				dark++
				continue
			}
			lit++
			if light.R <= light.B {
				t.Fatalf("a city at %v is lit %v, not warm", point,
					light)
			}
		}
	}
	if lit == nothing || dark == nothing {
		t.Errorf("the marble has %d lit points and %d dark", lit, dark)
	}
}

// anti-tests: two marbles of different seeds are not the same wall.
func TestMarblesOfOtherSeedsDiffer(t *testing.T) {
	one, other := marble{seed: orbitalSeed}, marble{seed: orbitalSeed + 1}
	for i := range marbleSteps {
		point := maths.V(float64(i)*marbleStep, nothing, orbitalRadius)
		if one.At(point) != other.At(point) {
			return
		}
	}
	t.Error("two seeds draw the same wall")
}
