package planet

import (
	"math"
	"testing"
)

// speckle eases from cell to cell, round the orbit and across it, and
// meets itself where the angle comes round: a small step anywhere moves
// it a little, never the jump of a hard edge or of a straight cut.
func TestSpeckleHasNoHardEdge(t *testing.T) {
	const radius, steps = 5.0, 31000
	most := 0.0
	last := speckle(1, 0.5, radius, 0)
	for i := 1; i <= steps; i++ {
		turn := 2 * math.Pi * float64(i) / steps
		value := speckle(1, 0.5, radius*math.Cos(turn), radius*math.Sin(turn))
		most = math.Max(most, math.Abs(value-last))
		last = value
	}
	for i := 1; i <= steps; i++ {
		value := speckle(1, 0.5+float64(i)*3/steps, radius, 0)
		most = math.Max(most, math.Abs(value-last))
		last = value
	}
	if most > 0.01 {
		t.Errorf("speckle moves %g in one small step", most)
	}
}

// speckle is a fresh value at each cell: read at cells' middles round
// the orbit, it takes many values, not one.
func TestSpeckleChangesFromCellToCell(t *testing.T) {
	const radius = 5.0
	around := math.Round(2 * math.Pi * radius)
	values := map[float64]bool{}
	for i := 0.0; i < around; i++ {
		turn := (i+0.5)/around*2*math.Pi - math.Pi
		values[speckle(1, 0.5, radius*math.Cos(turn),
			radius*math.Sin(turn))] = true
	}
	if len(values) < int(around)/2 {
		t.Errorf("%d values in %g cells", len(values), around)
	}
}
