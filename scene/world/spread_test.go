package world

import (
	"math"
	"testing"
)

// a world is about as many tiles across as the square root of a tenth of
// its tiles: 1000 tiles are 10 across and 40 are 2.
func TestAWorldIsAboutASplitAcross(t *testing.T) {
	for _, c := range []struct {
		tiles int
		want  float64
	}{{1000, 10}, {40, 2}, {250, 5}} {
		if got := on(chained(c.tiles), 1).across(); got != c.want {
			t.Errorf("%d tiles are %v across, want %v", c.tiles, got, c.want)
		}
	}
}

// the effect of a boundary carried inland along a row of tiles 0 to 4, in a
// world of ten tiles, one across, where the fall is the least it can be, 1.5
// tiles. tile k has the push times e to the -k / 1.5, and a tile no boundary
// reaches has zero.
func TestASpreadFadesWithTilesFromTheBoundary(t *testing.T) {
	w := on(chained(10, along(5)...), 1)
	push := make([]float64, 10)
	push[0] = 2
	var got []float64
	ends(t, "a spread from one end", func() { got = w.spread(push, []int{0}) })
	for k := range got {
		want := 0.0
		if k < 5 {
			want = 2 * math.Exp(-float64(k)/1.5)
		}
		if !close12(got[k], want) {
			t.Errorf("tile %d is %v, want %v", k, got[k], want)
		}
	}
}

// tiles reached from two boundaries take the nearer, and the first of
// them when they are as near. along the row 0 to 4 with boundaries at the
// two ends, of push 1 and -3, tile 1 is 1 from the first, tile 3 is 1
// from the other, and tile 2 is 2 from both and takes the first.
//
// tiles on a boundary keep their own push even beside another.
func TestASpreadTakesTheNearerBoundary(t *testing.T) {
	w := on(chained(10, along(5)...), 1)
	push := make([]float64, 10)
	push[0], push[4] = 1, -3
	var got []float64
	ends(t, "a spread from both ends", func() { got = w.spread(push, []int{0, 4}) })
	f := math.Exp(-1 / 1.5)
	want := []float64{1, f, f * f, -3 * f, -3, 0, 0, 0, 0, 0}
	for k := range want {
		if !close12(got[k], want[k]) {
			t.Errorf("tile %d is %v, want %v", k, got[k], want[k])
		}
	}
	two := on(chained(2, [2]int{0, 1}), 1)
	ends(t, "a spread between neighbours", func() {
		got = two.spread([]float64{1, -3}, []int{0, 1})
	})
	if got[0] != 1 || got[1] != -3 {
		t.Errorf("two boundary tiles side by side are %v, want 1 and -3", got)
	}
}

// the fall is a sixth of how many tiles across a big world is, once that
// is over a tile and a half. 900 tiles are sqrt(90) across, 9.4868, so
// the fall is sqrt(90) / 6 = 1.5811.
func TestTheFallOfABigWorldIsASixthOfItsWidth(t *testing.T) {
	w := on(chained(900, along(3)...), 1)
	push := make([]float64, 900)
	push[0] = 2
	var got []float64
	ends(t, "a spread in a big world", func() { got = w.spread(push, []int{0}) })
	fall := math.Sqrt(90) / 6
	for k := 1; k < 3; k++ {
		if want := 2 * math.Exp(-float64(k)/fall); !close12(got[k], want) {
			t.Errorf("tile %d is %v, want %v", k, got[k], want)
		}
	}
}

// the tiles away from the sea, counted in tiles. tiles 0 and 1 are under
// the sea and next to each other, tile 3 is exactly at the water line and
// is dry, tile 4 is land past it. the sea is 0 for both its tiles, and
// tiles 2, 3 and 4 are 1, 2 and 3 from it.
//
// tiles 5 and 6 are land with no sea in reach, as far as the world has
// tiles, 7.
func TestTilesAreCountedFromTheSea(t *testing.T) {
	w := on(chained(7, [2]int{0, 1}, [2]int{1, 2}, [2]int{2, 3}, [2]int{3, 4},
		[2]int{5, 6}), 1)
	w.height = []float64{-0.5, -0.1, 0.3, 0, 0.5, 0.2, 0.4}
	var got []int
	ends(t, "a count from the sea", func() { got = w.toSea() })
	want := []int{0, 0, 1, 2, 3, 7, 7}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tile %d is %d from the sea, want %d", i, got[i], want[i])
		}
	}
}
