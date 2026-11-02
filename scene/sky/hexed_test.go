package sky

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

var hexed = Hexed{Of: g, Tiling: hexes.Sphere{N: 6}}

// near is whether two lights are within e in every part.
func near(a, b maths.Light, e float64) bool {
	return math.Abs(a.R-b.R) <= e && math.Abs(a.G-b.G) <= e &&
		math.Abs(a.B-b.B) <= e
}

// Inside a tile, away from its seams, the sky is one flat colour: the
// sky's own colour at the tile's middle.
func TestHexedIsFlatInsideATile(t *testing.T) {
	tl := hexed.Tiling
	for _, tile := range tl.Tiles() {
		c := tl.Centre(tile)
		want := g.Light(c)
		if got := hexed.Light(c); got != want {
			t.Fatalf("tile %v: middle is %v, sky there is %v", tile,
				got, want)
		}
		// a little off the middle, still well inside the melt
		off := c.Add(maths.V(0.001, 0.0007, -0.0004)).Unit()
		if tl.Nearest(off) != tile {
			continue
		}
		if got := hexed.Light(off); got != want {
			t.Fatalf("tile %v: off the middle is %v, want %v", tile,
				got, want)
		}
	}
}

// At a seam the hexed sky is the true sky, so two tiles meet at the same
// colour and a step across the seam is a small step in light.
func TestHexedIsTheSkyAtTheSeam(t *testing.T) {
	tl := hexed.Tiling
	tile := tl.Nearest(maths.V(0.3, 0.8, 0.2))
	nb := tl.Neighbours(tile)[0]
	mid := tl.Centre(tile).Add(tl.Centre(nb)).Unit()
	if got, want := hexed.Light(mid), g.Light(mid); !near(got, want, 1e-9) {
		t.Fatalf("at the seam %v, the sky is %v", got, want)
	}
	step := tl.Centre(tile).Sub(tl.Centre(nb)).Scale(1e-4)
	a, b := hexed.Light(mid.Add(step)), hexed.Light(mid.Sub(step))
	if !near(a, b, 1e-3) {
		t.Fatalf("across the seam, %v then %v", a, b)
	}
}

// Without a sky to tile there is no light.
func TestHexedNothingIsNothing(t *testing.T) {
	if got := (Hexed{}).Light(maths.Up); got != (maths.Light{}) {
		t.Fatalf("no sky gave %v", got)
	}
}
