package texture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// Painter says what paint a tile has. A world answers by biome; Hexed
// answers by hash.
type Painter interface {
	Paint(tile hexes.Tile) maths.Light
}

// Tiled is a surface in tiles: for a point on the surface, the tile it is
// in, painted by Paints, with a dark line where the point is near the
// tile's edge.
type Tiled struct {
	// Tiling is the tiles: a sphere, a band or a plane. Nil is no tiles,
	// and the surface is black.
	Tiling hexes.Tiling
	// Paints is what each tile is painted. Nil is black everywhere.
	Paints Painter
	// Lines is how dark the edges between tiles are, zero for none and one
	// for black.
	Lines float64
	// Width is how wide a line is, as a chord of the unit sphere. Zero
	// means six hundredths of the tiling's spacing.
	Width float64
}

// width is the line's width, with zero read as a share of the spacing.
func (t Tiled) width() float64 {
	if t.Width == 0 {
		return t.Tiling.Spacing() * 0.06
	}
	return t.Width
}

// At is the tile's paint at p, darkened by the line if p is near an edge.
// The point is handed to the tiling as it is: a sphere reads its
// direction, a band its angle and height, a plane its place.
func (t Tiled) At(p maths.Vec) maths.Light {
	if t.Paints == nil || t.Tiling == nil {
		return maths.Light{}
	}
	tile := t.Tiling.Nearest(p)
	paint := t.Paints.Paint(tile)
	if t.Lines <= 0 {
		return paint
	}
	line := 1 - maths.Smooth(0, t.width(), t.Tiling.Edge(p, tile))
	return paint.Scale(1 - t.Lines*line)
}
