package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The plain street, from the culture: 12 m wide, 96 m long, six storeys of
// a 4.4 m ground and 3.2 m above it.
const (
	plainWidth  = 12.0
	plainLength = 96.0
	plainHeight = 4.4 + 5*3.2
)

// rowsOf is the plain rows of an ask and seed, or the test's end.
func rowsOf(t *testing.T, ask RowsAsk, seed Seed) *PlainRows {
	t.Helper()
	rows, err := NewPlainRows(ask, seed)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestPlainRowsFaceEachOtherAcrossTheCulturesStreet(t *testing.T) {
	rows := rowsOf(t, RowsAsk{}, loneSeed)
	west, east := math.Inf(-1), math.Inf(1)
	for _, volume := range rows.Volumes {
		if volume.High[crossingX] <= 0 {
			west = math.Max(west, volume.High[crossingX])
		} else {
			east = math.Min(east, volume.Low[crossingX])
		}
		if math.Abs(volume.High[heightAxis]-plainHeight) > closeEnough {
			t.Errorf("a block is %v m high, not %v", volume.High[heightAxis],
				plainHeight)
		}
	}
	if math.Abs(east-west-plainWidth) > closeEnough {
		t.Errorf("the street is %v m wide, not %v", east-west, plainWidth)
	}
}

func TestEachRowCoversTheLengthOfTheStreetWithoutAGap(t *testing.T) {
	rows := rowsOf(t, RowsAsk{}, loneSeed)
	for _, side := range []func(Volume) bool{
		func(volume Volume) bool { return volume.High[crossingX] <= 0 },
		func(volume Volume) bool { return volume.Low[crossingX] >= 0 }} {
		end := -plainLength / 2
		for _, volume := range rows.Volumes {
			if !side(volume) {
				continue
			}
			if math.Abs(volume.Low[crossingZ]-end) > closeEnough {
				t.Errorf("a block starts at %v, the last ended at %v",
					volume.Low[crossingZ], end)
			}
			end = volume.High[crossingZ]
		}
		if math.Abs(end-plainLength/2) > closeEnough {
			t.Errorf("a row ends at %v, not %v", end, plainLength/2)
		}
	}
}

func TestPlainRowsTakeTheWidthAndLengthTheyAreAsked(t *testing.T) {
	rows := rowsOf(t, RowsAsk{Width: 20, Length: 50}, loneSeed)
	low, high := math.Inf(1), math.Inf(-1)
	gap := math.Inf(1)
	for _, volume := range rows.Volumes {
		low, high = math.Min(low, volume.Low[crossingZ]),
			math.Max(high, volume.High[crossingZ])
		gap = math.Min(gap, math.Abs(volume.Low[crossingX]))
		gap = math.Min(gap, math.Abs(volume.High[crossingX]))
	}
	if math.Abs(high-low-50) > closeEnough || math.Abs(gap-10) > closeEnough {
		t.Errorf("a street 50 m long and 20 wide came out %v long and %v "+
			"to its walls", high-low, 2*gap)
	}
}

func TestPlainRowsAreTheSameForTheSameSeed(t *testing.T) {
	one, other := rowsOf(t, RowsAsk{}, loneSeed), rowsOf(t, RowsAsk{}, loneSeed)
	if len(one.Volumes) != len(other.Volumes) || len(one.Design().Elements) !=
		len(other.Design().Elements) {
		t.Errorf("one seed drew %d blocks, then %d", len(one.Volumes),
			len(other.Volumes))
	}
}

func TestPlainRowsHaveLitAndDarkPanes(t *testing.T) {
	rows := rowsOf(t, RowsAsk{}, loneSeed)
	lit, dark := 0, 0
	for _, pane := range solidsNamed(rows.Design(), "pane") {
		if pane.Material.Glow != (maths.Light{}) {
			lit++
		} else {
			dark++
		}
	}
	if lit == 0 || dark == 0 {
		t.Errorf("%d panes lit and %d dark", lit, dark)
	}
}

// Anti: a width or a length below nought is refused.
func TestPlainRowsRefuseANegativeStreet(t *testing.T) {
	for _, ask := range []RowsAsk{{Width: -1}, {Length: -1}} {
		if _, err := NewPlainRows(ask, loneSeed); err == nil {
			t.Errorf("the street %v was taken", ask)
		}
	}
}

func TestADefaultDensityIsFromNoughtToOne(t *testing.T) {
	density, err := DefaultDensity()
	if err != nil {
		t.Fatal(err)
	}
	if !(density >= 0 && density <= 1) {
		t.Errorf("the default density is %v", density)
	}
}
