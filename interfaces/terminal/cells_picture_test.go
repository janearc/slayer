package terminal

import (
	"image"
	"image/color"
	"testing"
)

// a cell is two squares across and three down, bit 0 the top left, bit 1
// the top right, bits 2 and 3 the middle pair, 4 and 5 the bottom. a grid
// of two by two cells is an image four across and six down, and each set
// bit shows its cell's fg and each unset bit its bg.
//
// worked by hand. the cells, with fg f and bg b, and their patterns:
// top left 1 (only the top left square), top right 32 (only the bottom
// right), bottom left 42 (the right column), bottom right 21 (the left
// column). the six rows then read, left to right:
//
//	f0 b0 b1 b1
//	b0 b0 b1 b1
//	b0 b0 b1 f1
//	b2 f2 f3 b3
//	b2 f2 f3 b3
//	b2 f2 f3 b3
func TestPictureShowsEachSquareInItsCellsColour(t *testing.T) {
	f := [4]color.RGBA{{10, 0, 0, 255}, {20, 0, 0, 255}, {30, 0, 0, 255}, {40, 0, 0, 255}}
	b := [4]color.RGBA{{0, 10, 0, 255}, {0, 20, 0, 255}, {0, 30, 0, 255}, {0, 40, 0, 255}}
	c := &Cells{W: 2, H: 2}
	for i, pattern := range []uint8{1, 32, 42, 21} {
		c.Cells = append(c.Cells, Cell{Fg: f[i], Bg: b[i], Pattern: pattern})
	}
	want := [6][4]color.RGBA{
		{f[0], b[0], b[1], b[1]},
		{b[0], b[0], b[1], b[1]},
		{b[0], b[0], b[1], f[1]},
		{b[2], f[2], f[3], b[3]},
		{b[2], f[2], f[3], b[3]},
		{b[2], f[2], f[3], b[3]},
	}
	var img *image.RGBA
	ends(t, "picture", func() { img = Picture(c) })
	if got := img.Bounds(); got != image.Rect(0, 0, 4, 6) {
		t.Fatalf("bounds %v, want 4 across and 6 down from the origin", got)
	}
	for y := range 6 {
		for x := range 4 {
			if got := img.RGBAAt(x, y); got != want[y][x] {
				t.Errorf("pixel (%d, %d) = %v, want %v", x, y, got, want[y][x])
			}
		}
	}
}

// a grid wider than it is tall, so a width read as a height shows. three
// cells across and one down is six across and three down, whatever the
// patterns: here every square is a bg, so the whole image is the bg.
func TestPictureOfARowOfCellsIsSixByThree(t *testing.T) {
	bg := color.RGBA{7, 8, 9, 255}
	c := &Cells{W: 3, H: 1, Cells: make([]Cell, 3)}
	for i := range c.Cells {
		c.Cells[i].Bg = bg
	}
	img := Picture(c)
	if got := img.Bounds(); got != image.Rect(0, 0, 6, 3) {
		t.Fatalf("bounds %v, want 6 across and 3 down from the origin", got)
	}
	for y := range 3 {
		for x := range 6 {
			if got := img.RGBAAt(x, y); got != bg {
				t.Errorf("pixel (%d, %d) = %v, want the bg %v", x, y, got, bg)
			}
		}
	}
}
