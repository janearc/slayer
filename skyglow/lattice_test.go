package skyglow

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// latticeTolerance is how far a box's corner or a read may be from what
// it is held to: rounding of sums of a few numbers of hundreds of metres.
const latticeTolerance = 1e-9

// latticeField is a field of one source in the city's air on the tests'
// grid, solved to the first order only, which is enough to read.
func latticeField(t *testing.T) *Field {
	t.Helper()
	options := testOptions()
	options.Orders = 1
	streetLamp := aLamp(maths.V(100, 50, -200), 1e6, false)
	field, err := Solve(cityAir(), testRegion(),
		[]Source{streetLamp}, options)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// samePlace says whether two places are the same to the tolerance.
func samePlace(first, second maths.Vec) bool {
	return first.Sub(second).Len() <= latticeTolerance
}

// apart is the most two lights differ in any channel.
func apart(first, second maths.Light) float64 {
	redGap := math.Abs(first.R - second.R)
	greenGap := math.Abs(first.G - second.G)
	blueGap := math.Abs(first.B - second.B)
	return math.Max(redGap, math.Max(greenGap, blueGap))
}

// The box the cells fill is the region the field was solved over, and the
// counts are the grid's.
func TestCellsFillTheRegionSolvedOver(t *testing.T) {
	field := latticeField(t)
	low, high, count := field.Cells()
	region := testRegion()
	if !samePlace(low, region.Low) || !samePlace(high, region.High) {
		t.Errorf("cells fill %v to %v, the region is %v to %v", low, high,
			region.Low, region.High)
	}
	if count != testNodes {
		t.Errorf("%v cells, the grid is %v", count, testNodes)
	}
}

// Mean at the middle of a cell is that cell's value, so a drawer that
// reads between the middles by the same weights reads what Mean does.
func TestMeanAtTheMiddleOfACellIsTheCell(t *testing.T) {
	field := latticeField(t)
	low, high, count := field.Cells()
	span := high.Sub(low)
	for _, place := range [][3]int{{0, 0, 0}, {5, 2, 9}, {11, 5, 11}} {
		shiftX := span.X * (float64(place[0]) + half) / float64(count[0])
		shiftY := span.Y * (float64(place[1]) + half) / float64(count[1])
		shiftZ := span.Z * (float64(place[2]) + half) / float64(count[2])
		middle := low.Add(maths.V(
			shiftX,
			shiftY,
			shiftZ))
		want := field.Cell(place[0], place[1], place[2])
		got := field.Mean(middle)
		if apart(got, want) > latticeTolerance*want.Luminance() {
			t.Errorf("at cell %v Mean is %v and the cell %v", place, got,
				want)
		}
	}
}

// A place beyond the grid on any side is the cell at its edge, as Mean
// holds the edge value: no cell past the grid is read.
func TestACellBeyondTheGridIsTheEdge(t *testing.T) {
	field := latticeField(t)
	_, _, count := field.Cells()
	last := [3]int{count[0] - 1, count[1] - 1, count[2] - 1}
	if field.Cell(-1, 3, 4) != field.Cell(0, 3, 4) ||
		field.Cell(2, -3, 4) != field.Cell(2, 0, 4) ||
		field.Cell(2, 3, -9) != field.Cell(2, 3, 0) {
		t.Error("a cell below the grid is not the edge cell")
	}
	if field.Cell(count[0], 3, 4) != field.Cell(last[0], 3, 4) ||
		field.Cell(2, count[1]+2, 4) != field.Cell(2, last[1], 4) ||
		field.Cell(2, 3, count[2]) != field.Cell(2, 3, last[2]) {
		t.Error("a cell above the grid is not the edge cell")
	}
}
