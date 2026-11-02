package texture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// oneTile is a tiling of a single tile whose spacing and edge distance
// are numbers the test sets, so a line's width and weight can be worked
// by hand. what it does not define panics, because the embedded tiling
// is nil, so a test learns at once when the texture asks for more.
type oneTile struct {
	hexes.Tiling
	spacing, edge float64
	asked         *int
}

func (o oneTile) Nearest(maths.Vec) hexes.Tile { return hexes.Tile{I: 7} }

func (o oneTile) Spacing() float64 { return o.spacing }

func (o oneTile) ID(tile hexes.Tile) int64 { return int64(tile.I) }

func (o oneTile) Edge(maths.Vec, hexes.Tile) float64 {
	if o.asked != nil {
		*o.asked++
	}
	return o.edge
}

// flatPaint paints every tile one light.
type flatPaint struct{ l maths.Light }

func (f flatPaint) Paint(hexes.Tile) maths.Light { return f.l }
