package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

var terran = Terrain{Tiling: hexes.Sphere{N: 14}, Seed: 5, Ocean: 0.6,
	Plates: 11, Axis: maths.V(0.2, 1, 0)}

// A world asked for sixty percent sea gets sixty percent, and its land
// tops out at one and its sea floor at minus one.
func TestTheSeaIsPouredToTheShareAsked(t *testing.T) {
	w := New(terran)
	under, top, bottom := 0, 0.0, 0.0
	for _, tile := range w.Tiles() {
		h := w.Height(tile)
		if h < 0 {
			under++
		}
		top, bottom = math.Max(top, h), math.Min(bottom, h)
	}
	share := float64(under) / float64(len(w.Tiles()))
	if math.Abs(share-0.6) > 0.02 {
		t.Errorf("%.3f of the tiles are under the sea", share)
	}
	if top < 0.99 || bottom > -0.99 {
		t.Errorf("land tops out at %v and the sea floor at %v", top, bottom)
	}
}

// Plates make continents, not speckle: most of a land tile's neighbours
// are land, and most of a sea tile's are sea. And it has several biomes.
func TestLandClustersAndBiomesVary(t *testing.T) {
	w := New(terran)
	same, all := 0, 0
	seen := map[Biome]bool{}
	for _, tile := range w.Tiles() {
		seen[w.Biome(tile)] = true
		land := w.Height(tile) >= 0
		for _, nb := range w.Tiling.Neighbours(tile) {
			all++
			if (w.Height(nb) >= 0) == land {
				same++
			}
		}
	}
	if share := float64(same) / float64(all); share < 0.8 {
		t.Errorf("only %.2f of neighbours are alike", share)
	}
	if len(seen) < 5 {
		t.Errorf("only %d biomes: %v", len(seen), seen)
	}
}

// The same terrain is the same world, every time, and another seed is
// another world.
func TestAWorldIsRepeatable(t *testing.T) {
	a, b := New(terran), New(terran)
	other := terran
	other.Seed = 6
	c := New(other)
	differ := 0
	for _, tile := range a.Tiles() {
		if a.Biome(tile) != b.Biome(tile) {
			t.Fatalf("tile %v changed between two makings", tile)
		}
		if a.Biome(tile) != c.Biome(tile) {
			differ++
		}
	}
	if differ == 0 {
		t.Error("another seed made the same world")
	}
}

// Painted is each tile's biome's paint.
func TestPaintedIsTheBiomesPaint(t *testing.T) {
	w := New(terran)
	var look Look
	for b := range Biomes {
		look.Paints[b] = maths.Grey(float64(b) / float64(Biomes))
	}
	tx := w.Painted(look)
	for _, tile := range w.Tiles() {
		c := w.Tiling.Centre(tile)
		if got, want := tx.At(c), look.Paints[w.Biome(tile)]; got != want {
			t.Fatalf("tile %v: %v, want %v for %d", tile, got, want,
				w.Biome(tile))
		}
	}
}

// A frozen world is mostly ice and snow; a world with no sea is all land
// and does not hang.
func TestColdAndDry(t *testing.T) {
	cold := terran
	cold.Cold = 0.55
	w := New(cold)
	icy := 0
	for _, tile := range w.Tiles() {
		if b := w.Biome(tile); b == Ice || b == Snow || b == Tundra {
			icy++
		}
	}
	if share := float64(icy) / float64(len(w.Tiles())); share < 0.5 {
		t.Errorf("a frozen world is only %.2f ice, snow and tundra", share)
	}
	dry := terran
	dry.Ocean = 0
	d := New(dry)
	for _, tile := range d.Tiles() {
		if d.Height(tile) < 0 {
			t.Fatal("a world with no sea has sea")
		}
	}
}

// A world on a band or a plane is a world: its plates are many and not a
// few wedges, and its latitude runs from the middle to the rims, or along
// the plane's north, so the cold biomes gather there.
func TestAWorldOnABandOrAPlaneHasPlates(t *testing.T) {
	for _, c := range []struct {
		name string
		t    Terrain
	}{
		{"band", Terrain{Tiling: hexes.Band{Radius: 30, Width: 20, Around: 90},
			Seed: 3, Ocean: 0.5, Plates: 8}},
		{"plane", Terrain{Tiling: hexes.Flat{Step: 1, Across: 40, Down: 40},
			Seed: 3, Ocean: 0.5, Plates: 8, Axis: maths.V(0, 0, 1)}},
	} {
		w := &World{Terrain: c.t}
		w.lay()
		plate, _, _ := w.plates()
		have := map[int]bool{}
		for _, p := range plate {
			have[p] = true
		}
		if len(have) < 6 {
			t.Errorf("the %s has %d plates of %d", c.name, len(have), c.t.Plates)
		}
		w = New(c.t)
		var mid, rim float64
		for _, tile := range w.tiles {
			l := w.Tiling.Latitude(w.Tiling.Centre(tile), w.north())
			if w.Biome(tile) == Snow || w.Biome(tile) == Tundra || w.Biome(tile) == Ice {
				if l > 0.5 {
					rim++
				} else {
					mid++
				}
			}
		}
		if rim <= mid {
			t.Errorf("the %s is cold in the middle (%v) more than at its poles (%v)",
				c.name, mid, rim)
		}
	}
}
