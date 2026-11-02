package sky

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// Hexed is any sky gently rendered into hexes: inside a tile it is the
// sky's colour at the tile's middle, one flat colour, and toward the seams
// it melts back into the true sky, so no two tiles meet at a hard line.
// Stood on a body, this is the sky overhead.
type Hexed struct {
	// Of is the sky being tiled.
	Of Sky
	// Tiling is the tiles: a sphere, a band or a plane. Nil is one tile
	// everywhere, which is to say no tiles.
	Tiling hexes.Tiling
	// Soft is how far in from a tile's edge the melt reaches, as a chord
	// of the unit sphere. Zero means a quarter of the tiling's spacing,
	// which is in radians: near enough a chord at these sizes, not one.
	Soft float64
}

// soft is the melt, with zero read as a quarter of the spacing.
func (h Hexed) soft() float64 {
	if h.Soft == 0 {
		return h.Tiling.Spacing() / 4
	}
	return h.Soft
}

// Light is the light from the direction toward: the tile's flat colour
// inside, the true sky at the seam, and a smooth blend between. A sky
// is directions, so its tiling should be a sphere; a band or a plane is
// not refused, and tiles directions as if they were places.
func (h Hexed) Light(toward maths.Vec) maths.Light {
	if h.Of == nil {
		return maths.Light{}
	}
	if h.Tiling == nil {
		return h.Of.Light(toward)
	}
	if !measurable(toward) {
		// measured at its own size and asked again, or no number
		d := direction(toward)
		if math.IsNaN(d.X) {
			return maths.Grey(math.NaN())
		}
		return h.Light(d)
	}
	// as it was from here, line for line, so nothing drawn moves
	d := toward.Unit()
	tile := h.Tiling.Nearest(d)
	flat := h.Of.Light(h.Tiling.Centre(tile))
	in := maths.Smooth(0, h.soft(), h.Tiling.Edge(d, tile))
	if in >= 1 {
		return flat
	}
	true_ := h.Of.Light(d)
	return true_.Scale(1 - in).Add(flat.Scale(in))
}
