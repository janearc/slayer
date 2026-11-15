package progress_test

import (
	"testing"

	"github.com/janearc/slayer/render/progress"
)

// a band is the rows it was made of, in a picture of the size it was made
// for.
func TestABandKeepsWhatItWasMadeOf(t *testing.T) {
	band, err := progress.NewBand(640, 480, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if band.Width() != 640 || band.Height() != 480 || band.First() != 10 ||
		band.Count() != 20 {
		t.Errorf("got %d by %d, rows %d for %d", band.Width(),
			band.Height(), band.First(), band.Count())
	}
}

// the edges of a picture are inside it: the first row, and a band that
// ends on the last row.
func TestABandMayTouchBothEdgesOfThePicture(t *testing.T) {
	for _, edge := range [][4]int{{1, 1, 0, 1}, {4, 6, 0, 6}, {4, 6, 5, 1}} {
		if _, err := progress.NewBand(edge[0], edge[1], edge[2],
			edge[3]); err != nil {
			t.Errorf("%v: %v", edge, err)
		}
	}
}

// anti-test: no band is made of a picture of no size, of no rows, or of rows
// that begin before the picture or end past it.
func TestNoBandIsMadeOutsideItsPicture(t *testing.T) {
	for what, bad := range map[string][4]int{
		"no width":         {0, 6, 0, 1},
		"no height":        {4, 0, 0, 1},
		"negative width":   {-1, 6, 0, 1},
		"no rows":          {4, 6, 0, 0},
		"negative rows":    {4, 6, 0, -1},
		"before the top":   {4, 6, -1, 2},
		"one past the end": {4, 6, 5, 2},
		"starts past it":   {4, 6, 6, 1},
		"count overflows":  {4, 6, 1, int(^uint(0) >> 1)},
	} {
		if band, err := progress.NewBand(bad[0], bad[1], bad[2],
			bad[3]); err == nil {
			t.Errorf("%s: made %+v", what, band)
		}
	}
}

// a sink is a function that takes a band.
func TestASinkTakesABand(t *testing.T) {
	var got progress.Band
	var sink progress.Sink = func(band progress.Band) { got = band }
	band, _ := progress.NewBand(2, 2, 0, 1)
	sink(band)
	if got != band {
		t.Errorf("the sink got %+v, sent %+v", got, band)
	}
}
