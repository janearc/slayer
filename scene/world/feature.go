package world

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// A feature is what biomes cannot make: a volcano, a canyon, an ice
// shelf, a cliff. Biomes say what the ground is where it is; a feature
// says what is there because something happened.
//
// It has a footprint, the tiles it reaches, and a rule for each: its new
// height, and the class it fixes, if it fixes one. Features are applied
// after the sea is poured and before the biomes are picked, in order.

// Feature is a change to a world over a footprint of tiles.
type Feature interface {
	// Footprint is where the feature is and how far it reaches, in the
	// surface's own units: a chord for a sphere, a distance for a plane.
	Footprint() (at maths.Vec, reach float64)
	// Share is how far into the feature a tile at p is, from zero at the
	// heart to one at the edge; above one it is outside.
	Share(p, at maths.Vec, reach float64) float64
	// Shape is a tile's new height from its old one and its share.
	Shape(height, share float64) float64
	// Class is the biome the feature fixes for a tile at that share, or
	// Unfixed to leave it to the rules.
	Class(share float64) Biome
}

// Unfixed is a class a feature leaves to the rules.
const Unfixed Biome = Biomes

// round is the plain share: how far p is from the heart as a share of
// the reach.
func round(p, at maths.Vec, reach float64) float64 {
	if reach <= 0 {
		return 2
	}
	return p.Sub(at).Len() / reach
}

// Volcano is a cone with a caldera: the ground rises toward the heart,
// and at the heart falls into a crater of lava, with ash on the slopes.
type Volcano struct {
	// At is the heart, and Reach how far the slopes run.
	At    maths.Vec
	Reach float64
	// Rise is how high the rim stands above the ground it sits on.
	Rise float64
	// Dead is a volcano long cold: rock in the crater and on the
	// slopes, not lava and ash.
	Dead bool
}

// Footprint is the heart and the reach.
func (v Volcano) Footprint() (maths.Vec, float64) { return v.At, v.Reach }

// Share is the plain one.
func (v Volcano) Share(p, at maths.Vec, reach float64) float64 {
	return round(p, at, reach)
}

// Shape is the cone: at the crater's wall 0.85 of the rise, stepping to
// the full rise just past it, falling off to the edge, and at the heart
// a crater a third of the way back down. past the edge, a share over one,
// it is the ground as it was.
func (v Volcano) Shape(height, share float64) float64 {
	if share > 1 {
		return height
	}
	if share <= 0.2 {
		return height + v.Rise*(0.65+share)
	}
	return height + v.Rise*math.Pow(1-(share-0.2)/0.8, 1.6)
}

// Class is lava in the crater, ash on the upper slopes, and the rules
// below that.
func (v Volcano) Class(share float64) Biome {
	switch {
	case share > 0.55:
		return Unfixed
	case v.Dead:
		return Rock
	case share <= 0.2:
		return Lava
	}
	return Ash
}

// Canyon is a cut along a path from one place to another: the ground
// falls away toward the line, bare rock down its walls and floor.
type Canyon struct {
	From, To maths.Vec
	// Width is how far from the line the walls reach, and Depth how far
	// below the ground the floor is.
	Width, Depth float64
}

// Footprint is the middle of the path, and half its length plus the
// width.
func (c Canyon) Footprint() (maths.Vec, float64) {
	mid := c.From.Add(c.To).Scale(0.5)
	return mid, c.From.Sub(c.To).Len()/2 + c.Width
}

// Share is how far p is from the line between From and To, as a share
// of the width.
func (c Canyon) Share(p, _ maths.Vec, _ float64) float64 {
	if c.Width <= 0 {
		return 2
	}
	return toSegment(p, c.From, c.To) / c.Width
}

// Shape is the floor at the line, the ground at the walls' edge, and a
// smooth fall between.
func (c Canyon) Shape(height, share float64) float64 {
	return height - c.Depth*(1-maths.Smooth(0, 1, share))
}

// Class is rock down the walls and the floor.
func (c Canyon) Class(share float64) Biome {
	if share < 0.85 {
		return Rock
	}
	return Unfixed
}

// Shelf is an ice shelf: the sea near a coast frozen over, flat ice just
// above the water, from the heart out to the reach.
type Shelf struct {
	At    maths.Vec
	Reach float64
}

// Footprint is the heart and the reach.
func (s Shelf) Footprint() (maths.Vec, float64) { return s.At, s.Reach }

// Share is the plain one.
func (s Shelf) Share(p, at maths.Vec, reach float64) float64 {
	return round(p, at, reach)
}

// Shape leaves land alone and lifts the sea to just above the water.
func (s Shelf) Shape(height, share float64) float64 {
	if height >= 0 {
		return height
	}
	return 0.01
}

// Class is ice, everywhere the shelf reaches.
func (s Shelf) Class(float64) Biome { return Ice }

// Plateau is a flat top with a cliff round it: the ground inside is
// raised to the top where it is lower, and left where it is higher.
type Plateau struct {
	At    maths.Vec
	Reach float64
	// Top is the height of the flat.
	Top float64
}

// Footprint is the heart and the reach.
func (p Plateau) Footprint() (maths.Vec, float64) { return p.At, p.Reach }

// Share is the plain one.
func (p Plateau) Share(q, at maths.Vec, reach float64) float64 {
	return round(q, at, reach)
}

// Shape is the top, where the ground was lower.
func (p Plateau) Shape(height, share float64) float64 {
	if share > 1 {
		return height
	}
	return math.Max(height, p.Top)
}

// Class is rock at the cliff's edge and the rules on the flat.
func (p Plateau) Class(share float64) Biome {
	if share > 0.8 {
		return Rock
	}
	return Unfixed
}

// toSegment is how far p is from the line from a to b, between its ends.
func toSegment(p, a, b maths.Vec) float64 {
	ab := b.Sub(a)
	l := ab.Dot(ab)
	if l == 0 {
		return p.Sub(a).Len()
	}
	t := math.Max(0, math.Min(1, p.Sub(a).Dot(ab)/l))
	return p.Sub(a.Add(ab.Scale(t))).Len()
}

// carve applies the terrain's features to the heights, in order, and
// records the classes they fix.
func (w *World) carve() {
	w.fixed = make([]Biome, len(w.tiles))
	for i := range w.fixed {
		w.fixed[i] = Unfixed
	}
	for n, f := range w.Features {
		at, reach := f.Footprint()
		if math.IsNaN(reach) || math.IsInf(reach, 0) {
			panic(fmt.Sprintf("world: feature %d reaches %v, which is "+
				"not a size", n, reach))
		}
		for _, i := range w.within(at, reach) {
			c := w.Tiling.Centre(w.tiles[i])
			share := f.Share(c, at, reach)
			// written so a share of no number, never within, is
			// passed over as one past the edge is
			if !(share <= 1) {
				continue
			}
			w.height[i] = f.Shape(w.height[i], share)
			if b := f.Class(share); b != Unfixed {
				if b > Unfixed {
					panic(fmt.Sprintf("world: feature %d fixes biome %d, "+
						"and there are %d", n, b, Biomes))
				}
				w.fixed[i] = b
			}
		}
	}
}

// within is the indexes of the tiles whose middles are within reach of
// at, found by walking outward from the nearest.
func (w *World) within(at maths.Vec, reach float64) []int {
	start, ok := w.at[w.Tiling.ID(w.Tiling.Nearest(at))]
	if !ok {
		return nil
	}
	seen := map[int]bool{start: true}
	front, out := []int{start}, []int{}
	for len(front) > 0 {
		var next []int
		for _, i := range front {
			if w.Tiling.Centre(w.tiles[i]).Sub(at).Len() > reach {
				continue
			}
			out = append(out, i)
			for _, j := range w.near[i] {
				if !seen[j] {
					seen[j] = true
					next = append(next, j)
				}
			}
		}
		front = next
	}
	return out
}

// Find is the nearest tile to a direction, or point, for which want
// holds: land near here, sea near here. ok is false when no tile does. a
// want of nil is refused by name.
func (w *World) Find(toward maths.Vec, want func(Biome) bool) (hexes.Tile,
	bool) {
	if want == nil {
		panic("world: Find wants a want, and was given nil")
	}
	start, ok := w.at[w.Tiling.ID(w.Tiling.Nearest(toward))]
	if !ok {
		return hexes.Tile{}, false
	}
	seen := map[int]bool{start: true}
	front := []int{start}
	for len(front) > 0 {
		var next []int
		for _, i := range front {
			if want(w.biome[i]) {
				return w.tiles[i], true
			}
			for _, j := range w.near[i] {
				if !seen[j] {
					seen[j] = true
					next = append(next, j)
				}
			}
		}
		front = next
	}
	return hexes.Tile{}, false
}

// Land is whether a biome is dry ground.
func Land(b Biome) bool { return b >= Beach && b != Ice }

// Water is whether a biome is sea.
func Water(b Biome) bool { return b == Deep || b == Sea }

// Patch is a class laid over the ground as it is: an ice plain, a dark
// mare, a salt flat. It moves nothing.
type Patch struct {
	At    maths.Vec
	Reach float64
	Of    Biome
}

// Footprint is the heart and the reach.
func (p Patch) Footprint() (maths.Vec, float64) { return p.At, p.Reach }

// Share is the plain one.
func (p Patch) Share(q, at maths.Vec, reach float64) float64 {
	return round(q, at, reach)
}

// Shape leaves the height alone.
func (p Patch) Shape(height, _ float64) float64 { return height }

// Class is the patch's class, everywhere it reaches.
func (p Patch) Class(float64) Biome { return p.Of }
