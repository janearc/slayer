package world

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/texture"
)

// a point where a fine texture painted at each of these rungs is a
// different paint, so a test that sees one paint can say which rung it
// was painted at. tiles vary by a hash of their own, so a different
// tile is a different paint.
func apart(t *testing.T, w *World, look Look, rungs []int) maths.Vec {
	t.Helper()
	state := uint64(11)
	for try := 0; try < 400; try++ {
		var v [3]float64
		for k := range v {
			state = state*6364136223846793005 + 1442695040888963407
			v[k] = float64(state>>11)/(1<<53)*2 - 1
		}
		p := maths.V(v[0], v[1], v[2]).Unit()
		seen := map[maths.Light]bool{}
		for _, r := range rungs {
			seen[w.Fine(look, r, 0.4, 0).At(p)] = true
		}
		if len(seen) == len(rungs) {
			return p
		}
	}
	t.Fatalf("no point is painted differently at each of rungs %v", rungs)
	return maths.Vec{}
}

func paints() Look {
	var look Look
	for b := range Biomes {
		look.Paints[b] = maths.Grey(0.1 + 0.06*float64(b))
	}
	return look
}

// the rung a fine texture picks for a footprint: tiles are halved from the
// world's own, spacing S, until one is no wider than 6 footprints, to 40 at
// most. a world split 3 has S = atan(2) / 3 = 0.36905, and 6 * (S / 6) is S
// again, bit for bit.
//
// the footprint, and the rung worked by hand:
//
// S / 6, a tile exactly 6 wide: none. S / 13, 6 wide is 0.4615 S: S and S / 2
// are over it, S / 4 is under, two. S / 20, 0.3 S: the same, two. S / 100, 0.06
// S: S / 16 is over it, S / 32 under, five, where thirds would say three.
func TestAFineTextureChoosesItsRungByTheFootprint(t *testing.T) {
	w := New(Terrain{Tiling: hexes.Sphere{N: 3}, Seed: 5, Ocean: 0.5, Plates: 5})
	look := paints()
	s := w.Tiling.Spacing()
	if 6*(s/6) != s {
		t.Fatal("6 * (S / 6) is not S on this machine")
	}
	cases := []struct {
		name      string
		footprint float64
		rungs     int
	}{
		{"six wide", s / 6, 0},
		{"just under six wide", s / 13, 2},
		{"a fifth", s / 20, 2},
		{"a hundredth", s / 100, 5},
	}
	p := apart(t, w, look, []int{0, 1, 2, 3, 5})
	auto := w.Fine(look, 0, 0.4, 0).(texture.Footprinted)
	for _, c := range cases {
		want := w.Fine(look, c.rungs, 0.4, 0).At(p)
		if got := auto.Within(p, c.footprint); got != want {
			t.Errorf("%s: painted %v, want the rung %d's %v", c.name, got, c.rungs, want)
		}
	}
}

// a fine texture with no rung and no footprint, or a footprint of zero or
// under, is painted on the world's own rung. and one with a rung is painted on
// it for any footprint at all: with a rung of 1 or 2, a footprint small enough
// to want rung 5 or none at all.
func TestAFineTextureWithARungKeepsItAndAFootprintOfNoneIsCoarse(t *testing.T) {
	w := New(Terrain{Tiling: hexes.Sphere{N: 3}, Seed: 5, Ocean: 0.5, Plates: 5})
	look := paints()
	s := w.Tiling.Spacing()
	p := apart(t, w, look, []int{0, 1, 2, 3, 5})
	coarse := w.Fine(look, 0, 0.4, 0).At(p)
	auto := w.Fine(look, 0, 0.4, 0).(texture.Footprinted)
	for _, footprint := range []float64{0, -1} {
		if got := auto.Within(p, footprint); got != coarse {
			t.Errorf("a footprint of %v is %v, want the world's own %v", footprint,
				got, coarse)
		}
	}
	for _, rungs := range []int{1, 2} {
		fixed := w.Fine(look, rungs, 0.4, 0)
		want := fixed.At(p)
		for _, footprint := range []float64{s / 100, 0, -1, s * 2} {
			got := fixed.(texture.Footprinted).Within(p, footprint)
			if got != want {
				t.Errorf("rung %d with a footprint of %v is %v, want %v", rungs,
					footprint, got, want)
			}
		}
	}
}

// a footprint no tile is small enough for stops at forty rungs down, not
// thirty-nine or forty-one: on a chart, whose spacing is one, a footprint
// of 1e-30 would take ninety-some halvings, and is asked of the chart's
// Finer forty times, counted.
func TestTheRungsStopAtForty(t *testing.T) {
	c := chained(1)
	w := on(c, 1)
	w.biome = []Biome{Grass}
	auto := w.Fine(paints(), 0, 0, 0).(texture.Footprinted)
	ends(t, "Within", func() { auto.Within(maths.Vec{}, 1e-30) })
	if c.fined != 40 {
		t.Errorf("a finer rung was asked for %d times, want 40", c.fined)
	}
}
