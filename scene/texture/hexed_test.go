package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

var sea, moss = maths.RGB(0.1, 0.3, 0.6), maths.RGB(0.3, 0.5, 0.2)

var hexed = Hexed{Tiling: hexes.Sphere{N: 4}, Palette: []maths.Light{sea,
	moss}, Seed: 3, Lines: 0.5}

// Every tile is one of the paints, the same at its middle and anywhere
// well inside it, and both paints are used.
func TestHexedPaintsEveryTileFromThePalette(t *testing.T) {
	tl := hexed.Tiling
	seas, mosses := 0, 0
	for _, tile := range tl.Tiles() {
		c := tl.Centre(tile)
		got := hexed.At(c.Scale(3)) // any radius: it is a direction
		switch got {
		case sea:
			seas++
		case moss:
			mosses++
		default:
			t.Fatalf("tile %v is %v, not a paint", tile, got)
		}
		off := c.Add(maths.V(0.002, -0.001, 0.0015)).Unit()
		if tl.Nearest(off) == tile && hexed.At(off) != got {
			t.Fatalf("tile %v changes paint near its middle", tile)
		}
	}
	if seas == 0 || mosses == 0 {
		t.Fatalf("%d sea and %d moss", seas, mosses)
	}
}

// On the edge between two tiles the paint is darkened by the line, and
// with no line it is not.
func TestHexedDrawsLinesAtEdges(t *testing.T) {
	tl := hexed.Tiling
	tile := tl.Nearest(maths.V(0.2, 0.9, -0.3))
	nb := tl.Neighbours(tile)[0]
	mid := tl.Centre(tile).Add(tl.Centre(nb)).Unit()
	_, paint := hexed.Tile(mid)
	if got := hexed.At(mid); got != paint.Scale(0.5) {
		t.Fatalf("on the edge, %v; the paint is %v", got, paint)
	}
	plain := hexed
	plain.Lines = 0
	if got := plain.At(mid); got != paint {
		t.Fatalf("with no lines, %v; the paint is %v", got, paint)
	}
}

// With nothing in the palette every tile is black.
func TestHexedEmptyPaletteIsBlack(t *testing.T) {
	if got := (Hexed{}).At(maths.Up); got != (maths.Light{}) {
		t.Fatalf("no palette gave %v", got)
	}
}
