package world

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// A volcano on land: its heart is lava and higher than the ground was,
// its rim is higher still, its slopes are ash, and a tile two reaches
// away is as it was.
func TestAVolcanoRisesFromTheLand(t *testing.T) {
	plain := New(terran)
	heart, ok := plain.Find(maths.V(0.6, 0.2, 0.8), Land)
	if !ok {
		t.Fatal("no land to put a volcano on")
	}
	at := plain.Tiling.Centre(heart)
	reach := 4 * plain.Tiling.Spacing()
	with := terran
	with.Features = []Feature{Volcano{At: at, Reach: reach, Rise: 0.5}}
	w := New(with)
	if w.Biome(heart) != Lava {
		t.Errorf("the heart is %d, not lava", w.Biome(heart))
	}
	if !(w.Height(heart) > plain.Height(heart)) {
		t.Errorf("the heart is %v, was %v", w.Height(heart), plain.Height(heart))
	}
	rim, ash := false, 0
	for _, nb := range w.Tiling.Neighbours(heart) {
		rim = rim || w.Height(nb) > w.Height(heart)
	}
	for _, tile := range w.Tiles() {
		if w.Biome(tile) == Ash {
			ash++
		}
	}
	if !rim || ash == 0 {
		t.Errorf("rim higher than the heart: %v; ash tiles: %d", rim, ash)
	}
	far, _ := plain.Find(at.Neg(), Land)
	if w.Height(far) != plain.Height(far) || w.Biome(far) != plain.Biome(far) {
		t.Error("the far side of the world changed")
	}
}

// A canyon cuts a line of rock lower than the ground beside it.
func TestACanyonCutsALine(t *testing.T) {
	plain := New(terran)
	a, ok := plain.Find(maths.V(0.6, 0.2, 0.8), Land)
	if !ok {
		t.Fatal("no land")
	}
	from := plain.Tiling.Centre(a)
	to := from.Add(maths.V(0.25, 0, 0.1)).Unit()
	with := terran
	with.Features = []Feature{Canyon{From: from, To: to,
		Width: 1.2 * plain.Tiling.Spacing(), Depth: 0.4}}
	w := New(with)
	rock, lower := 0, 0
	for _, tile := range w.Tiles() {
		if w.Biome(tile) == Rock && plain.Biome(tile) != Rock {
			rock++
			if w.Height(tile) < plain.Height(tile) {
				lower++
			}
		}
	}
	if rock < 3 || lower < rock {
		t.Errorf("%d new rock tiles, %d of them lower", rock, lower)
	}
}

// An ice shelf freezes the sea near a coast to ice just above the water,
// and leaves the land it touches alone.
func TestAShelfFreezesTheSea(t *testing.T) {
	plain := New(terran)
	coast, ok := plain.Find(maths.V(0.1, 0.95, 0.1), Water)
	if !ok {
		t.Fatal("no sea near the pole")
	}
	at := plain.Tiling.Centre(coast)
	with := terran
	with.Features = []Feature{Shelf{At: at, Reach: 3 * plain.Tiling.Spacing()}}
	w := New(with)
	ice := 0
	for _, i := range w.within(at, 3*plain.Tiling.Spacing()) {
		tile := w.tiles[i]
		// below the water line is sea, whether open or already iced
		if plain.Height(tile) < 0 {
			if w.Biome(tile) != Ice || w.Height(tile) < 0 {
				t.Fatalf("sea at %v became %d at %v", tile, w.Biome(tile),
					w.Height(tile))
			}
			ice++
		} else if w.Height(tile) != plain.Height(tile) {
			t.Fatalf("land at %v moved", tile)
		}
	}
	if ice == 0 {
		t.Error("no sea froze")
	}
}

// A plateau is flat on top and rock at its edge, and a feature on a
// plane works as one on a sphere does.
func TestAPlateauOnAPlane(t *testing.T) {
	tr := Terrain{Tiling: hexes.Flat{Step: 1, Across: 30, Down: 30},
		Seed: 3, Ocean: 0.4, Plates: 5}
	plain := New(tr)
	at := maths.V(15, 0, 15)
	tr.Features = []Feature{Plateau{At: at, Reach: 5, Top: 0.7}}
	w := New(tr)
	flat, rock := 0, 0
	for _, i := range w.within(at, 5) {
		tile := w.tiles[i]
		if w.Height(tile) >= 0.7-1e-12 {
			flat++
		}
		if w.Biome(tile) == Rock && plain.Biome(tile) != Rock {
			rock++
		}
	}
	if flat < 20 || rock == 0 {
		t.Errorf("%d tiles on the flat, %d new rock at the edge", flat, rock)
	}
}
