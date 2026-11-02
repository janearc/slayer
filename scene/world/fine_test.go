package world

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/texture"
)

// A fine rung paints every fine tile from the coarse tile it sits in:
// with no vary and no mix it is the coarse painting exactly, and with
// them it differs from place to place inside one coarse tile while most
// of the tile keeps its class.
func TestFineFollowsTheCoarseWorld(t *testing.T) {
	w := New(terran)
	var look Look
	for b := range Biomes {
		look.Paints[b] = maths.Grey(0.1 + 0.06*float64(b))
	}
	plain := w.Fine(look, 3, 0, 0)
	coarse := w.Painted(look)
	fine := w.Tiling.Finer().Finer().Finer()
	for i, tile := range fine.Tiles() {
		if i%97 != 0 {
			continue
		}
		c := fine.Centre(tile)
		if plain.At(c) != coarse.At(c) {
			t.Fatalf("fine tile %v is %v, the coarse world %v", tile,
				plain.At(c), coarse.At(c))
		}
	}
	varied := w.Fine(look, 3, 0.3, 0.3)
	// inside one coarse tile of grass, the fine tiles vary and most are
	// still grass-coloured
	grass, ok := w.Find(maths.V(0.6, 0.2, 0.8), func(b Biome) bool { return b == Grass })
	if !ok {
		t.Skip("no grass on this world")
	}
	seen, same, n := map[maths.Light]bool{}, 0, 0
	for _, ft := range fine.(hexes.Sphere).Tiles() {
		c := fine.Centre(ft)
		if w.Tiling.Nearest(c) != grass {
			continue
		}
		n++
		p := varied.At(c)
		seen[p] = true
		if p.R > look.Paints[Grass].R*0.7 && p.R < look.Paints[Grass].R*1.3 {
			same++
		}
	}
	if n < 10 || len(seen) < n/2 || same < n/2 {
		t.Errorf("%d fine tiles in a grass tile, %d colours, %d near grass",
			n, len(seen), same)
	}
}

// Left to choose, a fine texture picks a rung by the pixel asking: a big
// footprint gets the world's own rung, a small one a rung where a tile
// is about Across footprints across, and no footprint the world's own.
func TestFineChoosesItsRungByTheFootprint(t *testing.T) {
	w := New(terran)
	var look Look
	for b := range Biomes {
		look.Paints[b] = maths.Grey(0.1 + 0.06*float64(b))
	}
	look.Lines = 0.5
	auto := w.Fine(look, 0, 0, 0).(texture.Footprinted)
	coarse := w.Painted(look)
	p := maths.V(0.3, 0.5, 0.8).Unit()
	big := w.Tiling.Spacing() * 2
	if auto.Within(p, big) != coarse.At(p) || auto.At(p) != coarse.At(p) {
		t.Error("a big footprint, or none, is not the coarse world")
	}
	// a small footprint: the same class, but lines fall where the fine
	// rung's lines are, so across a coarse tile the paint changes often
	small := w.Tiling.Spacing() / 200
	changes, last := 0, maths.Light{}
	tile := w.Tiling.Nearest(p)
	c := w.Tiling.Centre(tile)
	for i := 0; i < 400; i++ {
		q := c.Add(maths.V(float64(i)*w.Tiling.Spacing()/800, 0, 0)).Unit()
		if w.Tiling.Nearest(q) != tile {
			break
		}
		got := auto.Within(q, small)
		if i > 0 && got != last {
			changes++
		}
		last = got
	}
	if changes < 10 {
		t.Errorf("across a coarse tile a small footprint changed the paint %d times", changes)
	}
}
