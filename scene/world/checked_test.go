package world

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// a number that is none, or endless either way, is refused by name, for
// each of the five that are checked.
func TestEveryNumberOfATerrainIsCheckedBothWays(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for name, tr := range map[string]Terrain{
			"Ocean":    {Ocean: bad},
			"Cold":     {Cold: bad},
			"Axis's x": {Axis: maths.V(bad, 0, 0)},
			"Axis's y": {Axis: maths.V(0, bad, 0)},
			"Axis's z": {Axis: maths.V(0, 0, bad)},
		} {
			if got := refused(func() { tr.checked() }); !strings.Contains(got, name) {
				t.Errorf("a %s of %v: %q", name, bad, got)
			}
		}
	}
}

// an axis whose longest part is within what Unit can measure is given
// back as it was, on the edges too: 1e-150 and 5e149 are ordinary, and
// so are 1e-100, 1 and 1e100, and no axis at all.
//
// past either edge it is taken to the length of its longest part, so
// (s/3, 2s/9, s) is (1/3, 2/9, 1) for any such s.
func TestAnAxisIsLeftAloneUnlessItIsTooLongOrTooShort(t *testing.T) {
	for _, s := range []float64{1e-150, 1e-100, 1, 1e100, 5e149} {
		axis := maths.V(s, -s/2, s/3)
		if got := (Terrain{Axis: axis}).checked().Axis; got != axis {
			t.Errorf("an axis of %v was changed to %v", axis, got)
		}
	}
	if got := (Terrain{}).checked().Axis; got != (maths.Vec{}) {
		t.Errorf("no axis became %v", got)
	}
	for _, s := range []float64{9e-151, 1e-200, 6e149, 1e300} {
		got := (Terrain{Axis: maths.V(s/3, s/9*2, s)}).checked().Axis
		want := maths.V(1.0/3, 2.0/9, 1)
		if !got.Near(want, 1e-12) {
			t.Errorf("an axis of %v became %v, want %v", s, got, want)
		}
	}
}

// beyond the edge of the world a tile is the deep and minus one, and a tile
// inside it, the first included, is what the world holds. a world with no sea
// has no tile that is the deep or under zero, so the first tile is told from
// beyond the edge.
func TestBeyondTheWorldIsDeep(t *testing.T) {
	flat := hexes.Flat{Step: 1, Across: 6, Down: 6}
	w := New(Terrain{Tiling: flat, Seed: 3, Ocean: 0, Plates: 3})
	out := flat.Nearest(maths.V(500, 0, 500))
	if _, in := w.at[flat.ID(out)]; in {
		t.Fatal("a tile at 500 is in a world six across")
	}
	if w.Biome(out) != Deep || w.Height(out) != -1 {
		t.Errorf("beyond the world is %d at %v", w.Biome(out), w.Height(out))
	}
	for i, tile := range w.Tiles() {
		if w.Biome(tile) != w.biome[i] || w.Height(tile) != w.height[i] {
			t.Fatalf("tile %d is %d at %v, held as %d at %v", i, w.Biome(tile),
				w.Height(tile), w.biome[i], w.height[i])
		}
		if w.Biome(tile) == Deep || w.Height(tile) < 0 {
			t.Fatalf("tile %d of a world with no sea is %d at %v", i,
				w.Biome(tile), w.Height(tile))
		}
	}
}

// a terrain with no tiling is a sphere split once: twelve tiles, the
// twelve corners of the icosahedron.
func TestNoTilingIsTheBareSphere(t *testing.T) {
	w := New(Terrain{Seed: 1, Ocean: 0.5})
	if len(w.Tiles()) != 12 {
		t.Errorf("%d tiles, want 12", len(w.Tiles()))
	}
}
