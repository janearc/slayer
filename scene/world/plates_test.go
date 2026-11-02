package world

import (
	"math"
	"sort"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// scattered is a chart of n tiles at points in the box from (lo,lo,lo)
// to (hi,hi,hi), laid out by a fixed sequence, none next to another.
func scattered(n int, lo, hi float64) *chart {
	c := chained(n)
	state := uint64(5)
	for i := range c.at {
		var v [3]float64
		for k := range v {
			state = state*6364136223846793005 + 1442695040888963407
			v[k] = lo + float64(state>>11)/(1<<53)*(hi-lo)
		}
		c.at[i] = maths.V(v[0], v[1], v[2])
	}
	return c
}

// the plates a world is cut into, worked from the noise package with the
// streams the model draws from: plate k has its middle at (h1, h2, h3),
// is land if h4 is under 0.42, and turns about (h5 - 0.5, h6 - 0.5,
// h7 - 0.5), h the hash of k under the seed plus that number.
//
// the chart hands the three numbers back as the middle.
//
// a tile goes to the plate with the least distance to its middle plus
// 0.55 times the noise of the tile (seed plus 8, at 1.8 times the place,
// three octaves) times the hash of the plate under the seed plus 9. 300
// tiles in the unit box, ten plates.
func TestPlatesAreDrawnFromTheSeed(t *testing.T) {
	const seed, count = 9, 10
	c := scattered(300, 0, 1)
	w := on(c, seed)
	w.Plates = count
	plate, land, drift := w.plates()
	if len(land) != count || len(drift) != count {
		t.Fatalf("%d plates, %d drifts, want %d", len(land), len(drift), count)
	}
	h := func(stream uint64, k int) float64 { return noise.Hash(seed+stream, int64(k)) }
	lands, seas := 0, 0
	for k := range count {
		if want := h(4, k) < 0.42; land[k] != want {
			t.Errorf("plate %d is land: %v, want %v", k, land[k], want)
		}
		if land[k] {
			lands++
		} else {
			seas++
		}
		want := maths.V(h(5, k)-0.5, h(6, k)-0.5, h(7, k)-0.5).Unit()
		if !drift[k].Near(want, 1e-12) {
			t.Errorf("plate %d turns about %v, want %v", k, drift[k], want)
		}
		if k > 0 && drift[k] == drift[k-1] {
			t.Errorf("plates %d and %d turn alike", k-1, k)
		}
	}
	if lands == 0 || seas == 0 {
		t.Errorf("%d plates of land and %d of sea", lands, seas)
	}
	for i, p := range c.at {
		warp := noise.Fractal(seed+8, p.X*1.8, p.Y*1.8, p.Z*1.8, 3)
		best, want := math.Inf(1), -1
		for k := range count {
			mid := maths.V(h(1, k), h(2, k), h(3, k))
			if d := p.Sub(mid).Len() + 0.55*warp*h(9, k); d < best {
				best, want = d, k
			}
		}
		if plate[i] != want {
			t.Errorf("tile %d at %v is plate %d, want %d", i, p, plate[i], want)
		}
	}
}

// fewer than two plates is two, whatever was asked for.
func TestACrustHasTwoPlatesAtLeast(t *testing.T) {
	for _, asked := range []int{-3, 0, 1} {
		w := on(scattered(40, 0, 1), 4)
		w.Plates = asked
		plate, land, drift := w.plates()
		if len(land) != 2 || len(drift) != 2 {
			t.Errorf("%d plates asked, %d and %d made", asked, len(land), len(drift))
		}
		for i, p := range plate {
			if p < 0 || p > 1 {
				t.Fatalf("%d plates asked, tile %d is plate %d", asked, i, p)
			}
		}
	}
}

// two plates whose middles are at (-1,0,0) and (1,0,0), and tiles on the plane
// between them, the same distance from both. the plate a tile goes to has the
// lesser bias, the noise there times the hash of the plate: it is plate 0 if
// (h0 - h1) times the noise is under zero, else plate 1.
//
// at the origin the noise is zero, the two are as near as each other, and the
// first has it.
func TestWarpDecidesBetweenPlatesAsNearAsEachOther(t *testing.T) {
	const seed = 6
	c := scattered(300, -1, 1)
	for i := range c.at {
		c.at[i].X = 0
	}
	c.at[0] = maths.Vec{}
	w := on(c, seed)
	w.Plates = 2
	c.mids = []maths.Vec{maths.V(-1, 0, 0), maths.V(1, 0, 0)}
	plate, _, _ := w.plates()
	if plate[0] != 0 {
		t.Errorf("the origin, as near both, is plate %d, want 0", plate[0])
	}
	h0, h1 := noise.Hash(seed+9, 0), noise.Hash(seed+9, 1)
	for i, p := range c.at {
		warp := noise.Fractal(seed+8, p.X*1.8, p.Y*1.8, p.Z*1.8, 3)
		if i == 0 || math.Abs(warp*(h0-h1)) < 1e-9 {
			continue
		}
		want := 1
		if warp*(h0-h1) < 0 {
			want = 0
		}
		if plate[i] != want {
			t.Errorf("tile %d at %v, warp %v: plate %d, want %d", i, p, warp,
				plate[i], want)
		}
	}
}

// the height of a tile is the base of its plate, 0.22 for continent and
// -0.45 for ocean, plus half the stress of the boundaries and 0.26 of the
// noise (seed plus 40, at 2.2 times the place, four octaves), and then
// the sea is poured.
//
// checked on a world of 162 tiles against the parts, each worked out on
// its own: the plates and the boundaries by the tests above, the noise
// from its package, and the pouring by rank.
func TestAHeightIsBasePlusStressPlusNoise(t *testing.T) {
	tr := Terrain{Tiling: hexes.Sphere{N: 4}, Seed: 5, Ocean: 0.55, Plates: 7}
	w := New(tr)
	ref := &World{Terrain: tr}
	ref.lay()
	plate, land, drift := ref.plates()
	stress := ref.boundaries(plate, land, drift)
	height := make([]float64, len(ref.tiles))
	moved := 0
	for i, tile := range ref.tiles {
		c := ref.Tiling.Centre(tile)
		base := -0.45
		if land[plate[i]] {
			base = 0.22
		}
		height[i] = base + 0.5*stress[i] + 0.26*noise.Fractal(tr.Seed+40,
			c.X*2.2, c.Y*2.2, c.Z*2.2, 4)
		if stress[i] != 0 {
			moved++
		}
	}
	if moved == 0 {
		t.Fatal("no tile is moved by a boundary, which this world has")
	}
	want := byRank(height, tr.Ocean)
	for i, tile := range w.Tiles() {
		if math.Abs(w.Height(tile)-want[i]) > 1e-9 {
			t.Fatalf("tile %d is %v, want %v", i, w.Height(tile), want[i])
		}
	}
}

// byRank is the sea poured by ranking the tiles: the one at rank
// round(ocean * last) is the water line, 0.985 of the way up is the top
// mark and 0.015 the bottom. heights are taken off the line and divided
// by the distance to the mark on their side, if there is one.
func byRank(height []float64, ocean float64) []float64 {
	order := make([]int, len(height))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return height[order[a]] < height[order[b]] })
	last := float64(len(height) - 1)
	at := func(share float64) float64 { return height[order[int(share*last)]] }
	level := height[order[int(math.Round(math.Min(1, math.Max(0, ocean))*last))]]
	top, bottom := at(0.985)-level, level-at(0.015)
	out := make([]float64, len(height))
	for i, h := range height {
		switch e := h - level; {
		case e > 0 && top > 0:
			out[i] = e / top
		case e < 0 && bottom > 0:
			out[i] = e / bottom
		default:
			out[i] = e
		}
	}
	return out
}
