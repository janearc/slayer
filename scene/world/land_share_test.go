package world

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The middle of a tile well inside land is land and well inside sea is
// sea, by LandShare with the mix the ground is painted with.
func TestLandShareFollowsTheGround(t *testing.T) {
	w := New(terran)
	land, ok := w.Find(maths.V(1, 0, 0), func(b Biome) bool {
		return b == Forest || b == Grass || b == Desert
	})
	if !ok {
		t.Skip("no inland tile in the test world")
	}
	sea, ok := w.Find(maths.V(1, 0, 0), func(b Biome) bool {
		return b == Deep
	})
	if !ok {
		t.Fatal("no deep sea tile in the test world")
	}
	const mix, finest = 0.35, 0.0
	if share := w.LandShare(w.Tiling.Centre(land), finest, mix); share < 0.5 {
		t.Errorf("an inland tile's middle is %v land", share)
	}
	if share := w.LandShare(w.Tiling.Centre(sea), finest, mix); share > 0.5 {
		t.Errorf("a deep sea tile's middle is %v land", share)
	}
}

// With no mix the ground is the coarse tiles' own, and land share is
// wholly land or wholly sea by the tile's class.
func TestLandShareWithNoMixIsTheTilesClass(t *testing.T) {
	w := New(terran)
	for index, tile := range w.tiles {
		want := 0.0
		if Land(w.biome[index]) {
			want = 1
		}
		if got := w.LandShare(w.Tiling.Centre(tile), 0, 0); got != want {
			t.Fatalf("tile %d, %v, is %v land", index, w.biome[index], got)
		}
	}
}

// Anti-test: a world with no sea has no sea in it: every tile that is
// not ice is wholly land.
func TestLandShareHasNoSeaWhereThereIsNone(t *testing.T) {
	dry := terran
	dry.Ocean = 0
	w := New(dry)
	for index, tile := range w.tiles {
		share := w.LandShare(w.Tiling.Centre(tile), 0, 0)
		if share != 1 && w.biome[index] != Ice {
			t.Fatalf("a dry world has %v land at a %v tile", share,
				w.biome[index])
		}
	}
}
