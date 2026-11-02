// Package world generates planetary terrain and biomes on a hexagonal grid. The
// model was carried over from Miami and retains its plate-based approach.
//
// Continental plates sit higher than oceanic plates. Converging plates form
// mountains or coastal trenches; diverging plates form rifts or ridges.
// Boundary effects fade with distance, noise adds roughness, and sea level is
// chosen to submerge the requested fraction of tiles.
//
// Latitude and elevation determine cold; distance to sea influences rainfall.
// The resulting surface is computed once and queried by tile.
package world

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
)

// Biome is what a tile is.
type Biome uint8

// The biomes, from the sea floor to the snow line.
const (
	Deep Biome = iota
	Sea
	Ice
	Beach
	Desert
	Grass
	Forest
	Jungle
	Tundra
	Rock
	Snow
	Lava   // molten ground, a volcano's crater
	Ash    // a volcano's slopes
	Biomes // how many there are
)

// Terrain is how a world is made.
type Terrain struct {
	// Tiling is the surface, and the rung the plates and biomes are
	// decided on. It is held whole, every tile, so it is coarse: a sphere
	// split forty times is a world from orbit.
	Tiling hexes.Tiling
	Seed   uint64
	// Ocean is the share of tiles under the sea, and Plates how many
	// plates the crust is cut into; fewer than two is two.
	Ocean  float64
	Plates int
	// Axis is the world's north, for latitude; zero is the world's up.
	Axis maths.Vec
	// Cold is added to every tile's cold: greater than zero is an ice age,
	// and well above it a frozen world.
	Cold float64
	// Features are what happened to the ground, applied in order after
	// the sea is poured and before the biomes are picked.
	Features []Feature
}

// World is one terrain, worked out.
type World struct {
	Terrain
	tiles  []hexes.Tile  // by index
	at     map[int64]int // by tile id, the index
	near   [][]int       // by index, the neighbours' indexes
	height []float64     // above the sea is positive; by index
	biome  []Biome       // by index
	fixed  []Biome       // by index, a class a feature fixed, or Unfixed
}

// New works out the terrain. an Ocean, a Cold or an Axis that is no number
// or endless is refused by name, not made a plausible world; an Axis too
// long or too short for Unit is measured at its own size.
func New(t Terrain) *World {
	t = t.checked()
	w := &World{Terrain: t}
	w.lay()
	plate, land, drift := w.plates()
	stress := w.boundaries(plate, land, drift)
	w.height = make([]float64, len(w.tiles))
	for i, tile := range w.tiles {
		c := w.Tiling.Centre(tile)
		base := -0.45
		if land[plate[i]] {
			base = 0.22
		}
		rough := noise.Fractal(t.Seed+40, c.X*2.2, c.Y*2.2, c.Z*2.2, 4)
		w.height[i] = base + 0.5*stress[i] + 0.26*rough
	}
	w.pourSea()
	w.carve()
	w.classify()
	return w
}

// lay lists the tiles, and each one's neighbours, by index.
func (w *World) lay() {
	if w.Tiling == nil {
		w.Tiling = hexes.Sphere{}
	}
	w.tiles = w.Tiling.Tiles()
	w.at = make(map[int64]int, len(w.tiles))
	for i, tile := range w.tiles {
		w.at[w.Tiling.ID(tile)] = i
	}
	w.near = make([][]int, len(w.tiles))
	for i, tile := range w.tiles {
		for _, nb := range w.Tiling.Neighbours(tile) {
			if j, ok := w.at[w.Tiling.ID(nb)]; ok {
				w.near[i] = append(w.near[i], j)
			}
		}
	}
}

// index is a tile's index, or minus one for a tile the world does not
// hold: a plane's tiles beyond its extent.
func (w *World) index(tile hexes.Tile) int {
	if i, ok := w.at[w.Tiling.ID(tile)]; ok {
		return i
	}
	return -1
}

// Biome is what a tile is. Beyond the edge of the world it is the deep.
func (w *World) Biome(tile hexes.Tile) Biome {
	if i := w.index(tile); i >= 0 {
		return w.biome[i]
	}
	return Deep
}

// Height is how high a tile is: the sea level is zero, the highest few percent
// of land at one and the deepest few of sea at minus one, the very top past
// them, to 1.5 on a sphere and further on a plane. Beyond the edge of the world
// it is minus one.
func (w *World) Height(tile hexes.Tile) float64 {
	if i := w.index(tile); i >= 0 {
		return w.height[i]
	}
	return -1
}

// Tiles is every tile of the world, in the order the world holds them.
func (w *World) Tiles() []hexes.Tile { return w.tiles }

// across is about how many tiles the world is across: for a sphere the
// split, which is what the fall-off of a boundary and the reach of rain
// were tuned in, and the same number for any other surface.
func (w *World) across() float64 {
	return math.Sqrt(float64(len(w.tiles)) / 10)
}

// north is the world's north, with zero read as up.
func (w *World) north() maths.Vec {
	if w.Axis == (maths.Vec{}) {
		return maths.Up
	}
	return w.Axis.Unit()
}

// plates cuts the tiles into plates: each tile goes to the nearest of a
// few middles scattered over the surface, measured along it through
// noise so the plates are not polygons. the noise is in the unit
// sphere's units, so a plane or a band is barely warped.
//
// It returns each tile's plate, which plates are continent, and how each
// plate moves: a turning about an axis of its own, through the world's
// middle, at random.
func (w *World) plates() (plate []int, land []bool, drift []maths.Vec) {
	seed := w.Seed
	count := max(2, w.Plates)
	mids := make([]maths.Vec, count)
	land = make([]bool, count)
	drift = make([]maths.Vec, count)
	for p := range count {
		k := int64(p)
		mids[p] = w.Tiling.Somewhere(noise.Hash(seed+1, k),
			noise.Hash(seed+2, k), noise.Hash(seed+3, k))
		land[p] = noise.Hash(seed+4, k) < 0.42
		drift[p] = maths.V(noise.Hash(seed+5, k)-.5,
			noise.Hash(seed+6, k)-.5, noise.Hash(seed+7, k)-.5).Unit()
	}
	plate = make([]int, len(w.tiles))
	for i, tile := range w.tiles {
		c := w.Tiling.Centre(tile)
		warp := noise.Fractal(seed+8, c.X*1.8, c.Y*1.8, c.Z*1.8, 3)
		best := math.Inf(1)
		for p, m := range mids {
			d := w.Tiling.Distance(c, m)
			d += 0.55 * warp * noise.Hash(seed+9, int64(p))
			if d < best {
				best, plate[i] = d, p
			}
		}
	}
	return plate, land, drift
}

// boundaries is how much each tile is raised or lowered by the plates
// meeting near it.
func (w *World) boundaries(plate []int, land []bool,
	drift []maths.Vec) []float64 {
	n := len(w.tiles)
	push := make([]float64, n)
	var front []int
	for i, tile := range w.tiles {
		c := w.Tiling.Centre(tile)
		for _, j := range w.near[i] {
			pi, pj := plate[i], plate[j]
			if pi == pj {
				continue
			}
			// how fast the two plates close on each other here:
			// each plate's ground moves as its turning axis
			// crossed with where it is
			away := w.Tiling.Centre(w.tiles[j]).Sub(c).Unit()
			rel := drift[pi].Cross(c).Sub(drift[pj].Cross(c))
			closing := rel.Dot(away)
			effect := meeting(closing, land[pi], land[pj])
			if math.Abs(effect) > math.Abs(push[i]) {
				push[i] = effect
			}
		}
		if push[i] != 0 {
			front = append(front, i)
		}
	}
	return w.spread(push, front)
}

// meeting is what two plates closing at a rate do to the ground between
// them, by what each is made of.
func meeting(closing float64, a, b bool) float64 {
	switch {
	case closing > 0 && a && b:
		return 1.1 * closing // ranges
	case closing > 0 && a:
		return 0.8 * closing // a coast pushed up
	case closing > 0 && b:
		return -0.6 * closing // the trench beside it
	case closing > 0:
		return 0.5 * closing // an island arc
	case a && b:
		return 0.35 * closing // a rift
	}
	return -0.12 * closing // a ridge, small
}

// spread carries a boundary's effect inland: each tile takes the effect
// of the nearest boundary, fading with how many tiles away it is.
func (w *World) spread(push []float64, front []int) []float64 {
	n := len(w.tiles)
	reach := make([]int, n)
	for i := range reach {
		reach[i] = -1
	}
	for _, i := range front {
		reach[i] = 0
	}
	fall := math.Max(1.5, w.across()/6)
	out := make([]float64, n)
	for len(front) > 0 {
		var next []int
		for _, i := range front {
			out[i] = push[i] * math.Exp(-float64(reach[i])/fall)
			for _, j := range w.near[i] {
				if reach[j] >= 0 {
					continue
				}
				reach[j] = reach[i] + 1
				push[j] = push[i]
				next = append(next, j)
			}
		}
		front = next
	}
	return out
}

// pourSea lowers every tile by the height that leaves the given share
// under water, so a world asked for sixty percent sea gets sixty percent.
//
// Then land is scaled so its highest few percent reach one, and the sea
// floor so its deepest few reach minus one. How tall the plates threw the
// mountains depends on how fast they happened to move, and a biome rule
// written as "above 0.8 is snow" should mean the same on every world.
//
// at a sea of nearly all or nearly none, the few percent on one side are
// the other side's, and that side is not scaled. a world of no tiles has
// no sea to pour.
func (w *World) pourSea() {
	if len(w.height) == 0 {
		return
	}
	sorted := append([]float64(nil), w.height...)
	sort.Float64s(sorted)
	last := float64(len(sorted) - 1)
	share := math.Max(0, math.Min(1, w.Ocean))
	level := sorted[int(math.Round(share*last))]
	top := sorted[int(0.985*last)] - level
	bottom := level - sorted[int(0.015*last)]
	for i := range w.height {
		e := w.height[i] - level
		if e > 0 && top > 0 {
			e /= top
		}
		if e < 0 && bottom > 0 {
			e /= bottom
		}
		w.height[i] = e
	}
}

// classify picks each tile's biome from its height, its cold and its rain.
func (w *World) classify() {
	wet := w.toSea()
	north := w.north()
	w.biome = make([]Biome, len(w.tiles))
	for i, tile := range w.tiles {
		c := w.Tiling.Centre(tile)
		e := w.height[i]
		lat := w.Tiling.Latitude(c, north)
		nz := noise.Fractal(w.Seed+60, c.X*4, c.Y*4, c.Z*4, 3)
		cold := lat*1.05 + math.Max(0, e)*0.45 + 0.12*nz + w.Cold
		reach := math.Max(2, w.across()/5)
		rain := math.Exp(-float64(wet[i])/reach) +
			0.3*noise.Fractal(w.Seed+61, c.X*3, c.Y*3, c.Z*3, 3)
		w.biome[i] = pick(e, cold, rain, wet[i])
		if w.fixed[i] != Unfixed {
			w.biome[i] = w.fixed[i]
		}
	}
}

// pick is a biome from height, cold and rain.
func pick(e, cold, rain float64, fromSea int) Biome {
	if e < 0 {
		switch {
		case cold > 1.0:
			return Ice
		case e < -0.35:
			return Deep
		}
		return Sea
	}
	switch {
	case cold > 1.08 || e > 0.82:
		return Snow
	case e > 0.62:
		return Rock
	case cold > 0.88:
		return Tundra
	case fromSea == 1 && e < 0.03:
		return Beach
	case rain < 0.28:
		return Desert
	case rain < 0.5:
		return Grass
	case cold < 0.35 && rain > 0.75:
		return Jungle
	}
	return Forest
}

// toSea is, for every tile, how many tiles away the nearest sea is: zero for
// the sea itself, and every tile for a world with no sea.
func (w *World) toSea() []int {
	n := len(w.tiles)
	d := make([]int, n)
	var front []int
	for i := range d {
		d[i] = -1
		if w.height[i] < 0 {
			d[i] = 0
			front = append(front, i)
		}
	}
	for len(front) > 0 {
		var next []int
		for _, i := range front {
			for _, j := range w.near[i] {
				if d[j] < 0 {
					d[j] = d[i] + 1
					next = append(next, j)
				}
			}
		}
		front = next
	}
	for i := range d {
		if d[i] < 0 {
			d[i] = n
		}
	}
	return d
}

// Look is how a world is painted: a paint for each biome, and how dark
// the lines between tiles are.
type Look struct {
	Paints [Biomes]maths.Light
	Lines  float64
}

// Painted is the world's surface as a texture, in a look: each tile its
// biome's paint, with lines between.
func (w *World) Painted(look Look) texture.Texture {
	return texture.Tiled{Tiling: w.Tiling, Paints: painted{w, look},
		Lines: look.Lines}
}

// painted paints a tile by its biome.
type painted struct {
	w    *World
	look Look
}

// Paint is the paint of the tile's biome.
func (p painted) Paint(tile hexes.Tile) maths.Light {
	return p.look.Paints[p.w.Biome(tile)]
}

// checked is the terrain, refused by name where a number of it is none
// or endless, with an Axis too long or too short for Unit measured at its
// own size. an ordinary terrain is given back as it was.
func (t Terrain) checked() Terrain {
	for _, n := range []struct {
		name string
		v    float64
	}{{"Ocean", t.Ocean}, {"Cold", t.Cold}, {"Axis's x", t.Axis.X},
		{"Axis's y", t.Axis.Y}, {"Axis's z", t.Axis.Z}} {
		if math.IsNaN(n.v) || math.IsInf(n.v, 0) {
			panic(fmt.Sprintf("world: %s is %v, which makes no world",
				n.name, n.v))
		}
	}
	a := t.Axis
	if s := math.Max(math.Abs(a.X), math.Max(math.Abs(a.Y), math.Abs(a.Z))); s != 0 &&
		(s < 1e-150 || s > 5e149) {
		t.Axis = maths.V(a.X/s, a.Y/s, a.Z/s)
	}
	return t
}
