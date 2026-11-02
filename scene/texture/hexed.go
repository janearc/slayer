package texture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// Hexed is a surface in tiles painted from a palette by a hash of the
// tile, with a dark line where a point is near the tile's edge: a Tiled
// whose Painter is the hash. It is the simplest hex world, one colour a
// tile and nothing decided about why; a world with reasons is scene/world.
type Hexed struct {
	// Tiling is the tiles: a sphere, a band or a plane. Nil is no tiles,
	// and the surface is black.
	Tiling hexes.Tiling
	// Palette is the paints a tile may have; each tile picks one by hash.
	// Empty is black.
	Palette []maths.Light
	// Seed decides which tile gets which paint.
	Seed uint64
	// Lines is how dark the edges between tiles are, zero for none and one
	// for black.
	Lines float64
	// Width is how wide a line is, as a chord of the unit sphere. Zero
	// means six hundredths of the tiling's spacing.
	Width float64
}

// Paint is the paint a tile has: one of the palette, by hash.
func (h Hexed) Paint(tile hexes.Tile) maths.Light {
	if len(h.Palette) == 0 || h.Tiling == nil {
		return maths.Light{}
	}
	k := noise.Hash(h.Seed, h.Tiling.ID(tile)) * float64(len(h.Palette))
	return h.Palette[min(int(k), len(h.Palette)-1)]
}

// Tile is the tile a point is on, and the paint it has, before any line.
func (h Hexed) Tile(p maths.Vec) (hexes.Tile, maths.Light) {
	if h.Tiling == nil {
		return hexes.Tile{}, maths.Light{}
	}
	tile := h.Tiling.Nearest(p)
	return tile, h.Paint(tile)
}

// At is the paint at p, darkened by the line if p is near an edge.
func (h Hexed) At(p maths.Vec) maths.Light {
	return Tiled{Tiling: h.Tiling, Paints: h, Lines: h.Lines,
		Width: h.Width}.At(p)
}
