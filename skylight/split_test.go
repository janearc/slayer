package skylight

import (
	"fmt"
	"math"
	"testing"
)

// The shell that is split.
const (
	splitDepth  = 0.3  // picked: enough air that most photons scatter
	splitGround = 0.3  // picked: a ground that sends some back
	splitAt     = 3    // picked: a third of the way up
	splitShare  = 1e-9 // picked: the split only moves positions by
	// roundings, and a photon whose path it changes would move the
	// budget by one part in the photons, ten thousand times more
	splitPhotons = 5000 // picked: a fraction of a second
)

// splitWalk is the budget of one uniform shell of gas walked with the
// grid's kept heights, and its transmittance halfway up at sixty
// degrees.
func splitWalk(t *testing.T, column Column, heights []float64) (Budget,
	float64) {
	t.Helper()
	grid := groundGrid()
	grid.Heights = heights
	field, spent, err := Solve(column, grid, splitPhotons, seedOf,
		workersOf)
	if err != nil {
		t.Fatal(err)
	}
	return spent, field.Transmittance(airTop/two, sixty*degree)
}

// sixty is a sun well above the horizon, in degrees.
const sixty = 60 // picked: well above the horizon

// alike is nil when two budgets agree, end by end, to a share of what
// was launched.
func alike(first, second Budget) error {
	ends := [][two]float64{{first.Escaped, second.Escaped},
		{first.Air, second.Air}, {first.Ground, second.Ground},
		{first.Capped, second.Capped}}
	for _, pair := range ends {
		gap := math.Abs(pair[none] - pair[one])
		if !(gap <= splitShare*first.Launched) {
			return fmt.Errorf("%+v and %+v differ", first, second)
		}
	}
	return nil
}

// splitColumn is one uniform shell of gas over a grey ground.
func splitColumn() Column {
	gas := Shell{GasScattering: splitDepth / airTop}
	return uniformColumn(earthRadius, airTop, gas, splitGround)
}

// a height kept inside a shell splits it there and changes nothing in
// the air: the same walk gives the same budget, and the sun the same
// share.
func TestSplittingAShellChangesNothing(t *testing.T) {
	whole, wholeSun := splitWalk(t, splitColumn(), []float64{none})
	split, splitSun := splitWalk(t, splitColumn(),
		[]float64{none, airTop / splitAt})
	if err := alike(whole, split); err != nil {
		t.Error(err)
	}
	if !(math.Abs(wholeSun-splitSun) <= rounding) {
		t.Errorf("the sun's share moved from %g to %g", wholeSun, splitSun)
	}
}

// the anti-test: two shells whose upper one holds twice the gas give
// another budget, so the check would see a split that changed the air.
func TestAChangedSplitIsSeen(t *testing.T) {
	whole, _ := splitWalk(t, splitColumn(), []float64{none})
	changed := splitColumn()
	upper := changed.Shells[none]
	changed.Shells[none].Top = airTop / splitAt
	upper.Bottom, upper.GasScattering = airTop/splitAt,
		two*upper.GasScattering
	changed.Shells = append(changed.Shells, upper)
	split, _ := splitWalk(t, changed, []float64{none})
	if alike(whole, split) == nil {
		t.Error("a changed split gives the same budget")
	}
}
