// Package biome paints tiles from seeded height and wetness fields.
// Neighbouring values form coherent seas, coasts, forests, and deserts without
// storing a separate colour choice for every tile.
package biome

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/swatches"
	"math"
)

// the paints, in swatches.Terran's biome order
const (
	deepSea = iota
	shallowSea
	ice
	beach
	desert
	grass
	forest
	jungle
	tundra
	rock
	snow
)

// Fields is the two slow fields over a tiling. Seed makes every world its own;
// Scale is about how wide a continent is, in the tiling's units; Sea is the
// height below which a tile is under water, zero for about half.
type Fields struct {
	Tiling hexes.Tiling
	Seed   uint64
	Scale  float64
	Sea    float64
	// Sprawl is how far a city's lights spread, in tiles: a tight bright
	// heart and arms reaching out from it. zero is each city a single lit
	// tile.
	Sprawl float64
}

// Height is how high a point stands, about minus one to one: the land's
// shape, five octaves so coasts are ragged at every size.
func (f Fields) Height(p maths.Vec) float64 {
	s := f.scale()
	return noise.Fractal(f.Seed, p.X/s, p.Y/s, p.Z/s, 5)
}

// Wet is how much rain a point gets, about minus one to one, on a field of
// its own and a little smaller, so the dry and the wet do not follow the
// coasts.
func (f Fields) Wet(p maths.Vec) float64 {
	s := f.scale() * 0.7
	return noise.Fractal(f.Seed+1, p.X/s, p.Y/s, p.Z/s, 4)
}

// Above is how far a point stands above the sea, zero for water.
func (f Fields) Above(p maths.Vec) float64 {
	return max(0, f.Height(p)-f.Sea)
}

// scale is the field scale, one when none is set.
func (f Fields) scale() float64 {
	if f.Scale <= 0 {
		return 1
	}
	return f.Scale
}

// Paint is a tile's biome, by the height and wet at its centre.
func (f Fields) Paint(tile hexes.Tile) maths.Light {
	c := f.Tiling.Centre(tile)
	kind := classify(f.Height(c)-f.Sea, f.Wet(c))
	return swatches.Light(swatches.Terran[kind])
}

// World is a tiling's biomes worked out once, with the features that need the
// whole map and not one tile: rivers that run downhill to the sea and pool
// where they cannot, volcanoes on the peaks, and cities on the coasts and the
// rivers.
//
// It paints as Fields does, and Shine is what glows: lava, and the cities'
// lights.
type World struct {
	Fields
	biome map[hexes.Tile]int
	river map[hexes.Tile]bool
	lava  map[hexes.Tile]bool
	city  map[hexes.Tile]bool
	// lit is how bright each tile's city light is, when cities sprawl
	lit map[hexes.Tile]float64
	// the counts, for saying what was made
	Rivers, Lakes, Volcanoes, Cities int
	// Lights are the colours a city may glow, one picked a city by hash;
	// empty is a warm sodium light for them all
	Lights []maths.Light
}

// Make works the world out over every tile.
func Make(f Fields) *World {
	w := &World{Fields: f, biome: map[hexes.Tile]int{},
		river: map[hexes.Tile]bool{}, lava: map[hexes.Tile]bool{},
		city: map[hexes.Tile]bool{}}
	tiles := f.Tiling.Tiles()
	height := make(map[hexes.Tile]float64, len(tiles))
	for _, t := range tiles {
		c := f.Tiling.Centre(t)
		height[t] = f.Height(c) - f.Sea
		w.biome[t] = classify(height[t], f.Wet(c))
	}
	wet := func(t hexes.Tile) bool { return w.biome[t] <= shallowSea }
	chance := func(t hexes.Tile, salt int64) float64 {
		return noise.Hash(f.Seed+uint64(salt), f.Tiling.ID(t))
	}
	// rivers: from high, wet ground, the steepest way down until the sea;
	// a tile with no neighbour lower is a basin, and becomes a lake
	for _, t := range tiles {
		if wet(t) || height[t] < 0.12 ||
			f.Wet(f.Tiling.Centre(t)) < 0.1 || chance(t, 1) > 0.03 {
			continue
		}
		w.Rivers++
		at := t
		for range 400 {
			if wet(at) {
				break
			}
			w.river[at] = true
			next, low := at, height[at]
			for _, n := range f.Tiling.Neighbours(at) {
				if height[n] < low {
					next, low = n, height[n]
				}
			}
			if next == at {
				w.biome[at] = shallowSea
				w.Lakes++
				break
			}
			at = next
		}
	}
	// volcanoes: a peak higher than everything round it, now and then,
	// lava at the top and ash on the slopes
	for _, t := range tiles {
		if height[t] < 0.26 || chance(t, 2) > 0.4 {
			continue
		}
		peak := true
		for _, n := range f.Tiling.Neighbours(t) {
			if height[n] >= height[t] {
				peak = false
				break
			}
		}
		if !peak {
			continue
		}
		w.Volcanoes++
		w.lava[t] = true
		for _, n := range f.Tiling.Neighbours(t) {
			w.biome[n] = ash
		}
	}
	// cities: dry land beside the sea or a river, one in a few
	for _, t := range tiles {
		b := w.biome[t]
		if wet(t) || w.river[t] || b == snow || b == rock || b == ash ||
			chance(t, 3) > 0.18 {
			continue
		}
		for _, n := range f.Tiling.Neighbours(t) {
			if wet(n) || w.river[n] {
				w.city[t] = true
				w.Cities++
				break
			}
		}
	}
	if f.Sprawl > 0 {
		w.lit = map[hexes.Tile]float64{}
		for t := range w.city {
			// one city in thirty is an urban heart; the rest are
			// dark
			if chance(t, 8) < 1.0/30 {
				w.sprawl(t, wet)
			}
		}
	}
	return w
}

// the two last paints, after snow
const (
	lava = snow + 1 + iota
	ash
)

// classify is a tile's biome by its height above the sea and its wet.
func classify(h, m float64) int {
	switch {
	case h < -0.12:
		return deepSea
	case h < 0:
		return shallowSea
	case h < 0.025:
		return beach
	case h > 0.34:
		return snow
	case h > 0.24:
		return rock
	case m < -0.18:
		return desert
	case m < 0.02:
		return grass
	case m < 0.22:
		return forest
	}
	return jungle
}

// Paint is a tile's colour: lava, a river's water, a city's stone, or its
// biome.
func (w *World) Paint(tile hexes.Tile) maths.Light {
	switch {
	case w.lava[tile]:
		return swatches.Light(swatches.Terran[lava])
	case w.river[tile]:
		return swatches.Light(swatches.Terran[shallowSea])
	case w.city[tile]:
		return maths.RGB(0.55, 0.52, 0.48)
	}
	return swatches.Light(swatches.Terran[w.biome[tile]])
}

// Shine is the world's own light, as a painter for a material's Shine: lava
// glowing and the cities lit, nothing elsewhere.
func (w *World) Shine() Glow { return Glow{w} }

// Glow paints what a world gives off.
type Glow struct{ w *World }

// Paint is a tile's glow.
func (g Glow) Paint(tile hexes.Tile) maths.Light {
	switch {
	case g.w.lava[tile]:
		return maths.RGB(1.6, 0.45, 0.06)
	case g.w.lit != nil:
		return g.sprawled(tile)
	case g.w.city[tile] && len(g.w.Lights) > 0:
		pick := noise.Hash(g.w.Seed+4, g.w.Tiling.ID(tile))
		lights := len(g.w.Lights)
		return g.w.Lights[int(pick*float64(lights))%lights]
	case g.w.city[tile]:
		return maths.RGB(0.55, 0.42, 0.22)
	}
	return maths.Light{}
}

// sprawl spreads a city's light from its heart: a tight bright middle,
// and arms reaching out along a few ways from it, fading, then nothing,
// as night lights run out along roads. never onto the sea.
func (w *World) sprawl(heart hexes.Tile, wet func(hexes.Tile) bool) {
	reach := int(math.Ceil(3 * w.Sprawl))
	shape := w.armsOf(heart)
	seen := map[hexes.Tile]bool{heart: true}
	ring := []hexes.Tile{heart}
	for step := 0; step <= reach && len(ring) > 0; step++ {
		var next []hexes.Tile
		for _, t := range ring {
			if !wet(t) {
				centre := w.Tiling.Centre(t)
				w.lit[t] += shape.weight(centre, step)
			}
			for _, n := range w.Tiling.Neighbours(t) {
				if !seen[n] {
					seen[n] = true
					next = append(next, n)
				}
			}
		}
		ring = next
	}
}

// arms is one city's shape: where its heart is, two ways across the
// ground there to measure angles by, how many arms and how they turn,
// and how far it spreads
type arms struct {
	heart, across, along maths.Vec
	count, turn, spread  float64
}

// armsOf is a heart's shape, its arms counted and turned by a hash
func (w *World) armsOf(heart hexes.Tile) arms {
	at := w.Tiling.Centre(heart)
	out := at.Unit()
	reference := maths.V(0, 1, 0)
	if math.Abs(out.Dot(reference)) > 0.9 {
		reference = maths.V(1, 0, 0)
	}
	across := reference.Sub(out.Scale(out.Dot(reference))).Unit()
	id := w.Tiling.ID(heart)
	return arms{heart: at, across: across, along: out.Cross(across),
		count: 4 + math.Floor(4*noise.Hash(w.Seed+6, id)),
		turn:  2 * math.Pi * noise.Hash(w.Seed+7, id), spread: w.Sprawl}
}

// weight is how much light a tile gets from this city: the middle, a
// third of the spread wide, at full; along an arm, out to twice the
// spread; between the arms, nothing past the middle
func (shape arms) weight(at maths.Vec, step int) float64 {
	far := float64(step)
	middle := math.Exp(-math.Pow(far/(0.35*shape.spread), 2))
	away := at.Sub(shape.heart)
	angle := math.Atan2(away.Dot(shape.along), away.Dot(shape.across))
	along := math.Cos(shape.count*angle + shape.turn)
	onArm := math.Pow(math.Max(0, along), 8)
	arm := onArm * math.Exp(-math.Pow(far/(1.4*shape.spread), 2))
	return math.Max(middle, 0.6*arm)
}

// sprawled is a tile's share of the city light round it: dim and amber
// at the ends of the arms, dense and blue at the hearts, and broken up
// tile by tile so it reads as streets and not as a smooth glow. the
// colours run from the first of Lights to the last; with none, sodium.
func (g Glow) sprawled(tile hexes.Tile) maths.Light {
	level := math.Min(g.w.lit[tile], 1.6)
	if level < 0.04 {
		return maths.Light{}
	}
	grain := 0.35 + 0.65*noise.Hash(g.w.Seed+5, g.w.Tiling.ID(tile))
	if len(g.w.Lights) == 0 {
		return maths.RGB(0.55, 0.42, 0.22).Scale(level * grain)
	}
	colour := along(g.w.Lights, math.Min(level, 1))
	return colour.Scale(level * grain)
}

// along is a colour partway through a list, zero the first and one the last,
// mixed between the two it falls between
func along(colours []maths.Light, share float64) maths.Light {
	at := share * float64(len(colours)-1)
	low := int(math.Floor(at))
	if low >= len(colours)-1 {
		return colours[len(colours)-1]
	}
	mix := at - float64(low)
	return colours[low].Scale(1 - mix).Add(colours[low+1].Scale(mix))
}
