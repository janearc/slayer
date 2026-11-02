package world

import "testing"

// every walk over a world's tiles ends: spread, the count from the sea
// and the reach of a feature run until they have no more tiles to go to.
// one that never comes to the end would hold up every test after it, so
// this file is named to run first, as the colour package's is.
//
// each walk is tried on the smallest graph that shows it, a row of two
// or three tiles, since one that loops on a bigger world grows as it
// goes. the row of two, which loops without growing, is tried before
// the row of three, which can grow.
func TestTheWalksOverTilesEnd(t *testing.T) {
	for _, c := range []struct {
		name  string
		n     int
		front []int
	}{
		{"no boundary", 2, nil},
		{"a boundary at the end of a row of two", 2, []int{0}},
		{"two boundaries side by side", 2, []int{0, 1}},
		{"a boundary at the end of a row of three", 3, []int{0}},
	} {
		w := on(chained(c.n, along(c.n)...), 1)
		ends(t, "spread, with "+c.name, func() {
			w.spread(make([]float64, c.n), c.front)
		})
	}
	for _, c := range []struct {
		name   string
		height []float64
	}{
		{"no sea", []float64{0.3, 0.3}},
		{"two tiles of sea side by side", []float64{-0.5, -0.5}},
		{"sea at the end of a row of three", []float64{-0.5, 0.3, 0.3}},
	} {
		w := on(chained(len(c.height), along(len(c.height))...), 1)
		w.height = c.height
		ends(t, "the count from the sea, with "+c.name, func() { w.toSea() })
	}
	w := on(chained(2, along(2)...), 1)
	ends(t, "the reach of a feature", func() { w.within(w.Tiling.Centre(w.tiles[0]), 1) })
}
