package world

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// probe is a feature that does what a test says: its share is a function
// of where the point is, it adds a hundred to a height so a tile it
// reached can be seen, and it fixes what it is told for a share.
type probe struct {
	reach float64
	share func(p maths.Vec) float64
	class func(share float64) Biome
}

func (f probe) Footprint() (maths.Vec, float64) { return maths.Vec{}, f.reach }
func (f probe) Share(p, _ maths.Vec, _ float64) float64 {
	return f.share(p)
}
func (f probe) Shape(height, _ float64) float64 { return height + 100 }
func (f probe) Class(share float64) Biome       { return f.class(share) }

// a row of five tiles along x, at 0 to 4, each next to the next, a world
// of them with heights 0 to 4 kept.
func row() *World {
	at := make([]maths.Vec, 5)
	for i := range at {
		at[i] = maths.V(float64(i), 0, 0)
	}
	w := on(placed(at, along(5)...), 1)
	w.height = []float64{0, 1, 2, 3, 4}
	return w
}

// the tiles a feature reaches are the ones whose middles are within its
// reach of its heart, the edge in: from (0,0,0) a reach of 3 is the
// tiles at 0, 1, 2 and exactly 3, and not 4.
func TestAFeatureReachesTheTilesWithinItsReach(t *testing.T) {
	w := row()
	var got []int
	ends(t, "within", func() { got = w.within(maths.Vec{}, 3) })
	if len(got) != 4 || got[0] != 0 || got[1] != 1 || got[2] != 2 || got[3] != 3 {
		t.Errorf("tiles %v are within 3, want 0, 1, 2, 3", got)
	}
}

// carve shapes the tiles whose share is one or under, and passes over
// the rest, and a share that is no number. the tile at x has the share
// x / 2: 0, a half, one, a half past one, but tile 1 has none.
//
// the class is lava below a half and none at it. a reach of 3 takes
// tiles 0 to 3, and tile 4 is not reached at all.
func TestCarveShapesWhatIsWithinTheEdge(t *testing.T) {
	w := row()
	w.Features = []Feature{probe{reach: 3,
		share: func(p maths.Vec) float64 {
			if p.X == 1 {
				return math.NaN()
			}
			return p.X / 2
		},
		class: func(share float64) Biome {
			if share < 0.5 {
				return Lava
			}
			return Unfixed
		}}}
	ends(t, "carve", w.carve)
	want := []float64{100, 1, 102, 3, 4}
	fixed := []Biome{Lava, Unfixed, Unfixed, Unfixed, Unfixed}
	for i := range want {
		if w.height[i] != want[i] || w.fixed[i] != fixed[i] {
			t.Errorf("tile %d is %v and %d, want %v and %d", i, w.height[i],
				w.fixed[i], want[i], fixed[i])
		}
	}
}

// a feature that reaches minus infinity, or any size that is not one, is
// refused by name, as one that reaches plus infinity or no number is.
func TestAFeatureOfNoSizeIsRefusedBothWays(t *testing.T) {
	for _, reach := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
		w := row()
		w.Features = []Feature{probe{reach: reach}}
		if got := refused(w.carve); !strings.Contains(got, "not a size") {
			t.Errorf("a reach of %v: %q", reach, got)
		}
	}
}

// a feature whose heart is outside the world does nothing, however far
// it reaches, and a search from outside the world finds nothing. a plane
// six across has no tile at (500,0,500), and a plateau there that reaches
// a thousand would otherwise lift the whole plane.
func TestAHeartOutsideTheWorldChangesNothing(t *testing.T) {
	tr := Terrain{Tiling: hexes.Flat{Step: 1, Across: 6, Down: 6}, Seed: 3,
		Ocean: 0.3, Plates: 3}
	plain := New(tr)
	tr.Features = []Feature{Plateau{At: maths.V(500, 0, 500), Reach: 1000, Top: 5}}
	w := New(tr)
	for _, tile := range plain.Tiles() {
		if w.Height(tile) != plain.Height(tile) || w.Biome(tile) != plain.Biome(tile) {
			t.Fatalf("tile %v changed from %v to %v", tile, plain.Height(tile),
				w.Height(tile))
		}
	}
	if _, ok := plain.Find(maths.V(500, 0, 500), func(Biome) bool { return true }); ok {
		t.Error("a search from outside the world found a tile")
	}
}

// a search goes outward from the tile nearest the heart and takes the
// first tile the test holds for. on a row whose biomes are sea, sea,
// grass, forest, rock, dry ground from the sea end is the grass at 2.
//
// sea from the far end is the sea at 1. a test that never holds comes
// back with the empty tile, and false.
func TestFindGoesOutwardFromTheHeart(t *testing.T) {
	w := row()
	w.biome = []Biome{Sea, Sea, Grass, Forest, Rock}
	for _, c := range []struct {
		name   string
		toward maths.Vec
		want   func(Biome) bool
		tile   hexes.Tile
		ok     bool
	}{
		{"dry from the sea end", maths.V(0, 0, 0), Land, hexes.Tile{I: 2}, true},
		{"sea from the far end", maths.V(4, 0, 0), Water, hexes.Tile{I: 1}, true},
		{"rock from the sea end", maths.V(0, 0, 0),
			func(b Biome) bool { return b == Rock }, hexes.Tile{I: 4}, true},
		{"nothing", maths.V(2, 0, 0), func(Biome) bool { return false },
			hexes.Tile{}, false},
	} {
		var tile hexes.Tile
		var ok bool
		ends(t, "Find", func() { tile, ok = w.Find(c.toward, c.want) })
		if tile != c.tile || ok != c.ok {
			t.Errorf("%s: %v, %v; want %v, %v", c.name, tile, ok, c.tile, c.ok)
		}
	}
}
