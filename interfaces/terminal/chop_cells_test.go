package terminal

import (
	"image"
	"image/color"
	"testing"
)

// a cell takes two pixels across and three down, and a short edge is padded:
// the cells across are the width over two and the cells down the height over
// three, each rounded up. worked by hand for each pair below.
//
// zero wide or zero tall is zero cells, not a cell of padding.
func TestChopMakesWholeCellsAndPadsTheEdge(t *testing.T) {
	for _, w := range []struct{ dx, dy, cols, rows int }{
		{0, 0, 0, 0}, {0, 3, 0, 1}, {2, 0, 1, 0}, {1, 0, 1, 0},
		{1, 1, 1, 1}, {2, 3, 1, 1}, {2, 1, 1, 1}, {1, 3, 1, 1},
		{3, 4, 2, 2}, {4, 6, 2, 2}, {5, 7, 3, 3}, {6, 9, 3, 3},
		{7, 10, 4, 4},
	} {
		img := image.NewRGBA(image.Rect(0, 0, w.dx, w.dy))
		var got *Cells
		ends(t, "chop", func() { got = Chop(img) })
		if got.W != w.cols || got.H != w.rows || len(got.Cells) != w.cols*w.rows {
			t.Errorf("%d by %d pixels: %d across, %d down, %d cells; want %d, %d, %d",
				w.dx, w.dy, got.W, got.H, len(got.Cells), w.cols, w.rows, w.cols*w.rows)
		}
	}
}

// an image of no size has no cells and Chop says so, with nothing
// drawn from it.
func TestChopOfNoPixelsIsNoCells(t *testing.T) {
	got := Chop(image.NewRGBA(image.Rectangle{}))
	if got.W != 0 || got.H != 0 || len(got.Cells) != 0 {
		t.Errorf("got %d by %d with %d cells", got.W, got.H, len(got.Cells))
	}
}

// the next cell along reads the next two pixels and the next row of
// cells the next three, with no padding to hide a stride that is wrong.
// worked by hand on an image four across and six down that begins at
// (3,2), white but for five black pixels, numbered as above.
//
//	cell (0,0): black at (0,0) and (1,2), bits 0 and 5, pattern 33, so
//	  the opposite, 30, with white as the fg.
//	cell (1,0): black at (2,2), bit 4, pattern 16, black as the fg.
//	cell (0,1): black at (0,3), bit 0, pattern 1, so the opposite, 62.
//	cell (1,1): black at (3,4), bit 3, pattern 8.
//
// the glyphs are the Unicode sextants: U+1FB1C for 30, U+1FB0F for 16,
// U+1FB3B for 62 (the last of them) and U+1FB07 for 8.
func TestChopStepsTwoAcrossAndThreeDown(t *testing.T) {
	white, black := color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}
	img := image.NewRGBA(image.Rect(3, 2, 7, 8))
	for y := 2; y < 8; y++ {
		for x := 3; x < 7; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	for _, at := range [][2]int{{0, 0}, {1, 2}, {2, 2}, {0, 3}, {3, 4}} {
		img.SetRGBA(3+at[0], 2+at[1], black)
	}
	var got *Cells
	ends(t, "chop", func() { got = Chop(img) })
	if got.W != 2 || got.H != 2 {
		t.Fatalf("%d across and %d down, want 2 and 2", got.W, got.H)
	}
	for _, w := range []struct {
		x, y int
		want Cell
	}{
		{0, 0, Cell{Rune: 0x1FB1C, Fg: white, Bg: black, Pattern: 30}},
		{1, 0, Cell{Rune: 0x1FB0F, Fg: black, Bg: white, Pattern: 16}},
		{0, 1, Cell{Rune: 0x1FB3B, Fg: white, Bg: black, Pattern: 62}},
		{1, 1, Cell{Rune: 0x1FB07, Fg: black, Bg: white, Pattern: 8}},
	} {
		if cell := got.At(w.x, w.y); cell != w.want {
			t.Errorf("cell (%d, %d) = %+v, want %+v", w.x, w.y, cell, w.want)
		}
	}
}

// each cell reads its own six pixels, from an image that does not begin
// at the origin and does not divide evenly. worked by hand on an image
// three across and four down, white but for three black pixels, with
// each cell's pixels numbered 0 to 5 as the bits of a pattern.
//
//	cell (0,0), x 0-1, y 0-2: black at (1,1), which is bit 3.
//	cell (1,0), x 2-3, y 0-2: x 3 is past the edge and reads x 2,
//	  black at (2,0), so bits 0 and 1 are black.
//	cell (0,1), x 0-1, y 3-5: y 4 and 5 read y 3, black at (0,3),
//	  so bits 0, 2 and 4, the left column, are black.
//	cell (1,1): all six read (2,3), white.
//
// a pattern is tried only with its top left unset, and its opposite draws
// the same cell with the colours swapped, so the patterns are 8, 60 (the
// opposite of 3), 42 (the opposite of 21) and none, a space.
//
// the glyphs are the sextants of the Unicode rule: U+1FB07 for 8,
// U+1FB39 for 60, and for 42 the right half block U+2590.
func TestChopReadsEachCellsOwnSixPixels(t *testing.T) {
	white, black := color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}
	img := image.NewRGBA(image.Rect(5, 7, 8, 11))
	for y := 7; y < 11; y++ {
		for x := 5; x < 8; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	for _, at := range [][2]int{{1, 1}, {2, 0}, {0, 3}} {
		img.SetRGBA(5+at[0], 7+at[1], black)
	}
	var got *Cells
	ends(t, "chop", func() { got = Chop(img) })
	if got.W != 2 || got.H != 2 {
		t.Fatalf("%d across and %d down, want 2 and 2", got.W, got.H)
	}
	for _, w := range []struct {
		x, y int
		want Cell
	}{
		{0, 0, Cell{Rune: 0x1FB07, Fg: black, Bg: white, Pattern: 8}},
		{1, 0, Cell{Rune: 0x1FB39, Fg: white, Bg: black, Pattern: 60}},
		{0, 1, Cell{Rune: 0x2590, Fg: white, Bg: black, Pattern: 42}},
		{1, 1, Cell{Rune: ' ', Fg: white, Bg: white, Pattern: 0}},
	} {
		if cell := got.At(w.x, w.y); cell != w.want {
			t.Errorf("cell (%d, %d) = %+v, want %+v", w.x, w.y, cell, w.want)
		}
	}
}
