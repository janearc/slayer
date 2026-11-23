package texture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// painter says what paint a tile has. A world answers by biome; Hexed
// answers by hash.
type painter interface {
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
	Paints painter
	// Lines is how dark the edges between tiles are, zero for none and one
	// for black.
	Lines float64
	// Width is how wide a line is, as a chord of the unit sphere. Zero
	// means six hundredths of the tiling's spacing.
	Width float64
}

// tileLineShare is the share of a tiling's spacing a line is wide when no
// width is given.
var tileLineShare = pickedNumber("tile-line-share")

// width is the line's width, with zero read as a share of the spacing.
func (tiled Tiled) width() float64 {
	if tiled.Width == nought {
		return tiled.Tiling.Spacing() * tileLineShare
	}
	return tiled.Width
}

// At is the tile's paint at the point, darkened by the line if the point
// is near an edge. The point is handed to the tiling as it is: a sphere
// reads its direction, a band its angle and height, a plane its place.
func (tiled Tiled) At(point maths.Vec) maths.Light {
	if tiled.Paints == nil || tiled.Tiling == nil {
		return noPaint
	}
	tile := tiled.Tiling.Nearest(point)
	paint := tiled.Paints.Paint(tile)
	if tiled.Lines <= nought {
		return paint
	}
	edge := tiled.Tiling.Edge(point, tile)
	line := unity - maths.Smooth(nought, tiled.width(), edge)
	return paint.Scale(unity - tiled.Lines*line)
}
