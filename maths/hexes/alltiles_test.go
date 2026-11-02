package hexes

import "testing"

// this file's name sorts ahead of audit_test.go on purpose. go test runs
// files in name order and stops at the first failure, so a change that
// empties Tiles fails here, quickly, before the audit's tiles of no size
// are reached, which an emptied loop would never leave.

// a flat extent of Across by Down holds that many tiles, in order of id, row by
// row, and allocates only what it holds: a size less than zero holds none and
// allocates none.
func TestFlatTilesAreRowByRowAndAllocateWhatTheyHold(t *testing.T) {
	f := Flat{Step: 1, Across: 4, Down: 3}
	tiles := f.Tiles()
	if len(tiles) != 12 {
		t.Fatalf("%d tiles", len(tiles))
	}
	for k, tile := range tiles {
		if tile != (Tile{0, k % 4, k / 4}) {
			t.Fatalf("tile %d is %v", k, tile)
		}
	}
	for _, g := range []Flat{f, {Step: 1, Across: -2, Down: 3},
		{Step: 1, Across: 3, Down: -1}, {Step: 1, Across: 0, Down: 0},
		{Step: 1, Across: 5, Down: 0}} {
		got := g.Tiles()
		if cap(got) != len(got) {
			t.Errorf("%+v: %d tiles in room for %d", g, len(got), cap(got))
		}
	}
}

// a band allocates only the tiles it holds, row by row.
func TestBandTilesAreRowByRowAndAllocateWhatTheyHold(t *testing.T) {
	b := Band{Radius: 3, Width: 2, Around: 24}
	tiles := b.Tiles()
	for k, tile := range tiles {
		if tile != (Tile{0, k % 24, k / 24}) {
			t.Fatalf("tile %d is %v", k, tile)
		}
	}
	if cap(tiles) != len(tiles) {
		t.Errorf("%d tiles in room for %d", len(tiles), cap(tiles))
	}
}
