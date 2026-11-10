package fuelbed_test

import (
	"testing"
)

func TestExcessTemperatureEvolutionIsIndependentOfAbsoluteReference(t *testing.T) {
	s := fixtureWithHot()
	a := must(t, s, 5)
	s.Ambient = 400
	b := must(t, s, 5)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("cell %d evolved absolute rather than excess temperature: %v %v", i, a[i], b[i])
		}
	}
	// The caller converts an absolute ignition temperature to excess.
	// With the same absolute ignition, a warmer reference starts cooler
	// in these coordinates and must not reproduce the first evolution.
	s.Initial[4].Excess = 800
	c := must(t, s, 5)
	if c[4].Excess == a[4].Excess || c[4].Fuel == a[4].Fuel {
		t.Fatal("reference conversion had no thermal or reaction effect")
	}
}
