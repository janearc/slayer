package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// within is whether two numbers are as near as tol. a number that is no
// number is near nothing, so a test that asks it cannot pass on one.
func within(got, want, tol float64) bool { return math.Abs(got-want) <= tol }

// near is whether two lights are the same to a part in a billion.
func near(x, y maths.Light) bool {
	return math.Abs(x.R-y.R) < 1e-9 && math.Abs(x.G-y.G) < 1e-9 &&
		math.Abs(x.B-y.B) < 1e-9
}

// A line's width of zero is six hundredths of the spacing. With a spacing of
// one that is 0.06, and a point 0.03 from the edge is half way: Smooth(0, 0.06,
// 0.03) is 0.5, so the line is half there.
//
// with Lines of one the paint is scaled by 1 - 0.5, with Lines of 0.5 by 1 -
// 0.25.
func TestATiledLineOfNoWidthIsSixHundredthsOfTheSpacing(t *testing.T) {
	paint := maths.RGB(0.8, 0.4, 0.2)
	tl := oneTile{spacing: 1, edge: 0.03}
	if got, want := (Tiled{Tiling: tl, Paints: flatPaint{paint}, Lines: 1}).At(maths.Up),
		paint.Scale(0.5); !near(got, want) {
		t.Errorf("lines of one: %v, want %v", got, want)
	}
	if got, want := (Tiled{Tiling: tl, Paints: flatPaint{paint}, Lines: 0.5}).At(maths.Up),
		paint.Scale(0.75); !near(got, want) {
		t.Errorf("lines of a half: %v, want %v", got, want)
	}
}

// A width of its own is used as it is, whatever the spacing: 0.1 with a
// spacing of 7, and a point 0.05 from the edge is half way again.
func TestATiledLineOfItsOwnWidthIsThatWidth(t *testing.T) {
	paint := maths.RGB(0.8, 0.4, 0.2)
	tl := oneTile{spacing: 7, edge: 0.05}
	got := (Tiled{Tiling: tl, Paints: flatPaint{paint}, Lines: 1, Width: 0.1}).At(maths.Up)
	if want := paint.Scale(0.5); !near(got, want) {
		t.Errorf("a width of a tenth: %v, want %v", got, want)
	}
}

// With no lines the edge is not even asked, and a number less than zero is no
// lines either, not light added at the edge.
func TestATiledSurfaceWithNoLinesNeverAsksTheEdge(t *testing.T) {
	paint := maths.RGB(0.8, 0.4, 0.2)
	for _, lines := range []float64{0, -0.5} {
		asked := 0
		tl := oneTile{spacing: 1, edge: 0, asked: &asked}
		got := (Tiled{Tiling: tl, Paints: flatPaint{paint}, Lines: lines}).At(maths.Up)
		if got != paint || asked != 0 {
			t.Errorf("lines of %v: %v after %d questions of the edge, want %v after none",
				lines, got, asked, paint)
		}
	}
	asked := 0
	tl := oneTile{spacing: 1, edge: 0, asked: &asked}
	(Tiled{Tiling: tl, Paints: flatPaint{paint}, Lines: 1}).At(maths.Up)
	if asked != 1 {
		t.Errorf("a surface with lines asked the edge %d times, not once", asked)
	}
}

// A wire with no tiling is black.
func TestAWireOfNoTilingIsBlack(t *testing.T) {
	if got := (Wire{Light: maths.Grey(1), Width: 1}).At(maths.Up); got != (maths.Light{}) {
		t.Errorf("a wire with no tiling gave %v", got)
	}
}

// A wire's width of zero, or below it, is a tenth of the spacing: 0.1 with a
// spacing of one, so a point 0.05 from the edge is half way and the line is
// half there.
//
// a width of its own, even as little as one with a spacing of one, is used: a
// point 0.5 from the edge is half way in it.
func TestAWireOfNoWidthIsATenthOfTheSpacing(t *testing.T) {
	lamp := maths.RGB(1, 0.5, 0.25)
	tl := oneTile{spacing: 1, edge: 0.05}
	for _, width := range []float64{0, -1} {
		got := (Wire{Tiling: tl, Light: lamp, Width: width}).At(maths.Up)
		if want := lamp.Scale(0.5); !near(got, want) {
			t.Errorf("a width of %v: %v, want %v", width, got, want)
		}
	}
	wide := oneTile{spacing: 1, edge: 0.5}
	got := (Wire{Tiling: wide, Light: lamp, Width: 1}).At(maths.Up)
	if want := lamp.Scale(0.5); !near(got, want) {
		t.Errorf("a width of one: %v, want %v", got, want)
	}
}

// Past a line's width in from the edge a wire is black, and on the edge
// it is the whole light.
func TestAWireFadesToNothingAtItsWidth(t *testing.T) {
	lamp := maths.RGB(1, 0.5, 0.25)
	for _, c := range []struct {
		edge float64
		want maths.Light
	}{{0, lamp}, {0.1, maths.Light{}}, {0.3, maths.Light{}}} {
		got := (Wire{Tiling: oneTile{spacing: 1, edge: c.edge}, Light: lamp}).At(maths.Up)
		if !near(got, c.want) {
			t.Errorf("%v in from the edge: %v, want %v", c.edge, got, c.want)
		}
	}
}

// A hexed surface with a palette but no tiling paints a tile black, and
// asks for the tile of nothing, which is none; one with a tiling but no
// palette is black too.
func TestHexedWithHalfAWorldIsBlack(t *testing.T) {
	if got := (Hexed{Palette: []maths.Light{sea}}).Paint(hexes.Tile{I: 1}); got != (maths.Light{}) {
		t.Errorf("no tiling painted %v", got)
	}
	tile, got := (Hexed{Palette: []maths.Light{sea}}).Tile(maths.Up)
	if tile != (hexes.Tile{}) || got != (maths.Light{}) {
		t.Errorf("no tiling gave the tile %v painted %v", tile, got)
	}
	if got := (Hexed{Tiling: oneTile{spacing: 1}}).Paint(hexes.Tile{I: 1}); got != (maths.Light{}) {
		t.Errorf("no palette painted %v", got)
	}
}

// A palette of one paints every tile that one, whichever the hash.
func TestHexedPaletteOfOnePaintsEveryTileThatOne(t *testing.T) {
	h := Hexed{Tiling: oneTile{spacing: 1}, Palette: []maths.Light{moss}}
	for i := 0; i < 20; i++ {
		h.Seed = uint64(i)
		if got := h.Paint(hexes.Tile{I: i}); got != moss {
			t.Fatalf("seed %d tile %d is %v, not the one paint", i, i, got)
		}
	}
}
