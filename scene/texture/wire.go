package texture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// Wire is the edges of a tiling drawn as lines of light on black: a wireframe,
// a dressing for a surface that has nothing to show, as a horizon has. Light is
// the lines' colour and brightness, Width how wide a line is as a chord, zero
// for a tenth of the tiling's spacing.
type Wire struct {
	Tiling hexes.Tiling
	Light  maths.Light
	Width  float64
}

// lineWidth is the line's width, with no width read as a share of the
// tiling's spacing.
func (wire Wire) lineWidth() float64 {
	if wire.Width <= nought {
		return wire.Tiling.Spacing() / wireShare
	}
	return wire.Width
}

// At is the light at the point: the line's where the point is on an edge,
// fading to nothing a line's width in.
func (wire Wire) At(point maths.Vec) maths.Light {
	if wire.Tiling == nil {
		return noPaint
	}
	width := wire.lineWidth()
	tile := wire.Tiling.Nearest(point)
	line := unity - maths.Smooth(nought, width, wire.Tiling.Edge(point, tile))
	return wire.Light.Scale(line)
}
