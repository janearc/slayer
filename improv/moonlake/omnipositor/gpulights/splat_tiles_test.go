//go:build draw

package gpulights

import "testing"

func TestSplatTilesPreserveOrderAndBounds(t *testing.T) {
	splats := []Splat{{X: 1, Y: 1, Sigma: 1}, {X: 31, Y: 1, Sigma: 2}, {X: 64, Y: 1, Sigma: 1}, {X: -100, Y: -100, Sigma: 1}}
	tiles, err := binSplats(96, 32, splats)
	if err != nil {
		t.Fatal(err)
	}
	if tiles.columns != 3 || tiles.rows != 1 {
		t.Fatalf("tiles: %+v", tiles)
	}
	want := [][]uint32{{0, 1}, {1, 2}, {2}}
	for index, expected := range want {
		got := tiles.indices[tiles.offsets[index]:tiles.offsets[index+1]]
		if len(got) != len(expected) {
			t.Fatalf("tile %d: %v", index, got)
		}
		for item := range got {
			if got[item] != expected[item] {
				t.Fatalf("tile %d: %v", index, got)
			}
		}
	}
}
