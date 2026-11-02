package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// byID paints a tile by its id, so a check knows what to expect.
type byID struct{ tl hexes.Tiling }

func (b byID) Paint(tile hexes.Tile) maths.Light {
	return maths.Grey(float64(b.tl.ID(tile)%7) / 7)
}

// A tiled surface asks its painter for every tile and darkens the edges.
func TestTiledPaintsByTileAndLinesTheEdges(t *testing.T) {
	tl := hexes.Sphere{N: 3}
	td := Tiled{Tiling: tl, Paints: byID{tl}, Lines: 0.5}
	for _, tile := range tl.Tiles() {
		c := tl.Centre(tile)
		if got, want := td.At(c), (byID{tl}).Paint(tile); got != want {
			t.Fatalf("tile %v: %v, want %v", tile, got, want)
		}
	}
	tile := tl.Nearest(maths.V(0.1, 0.9, 0.2))
	nb := tl.Centre(tl.Neighbours(tile)[0])
	mid := tl.Centre(tile).Add(nb).Unit()
	if td.At(mid) != (byID{tl}).Paint(tl.Nearest(mid)).Scale(0.5) {
		t.Fatalf("on the edge, %v", td.At(mid))
	}
}

// With no painter every tile is black.
func TestTiledNoPainterIsBlack(t *testing.T) {
	if got := (Tiled{}).At(maths.Up); got != (maths.Light{}) {
		t.Fatalf("no painter gave %v", got)
	}
}
