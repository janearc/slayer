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

// At is the light at p: the line's where p is on an edge, fading to
// nothing a line's width in.
func (w Wire) At(p maths.Vec) maths.Light {
	if w.Tiling == nil {
		return maths.Light{}
	}
	width := w.Width
	if width <= 0 {
		width = w.Tiling.Spacing() / 10
	}
	tile := w.Tiling.Nearest(p)
	line := 1 - maths.Smooth(0, width, w.Tiling.Edge(p, tile))
	return w.Light.Scale(line)
}
