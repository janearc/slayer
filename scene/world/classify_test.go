package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// classed is the biomes of a world on a chart whose tile heights are
// given, with nothing fixed by a feature.
func classed(t *testing.T, c *chart, seed uint64, cold float64,
	height []float64) []Biome {
	t.Helper()
	w := on(c, seed)
	w.Cold = cold
	w.height = height
	w.fixed = make([]Biome, len(height))
	for i := range w.fixed {
		w.fixed[i] = Unfixed
	}
	ends(t, "classify", w.classify)
	return w.biome
}

// cold is the latitude times 1.05, plus 0.45 of the height if there is any,
// plus the world's own, and noise that is zero on the chart. a land probe is
// next to the one sea tile, so its rain is e to the minus a half, 0.6065, and
// it is forest unless cold.
//
// a cold of 0.88 is the tundra's edge and 1.0 the ice's.
//
// at a height of 0.5 the height adds 0.225, so a latitude of 0.62 is a cold of
// 0.651 + 0.225 = 0.876, under the edge, and 0.63 is 0.8865, over it. with the
// world 0.2 colder the edge is crossed between 0.4, a cold of 0.42 + 0.225 +
// 0.2 = 0.845, and 0.45, which is 0.8975.
//
// in the sea the height adds nothing, so a latitude of 0.94 is a cold of 0.987,
// sea, and 0.96 is 1.008, ice, at a height of -0.2 and at -0.9. a beach, 0.02
// high and a tile from the sea, at latitude zero has a cold of 0.009.
func TestColdIsLatitudeHeightAndTheWorld(t *testing.T) {
	for _, c := range []struct {
		name  string
		e     float64
		lat   float64
		world float64
		want  Biome
	}{
		{"mild at 0.5", 0.5, 0.62, 0, Forest},
		{"cold at 0.5", 0.5, 0.63, 0, Tundra},
		{"mild, the world colder", 0.5, 0.4, 0.2, Forest},
		{"cold, the world colder", 0.5, 0.45, 0.2, Tundra},
		{"sea under the ice", -0.2, 0.94, 0, Sea},
		{"ice at the pole", -0.2, 0.96, 0, Ice},
		{"deep sea not iced", -0.9, 0.94, 0, Deep},
		{"deep sea iced", -0.9, 0.96, 0, Ice},
		{"a beach", 0.02, 0, 0, Beach},
	} {
		ch := chained(2, [2]int{0, 1})
		if c.e < 0 {
			ch = chained(2)
		}
		ch.lat[ch.at[1]] = c.lat
		got := classed(t, ch, 1, c.world, []float64{-0.5, c.e})
		if got[1] != c.want {
			t.Errorf("%s: %d, want %d", c.name, got[1], c.want)
		}
	}
}

// rain is e to the minus the tiles from the sea over the reach of the
// rain, a fifth of the tiles across and at least 2. in a row of ten,
// the sea and nine of land, the reach is 2, and tiles 1 to 4 from the
// sea have 0.6065, 0.3679, 0.2231 and 0.1353 at a cold of 0.225:
//
// forest, grass, desert, desert.
//
// a row of 10000 is sqrt(1000) = 31.62 across, the reach is 6.325, and
// rain at 1, 4, 5, 8 and 9 tiles is 0.8538, 0.5313, 0.4538, 0.2823 and
// 0.2410: jungle, forest, grass, grass, desert.
func TestRainFallsWithTheDistanceFromTheSea(t *testing.T) {
	for _, c := range []struct {
		tiles int
		want  map[int]Biome
	}{
		{10, map[int]Biome{1: Forest, 2: Grass, 3: Desert, 4: Desert}},
		{10000, map[int]Biome{1: Jungle, 4: Forest, 5: Grass, 8: Grass, 9: Desert}},
	} {
		ch := chained(c.tiles, along(c.tiles)...)
		height := make([]float64, c.tiles)
		for i := range height {
			height[i] = 0.5
		}
		height[0] = -0.5
		got := classed(t, ch, 1, 0, height)
		for k, want := range c.want {
			if got[k] != want {
				t.Errorf("%d tiles, %d from the sea: %d, want %d", c.tiles, k,
					got[k], want)
			}
		}
	}
}

// the noise in cold and in rain, worked from the noise itself. 24 tiles are
// given a latitude of (0.88 +- 0.004 - 0.225 - 0.12 n) / 1.05, n the noise at
// the tile, which is a cold of 0.88 with a margin of 0.004 over or under the
// tundra's edge. they are tundra if over and not if under.
//
// 24 more have a latitude of zero and a rain of 0.6065 + 0.3 m within 0.006 of
// 0.5, m the noise there, picked from places in a row. they are grass if the
// rain is under 0.5 and forest if not, and any other noise moves some across.
func TestTheNoiseOfColdAndRain(t *testing.T) {
	const seed = 17
	state := uint64(7)
	place := func() maths.Vec {
		var v [3]float64
		for k := range v {
			state = state*6364136223846793005 + 1442695040888963407
			v[k] = float64(state>>11)/(1<<53)*4 - 2
		}
		return maths.V(v[0], v[1], v[2])
	}
	type probe struct {
		at   maths.Vec
		lat  float64
		want Biome // Tundra, Grass or Forest; Desert stands for not tundra
	}
	var probes []probe
	for len(probes) < 24 {
		p, margin, want := place(), 0.004, Tundra
		if len(probes)%2 == 1 {
			margin, want = -0.004, Desert
		}
		n := noise.Fractal(seed+60, p.X*4, p.Y*4, p.Z*4, 3)
		probes = append(probes, probe{p, (0.88 + margin - 0.225 - 0.12*n) / 1.05, want})
	}
	for len(probes) < 48 {
		p := place()
		m := noise.Fractal(seed+61, p.X*3, p.Y*3, p.Z*3, 3)
		if r := math.Exp(-0.5) + 0.3*m; math.Abs(r-0.5) < 0.006 {
			want := Forest
			if r < 0.5 {
				want = Grass
			}
			probes = append(probes, probe{p, 0, want})
		}
	}
	ch := chained(1 + len(probes))
	height := []float64{-0.5}
	for i, p := range probes {
		ch.at[i+1] = p.at
		ch.lat[p.at] = p.lat
		ch.near[0] = append(ch.near[0], i+1)
		ch.near[i+1] = []int{0}
		height = append(height, 0.5)
	}
	got := classed(t, ch, seed, 0, height)
	for i, p := range probes {
		if p.want == Desert && got[i+1] == Tundra || p.want != Desert && got[i+1] != p.want {
			t.Errorf("tile %d, wanted %d (a desert is not tundra), is %d", i, p.want, got[i+1])
		}
	}
}
