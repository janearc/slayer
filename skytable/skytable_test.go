package skytable_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/skytable"
)

// lopsided is a sky that is different every way: red to the east, green
// overhead, blue to the north.
type lopsided struct{}

// Light is the sky's light along a way.
func (lopsided) Light(toward maths.Vec) maths.Light {
	toward = toward.Unit()
	return maths.RGB(0.5+0.5*toward.X, 0.5+0.5*toward.Y,
		0.5-0.5*toward.Z)
}

// the table reads a smooth sky back the way it lies: east red, west not,
// overhead green, north blue, south not, and every way between within
// a hundredth, so a table mirrored, turned or transposed is caught.
func TestTheTableReadsTheSkyTheWayItLies(t *testing.T) {
	table := skytable.Of(lopsided{})
	ways := []maths.Vec{maths.V(1, 0, 0), maths.V(-1, 0, 0),
		maths.V(0, 1, 0), maths.V(0, 0, -1), maths.V(0, 0, 1),
		maths.V(0.3, 0.5, -0.8), maths.V(-0.6, 0.2, 0.4),
		maths.V(0.1, -0.7, 0.2)}
	for _, way := range ways {
		got, want := table.Light(way), lopsided{}.Light(way)
		for _, pair := range [][2]float64{{got.R, want.R},
			{got.G, want.G}, {got.B, want.B}} {
			if !(math.Abs(pair[0]-pair[1]) <= 0.01) {
				t.Errorf("toward %v the table gives %v, the sky %v", way,
					got, want)
				break
			}
		}
	}
}

// with no sky the table is dark every way.
func TestNoSkyIsDark(t *testing.T) {
	table := skytable.Of(nil)
	if got := table.Light(maths.V(0.2, 0.9, 0.1)); got != (maths.Light{}) {
		t.Errorf("no sky gives %v", got)
	}
}

// flat is a sky of one light every way.
type flat struct{ light maths.Light }

// Light is the sky's light along a way, the same every way.
func (sky flat) Light(maths.Vec) maths.Light { return sky.light }

// the cells are the table's size, row by row from nadir to zenith and
// column by column from negative z toward x, and a flat sky fills each
// with its one light.
func TestCellsAreRowByRowFromNadirAndEveryCellOfAFlatSky(t *testing.T) {
	want := maths.Light{R: 0.25, G: 0.5, B: 1}
	cells := skytable.Of(flat{want}).Cells()
	if len(cells) != skytable.Rows*skytable.Columns {
		t.Fatalf("the table has %d cells, want %d", len(cells),
			skytable.Rows*skytable.Columns)
	}
	for place, got := range cells {
		if math.Abs(got.R-want.R) > 1e-12 || math.Abs(got.G-want.G) > 1e-12 ||
			math.Abs(got.B-want.B) > 1e-12 {
			t.Fatalf("cell %d is %v, want %v", place, got, want)
		}
	}
	lopsided := skytable.Of(lopsided{}).Cells()
	nadir, zenith := lopsided[0], lopsided[(skytable.Rows-1)*skytable.Columns]
	if nadir.G > 0.01 || zenith.G < 0.99 {
		t.Errorf("green is %v at the nadir and %v at the zenith, want "+
			"about 0 and 1", nadir.G, zenith.G)
	}
	middle := (skytable.Rows - 1) / 2 * skytable.Columns
	east, west := lopsided[middle+skytable.Columns/4],
		lopsided[middle+3*skytable.Columns/4]
	if east.R < 0.99 || west.R > 0.01 {
		t.Errorf("red is %v east and %v west at the horizon, want about 1 "+
			"and 0", east.R, west.R)
	}
}
