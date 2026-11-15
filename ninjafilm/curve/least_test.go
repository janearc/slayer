package curve_test

import (
	"testing"

	"pgregory.net/rapid"
)

// A curve's least is the floor it was made with, and no exposure
// develops below it.
func TestACurvesLeastIsItsFloor(t *testing.T) {
	rapid.Check(t, leastIsTheFloor)
}

func leastIsTheFloor(t *rapid.T) {
	tone, least, _, _ := drawn(t)
	if got := tone.Least(); got != least {
		t.Fatalf("a curve made with a floor of %v says %v", least, got)
	}
	if got := tone.Density(exposure(t, "exposure")); got < least {
		t.Fatalf("a curve with a floor of %v developed to %v", least, got)
	}
}

// Anti: the least is not the most, nor the middle. Where no light fell
// the curve is not at its least.
func TestACurvesLeastIsNotItsMost(t *testing.T) {
	rapid.Check(t, leastIsNotMost)
}

func leastIsNotMost(t *rapid.T) {
	tone, _, most, _ := drawn(t)
	if tone.Least() >= most || tone.Density(none) == tone.Least() {
		t.Fatalf("a curve to %v says its least is %v", most, tone.Least())
	}
}
