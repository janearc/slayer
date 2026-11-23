package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// A ray made from two vectors has the first for its start and the second
// for its direction, and nothing is done to either.
func TestNewRayKeepsItsStartAndDirection(test *testing.T) {
	start, along := vector(test, "a"), vector(test, "b")
	got := maths.NewRay(start, along)
	exactly(test, "start", got.At, start)
	exactly(test, "direction, not made a unit", got.Dir, along)
}

// A ray is not the same ray with its start and direction swapped.
func TestNewRayOrderMatters(test *testing.T) {
	start, along := vector(test, "a"), vector(test, "b")
	if maths.NewRay(start, along) == maths.NewRay(along, start) {
		test.Errorf("a ray from %v along %v is the swapped ray", start, along)
	}
}

// A box made from two vectors has the first for its Min and the second
// for its Max, and a swapped pair is not put right.
func TestNewBoxKeepsItsCornersAsGiven(test *testing.T) {
	lesser, greater := vector(test, "a"), vector(test, "b")
	got := maths.NewBox(lesser, greater)
	exactly(test, "min", got.Min, lesser)
	exactly(test, "max", got.Max, greater)
	swapped := maths.NewBox(greater, lesser)
	exactly(test, "min of a swapped pair", swapped.Min, greater)
	exactly(test, "max of a swapped pair", swapped.Max, lesser)
}

// A box of the same corners in the other order is another box.
func TestNewBoxOrderMatters(test *testing.T) {
	lesser, greater := vector(test, "a"), vector(test, "b")
	if maths.NewBox(lesser, greater) == maths.NewBox(greater, lesser) {
		test.Errorf("a box from %v to %v is the swapped box", lesser, greater)
	}
}

// A band made from a wavelength carries exactly that number, and zero is
// the band of every wavelength.
func TestNewBandKeepsItsWavelength(test *testing.T) {
	wavelength := picked(test, "half")
	if got := maths.NewBand(wavelength); got.Nanometres != wavelength {
		test.Errorf("band of %v is %v", wavelength, got)
	}
	var every maths.Band
	if got := maths.NewBand(nought); got != every {
		test.Errorf("band of nought is %v, not the zero band", got)
	}
}

// Two different wavelengths are two different bands.
func TestNewBandDiffersByWavelength(test *testing.T) {
	if maths.NewBand(picked(test, "half")) == maths.NewBand(picked(test, "doubling")) {
		test.Errorf("two wavelengths made the same band")
	}
}
