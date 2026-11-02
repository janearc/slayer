package world

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// refused is the panic f gives, as a string, or "" for none.
func refused(f func()) (got string) {
	defer func() { got, _ = recover().(string) }()
	f()
	return ""
}

// a terrain with a number that is none, or endless, is refused by name,
// not made a world with no sea or no ice in it.
func TestATerrainOfNoNumberIsRefused(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	for name, tr := range map[string]Terrain{
		"Ocean":    {Tiling: hexes.Sphere{N: 3}, Ocean: nan},
		"Cold":     {Tiling: hexes.Sphere{N: 3}, Cold: inf},
		"Axis's y": {Tiling: hexes.Sphere{N: 3}, Axis: maths.V(0, nan, 0)},
	} {
		if got := refused(func() { New(tr) }); !strings.Contains(got, name) {
			t.Errorf("a terrain of no %s: %q", name, got)
		}
	}
}

// an axis of any length is the same north: 1e-200 or 1e200 up makes the
// world that up does, not one with its latitude lost.
func TestAnAxisOfAnyLengthIsTheSameNorth(t *testing.T) {
	want := New(Terrain{Tiling: hexes.Sphere{N: 4}, Seed: 2, Ocean: 0.6})
	for _, up := range []float64{1e-200, 1e200} {
		got := New(Terrain{Tiling: hexes.Sphere{N: 4}, Seed: 2, Ocean: 0.6,
			Axis: maths.V(0, up, 0)})
		for _, tile := range want.Tiles() {
			if got.Biome(tile) != want.Biome(tile) {
				t.Fatalf("an axis %v up: %v is %v, not %v", up, tile,
					got.Biome(tile), want.Biome(tile))
			}
		}
	}
}

// a world of no tiles has none, and no sea to pour into them.
func TestAWorldOfNoTilesIsEmpty(t *testing.T) {
	w := New(Terrain{Tiling: hexes.Flat{}})
	if len(w.Tiles()) != 0 {
		t.Errorf("a flat of no size has %d tiles", len(w.Tiles()))
	}
}

// a feature that reaches no number, or forever, is refused by name; one
// that fixes a biome there is not is refused before it is painted.
func TestAFeatureIsASizeAndABiome(t *testing.T) {
	for _, reach := range []float64{math.NaN(), math.Inf(1)} {
		tr := Terrain{Tiling: hexes.Sphere{N: 3},
			Features: []Feature{Volcano{At: maths.Up, Reach: reach, Rise: 1}}}
		if got := refused(func() { New(tr) }); !strings.Contains(got, "not a size") {
			t.Errorf("a volcano reaching %v: %q", reach, got)
		}
	}
	tr := Terrain{Tiling: hexes.Sphere{N: 3},
		Features: []Feature{Patch{At: maths.Up, Reach: 0.5, Of: Biome(99)}}}
	if got := refused(func() { New(tr) }); !strings.Contains(got, "biome 99") {
		t.Errorf("a patch of biome 99: %q", got)
	}
	w := New(Terrain{Tiling: hexes.Sphere{N: 3}})
	if got := refused(func() { w.Find(maths.Up, nil) }); !strings.Contains(got, "nil") {
		t.Errorf("Find of nil: %q", got)
	}
}

// past a volcano's edge the ground is as it was, and at the edge too: a
// share of one and a half is not a power of a negative.
func TestPastAVolcanoTheGroundIsAsItWas(t *testing.T) {
	v := Volcano{Rise: 1}
	for _, share := range []float64{1, 1.5, 3} {
		if got := v.Shape(0.25, share); got != 0.25 {
			t.Errorf("a volcano at a share of %v is %v", share, got)
		}
	}
}
