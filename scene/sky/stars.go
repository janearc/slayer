package sky

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// Stars is the night: a sky of points, so many to a tile of a fine
// sphere tiling, each where its tile's hash puts it, as bright as its
// hash says, white with a little warmth or coolness. A direction that
// lands near one brings back its light; the rest is Dark.
type Stars struct {
	// Tiling is the rung the stars are placed on; one star at most a
	// tile. Zero means a hundred and twenty splits, about 144,000 tiles.
	Tiling hexes.Sphere
	// Share is what share of the tiles hold a star, from zero to one. Zero
	// means a fifth.
	Share float64
	// Size is a star's radius as a share of the tiling's spacing. Zero
	// means a tenth.
	Size float64
	// Bright is the light of the brightest star, the rest falling off
	// from it, but for its tint, which takes red or blue to 1.15 of it.
	// Zero means one.
	Bright float64
	// Dark is the sky between the stars.
	Dark maths.Light
	Seed uint64
}

// tiling is the rung, with zero read as a hundred and twenty.
func (s Stars) tiling() hexes.Sphere {
	if s.Tiling.N == 0 {
		return hexes.Sphere{N: 120}
	}
	return s.Tiling
}

// share is the share of tiles with a star, with zero read as a fifth.
func (s Stars) share() float64 {
	if s.Share <= 0 {
		return 0.2
	}
	return s.Share
}

// size is a star's radius as a share of the spacing, zero read as a
// tenth.
func (s Stars) size() float64 {
	if s.Size <= 0 {
		return 0.1
	}
	return s.Size
}

// bright is the brightest star's light, with zero read as one.
func (s Stars) bright() float64 {
	if s.Bright <= 0 {
		return 1
	}
	return s.Bright
}

// Light is the dark plus the stars near the direction, each soft to its
// edge: the one in its tile, and those in the tiles beside it when Size
// is past 0.3. a direction that points nowhere is no number of light.
//
// a star is cut where it reaches past the tiles walked. measured against
// a sum over every tile: at Size 0.1, 2 of 608 lit directions; at 0.3,
// 172 of 5390; at 1.2, about one in eleven; from 0.35 to 0.6 none.
func (s Stars) Light(toward maths.Vec) maths.Light {
	if !measurable(toward) {
		// measured at its own size and asked again, or no number
		d := direction(toward)
		if math.IsNaN(d.X) {
			return maths.Grey(math.NaN())
		}
		return s.Light(d)
	}
	// as it was from here, line for line, so nothing drawn moves
	d := toward.Unit()
	tl := s.tiling()
	tile := tl.Nearest(d)
	light := s.Dark
	light = light.Add(s.star(tl, tile, d))
	if s.size() > 0.3 {
		// a big star reaches into the tiles beside its own
		for _, n := range tl.Neighbours(tile) {
			light = light.Add(s.star(tl, n, d))
		}
	}
	return light
}

// star is the light of the tile's star in a direction, if the tile has
// one and the direction is within its size of it: soft to its edge.
func (s Stars) star(tl hexes.Sphere, tile hexes.Tile, d maths.Vec) maths.Light {
	id := tl.ID(tile)
	if noise.Hash(s.Seed, id) > s.share() {
		return maths.Light{}
	}
	// the star sits a little off the tile's middle, so the sky is not a
	// lattice
	c := tl.Centre(tile)
	off := maths.V(noise.Hash(s.Seed+1, id)-0.5,
		noise.Hash(s.Seed+2, id)-0.5,
		noise.Hash(s.Seed+3, id)-0.5).Scale(tl.Spacing() * 0.6)
	at := c.Add(off).Unit()
	radius := s.size() * tl.Spacing()
	apart := d.Sub(at).Len()
	if apart > radius {
		return maths.Light{}
	}
	// most stars are faint: the brightness falls as the cube of a hash
	b := noise.Hash(s.Seed+4, id)
	light := s.bright() * b * b * b
	// a little colour: the warm ones redder, the cool ones bluer
	tint := noise.Hash(s.Seed+5, id) - 0.5
	col := maths.RGB(1+0.3*tint, 1, 1-0.3*tint).Scale(light)
	edge := 1 - maths.Smooth(0, 1, apart/radius)
	return col.Scale(edge)
}
