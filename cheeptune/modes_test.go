package cheeptune

import (
	"math"
	"testing"
)

func TestTheBarRootsAreRootsOfTheFreeBarsEquation(t *testing.T) {
	for i, x := range barRoots {
		// cos(x)cosh(x) = 1, which is cos(x) = 1/cosh(x); written that
		// way the two sides are small and the check is not swamped by
		// cosh.
		if off := math.Cos(x) - 1/math.Cosh(x); math.Abs(off) > 1e-6 {
			t.Errorf("root %d, %v, misses by %g", i+1, x, off)
		}
		// and no root was skipped: each lies near (2n+1) pi over 2.
		near := float64(2*i+3) * math.Pi / 2
		if math.Abs(x-near) > 0.02 {
			t.Errorf("root %d, %v, is not near %v", i+1, x, near)
		}
	}
	// the anti-test: a number that is not a root does not pass.
	if off := math.Cos(4.7) - 1/math.Cosh(4.7); math.Abs(off) < 1e-3 {
		t.Errorf("4.7 passes as a root")
	}
}

func TestTheSetsRatiosAreWhatTheirWorkingGives(t *testing.T) {
	near := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 0.001 {
			t.Errorf("%s is %.4f, want %.4f", name, got, want)
		}
	}
	bar := modeSets["bar"].ratios
	near("the bar's lowest", bar[0], 1)
	near("the bar's second", bar[1], 2.7565)
	near("the bar's third", bar[2], 5.4039)
	tube := modeSets["tube"].ratios
	near("the tube's fourth mode", tube[tubeNote], 2)
	near("the tube's fifth mode", tube[4], 2.9877)
	near("the tube's sixth mode", tube[5], 4.1728)
	ring := modeSets["bowl"].ratios
	near("the ring's lowest", ring[0], 1)
	near("the ring's second", ring[1], 2.8284)
	for name, set := range modeSets {
		if len(set.levels) != len(set.ratios) {
			t.Errorf("%s has %d ratios and %d levels", name,
				len(set.ratios), len(set.levels))
		}
		// no shipped set is a string: some mode is off every whole
		// multiple of the lowest by more than a hundredth.
		uneven := false
		for _, r := range set.ratios {
			ratio := r / set.ratios[0]
			if math.Abs(ratio-math.Round(ratio)) > 0.01 {
				uneven = true
			}
		}
		if !uneven && name != "marimba" {
			t.Errorf("%s has only whole ratios", name)
		}
	}
}
