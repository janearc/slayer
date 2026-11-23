package texture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// noTile is the tile of a surface that has no tiles.
var noTile hexes.Tile

// Hexed is a surface in tiles painted from a palette by a hash of the
// tile, with a dark line where a point is near the tile's edge: a Tiled
// whose painter is the hash. It is the simplest hex world, one colour a
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
func (hexed Hexed) Paint(tile hexes.Tile) maths.Light {
	if len(hexed.Palette) == nought || hexed.Tiling == nil {
		return noPaint
	}
	hash := noise.Hash(hexed.Seed, hexed.Tiling.ID(tile))
	place := hash * float64(len(hexed.Palette))
	return hexed.Palette[min(int(place), len(hexed.Palette)-unity)]
}

// Tile is the tile a point is on, and the paint it has, before any line.
func (hexed Hexed) Tile(point maths.Vec) (hexes.Tile, maths.Light) {
	if hexed.Tiling == nil {
		return noTile, noPaint
	}
	tile := hexed.Tiling.Nearest(point)
	return tile, hexed.Paint(tile)
}

// At is the paint at the point, darkened by the line if the point is
// near an edge.
func (hexed Hexed) At(point maths.Vec) maths.Light {
	return Tiled{Tiling: hexed.Tiling, Paints: hexed, Lines: hexed.Lines,
		Width: hexed.Width}.At(point)
}
