package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// citySeed is the seed of the cities worked here. it was picked so that
// the hashes below fall where the tests need them, and each test checks
// its own before it leans on it.
const citySeed = 63

// pointOfThem is the place asked about: it looks toward (3, 4, 12)/13,
// which is 6/13 from the heart of a metro.
var pointOfThem = maths.V(0.3, 0.4, 1.2)

// coarseSpecs is a coarse rung of the given spacing with a metro's
// heart as tile 0 and open sea as tile 1, naming the tile asked.
func coarseSpecs(spacing float64, nearest int, around map[int][]int) rungSpec {
	return rungSpec{spacing: spacing, nearest: nearest, around: around,
		centres: map[int]maths.Vec{0: heart, 1: deep, 2: deep}}
}

// A metro's heart is one and its edge is zero, a coarse tile's width out. the
// point is 6/13 from the heart, so with a spacing of 1.2 the share is 1 -
// (6/13)/1.2 = 8/13; with 2.4 it is 1 - (6/13)/2.4 = 10.5/13.
//
// the coarse rung is two below the world's, and none for a world with fewer
// than two rungs.
func TestCitiesMetroCoreIsOneLessTheDistanceOverTheSpacing(t *testing.T) {
	specs := []rungSpec{coarseSpecs(1.2, 0, nil), coarseSpecs(2.4, 0, nil), junk, junk}
	for _, c := range []struct {
		rungs int
		core  float64
	}{{-1, 8.0 / 13}, {0, 8.0 / 13}, {1, 8.0 / 13}, {2, 8.0 / 13}, {3, 10.5 / 13}} {
		m := Cities{Tiling: rungs{specs: specs}, Land: inland, Rungs: c.rungs,
			Metro: 10, Seed: citySeed}
		core, near := m.metro(pointOfThem)
		if !within(core, c.core, 1e-12) || !near {
			t.Errorf("rungs %d: the core is %v and near %v, want %v and true", c.rungs, core, near, c.core)
		}
	}
}

// Past a coarse tile's width from the heart the core is zero, not below it, and
// the point is still in the metro's tile.
func TestCitiesMetroCoreIsNoneAWidthOut(t *testing.T) {
	m := Cities{Tiling: rungs{specs: []rungSpec{coarseSpecs(0.4, 0, nil)}}, Land: inland,
		Rungs: 2, Metro: 10, Seed: citySeed}
	if core, near := m.metro(pointOfThem); core != 0 || !near {
		t.Errorf("the core is %v and near %v, want 0 and true", core, near)
	}
}

// A point in a coarse tile that is not a metro is near one when a
// neighbour is, with no core, and not near when none is.
func TestCitiesMetroIsNearWhenANeighbourIsOne(t *testing.T) {
	for _, c := range []struct {
		why    string
		around map[int][]int
		near   bool
	}{
		{"a metro next door", map[int][]int{1: {0}}, true},
		{"a metro among the neighbours", map[int][]int{1: {2, 0}}, true},
		{"only sea next door", map[int][]int{1: {2}}, false},
		{"no neighbours", nil, false},
	} {
		m := Cities{Tiling: rungs{specs: []rungSpec{coarseSpecs(1.2, 1, c.around)}}, Land: inland,
			Rungs: 2, Metro: 10, Seed: citySeed}
		if core, near := m.metro(pointOfThem); core != 0 || near != c.near {
			t.Errorf("%s: the core is %v and near %v, want 0 and %v", c.why, core, near, c.near)
		}
	}
}

// A world with no metros, or fewer than none, is not asked about its
// land at all.
func TestCitiesWithNoMetrosNeverAskTheLand(t *testing.T) {
	for _, metro := range []float64{0, -1} {
		m := Cities{Tiling: rungs{specs: []rungSpec{coarseSpecs(1.2, 0, nil)}},
			Land: func(maths.Vec) bool {
				t.Error("a world with no metros asked about its land")
				return true
			}, Rungs: 2, Metro: metro, Seed: citySeed}
		if core, near := m.metro(pointOfThem); core != 0 || near {
			t.Errorf("metro %v: the core is %v and near %v, want 0 and false", metro, core, near)
		}
	}
}

// A coarse tile is a metro when its hash, seeded two past the world's,
// is below the metro share times its weight: strictly below.
//
// the hash of tile 0 at seed 65 is 0.0877; at the other seeds a mutant
// might take, 61 and 66, it is 0.955 and 0.989. a bound half way
// between is passed by the first and by neither of the others.
//
// a bound of the hash itself is not passed. a metro share of one gives
// the heart, whose weight is 0.55, over the hash.
func TestCitiesMetroIsAMetroByItsOwnHashBelowTheBound(t *testing.T) {
	h := noise.Hash(citySeed+2, 0)
	other := math.Min(noise.Hash(citySeed-2, 0), noise.Hash(citySeed+3, 0))
	if !(h < 0.2 && other > h+0.25) {
		t.Fatalf("the hashes were meant to be small and far from the others: %v, %v", h, other)
	}
	w := (Cities{}).weight(heart)
	specs := []rungSpec{coarseSpecs(1.2, 0, nil)}
	ask := func(metro float64) (float64, bool) {
		m := Cities{Tiling: rungs{specs: specs}, Land: inland, Rungs: 2, Metro: metro, Seed: citySeed}
		return m.metro(pointOfThem)
	}
	if core, near := ask((h + other) / 2 / w); !within(core, 8.0/13, 1e-12) || !near {
		t.Errorf("a bound between the hashes: %v, %v", core, near)
	}
	if core, near := ask(1); !within(core, 8.0/13, 1e-12) || !near {
		t.Errorf("a metro share of one: %v, %v (weight %v, hash %v)", core, near, w, h)
	}
	metro := h / w
	for i := 0; metro*w != h && i < 8; i++ {
		metro = math.Nextafter(metro, math.Inf(1))
	}
	for i := 0; metro*w != h && i < 16; i++ {
		metro = math.Nextafter(metro, math.Inf(-1))
	}
	if metro*w != h {
		t.Fatalf("no metro share times %v gives the hash %v", w, h)
	}
	if core, near := ask(metro); core != 0 || near {
		t.Errorf("a bound of the hash itself: %v, %v, not nought and false", core, near)
	}
}
