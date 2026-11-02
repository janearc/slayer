package terminal

import (
	"math"
	"strings"
	"testing"
)

// a terminal that says it is zero cells across or down, as a fresh one may, is
// an error that names how big it said, not a screen of no size. a fresh pty
// says 0 by 0.
func TestMeasuringATerminalOfNoSizeIsAnError(t *testing.T) {
	for _, w := range []struct {
		cols, rows int
		said       string
	}{
		{0, 0, "says it is 0 by 0 cells"},
		{0, 24, "says it is 0 by 24 cells"},
		{80, 0, "says it is 80 by 0 cells"},
	} {
		master, slave := ptyPair(t)
		if w.cols != 0 || w.rows != 0 {
			resize(t, master, w.cols, w.rows, 0, 0)
		}
		got, err := Measure(slave)
		if err == nil || !strings.Contains(err.Error(), w.said) || !strings.Contains(err.Error(), slave.Name()) {
			t.Errorf("%d by %d: got %+v, %v; want an error naming the file and saying %q",
				w.cols, w.rows, got, err, w.said)
		}
	}
}

// a screen is its cells, the pixels two a cell across and three down, and the
// shape of a pixel, its width over its height, from the cell's size in pixels
// when the terminal says it, else 0.75, which is a cell twice as tall as it is
// wide: (w/2) / (2w/3) = 3/4.
//
// worked by hand, cols by rows then pixels: a cell is x/cols across and y/rows
// down, a pixel half of that across and a third of it down.
//
//	80 by 24 at 720 by 480: a cell 9 by 20, a pixel 4.5 by 6.67,
//	  4.5 / (20/3) = 0.675.
//	1 by 1 at 1 by 1: a cell 1 by 1, a pixel (1/2) / (1/3) = 1.5.
//
// a terminal that gives zero pixels in either direction has not said: the guess
// is used and Measured is false.
func TestMeasuringATerminalGivesCellsPixelsAndTheShapeOfAPixel(t *testing.T) {
	for _, w := range []struct {
		what                   string
		cols, rows, xpix, ypix int
		want                   Screen
	}{
		{"no pixels given", 80, 24, 0, 0, Screen{80, 24, 160, 72, 0.75, false}},
		{"one cell, no pixels", 1, 1, 0, 0, Screen{1, 1, 2, 3, 0.75, false}},
		{"measured", 80, 24, 720, 480, Screen{80, 24, 160, 72, 0.675, true}},
		{"the smallest measured", 1, 1, 1, 1, Screen{1, 1, 2, 3, 1.5, true}},
		{"no width in pixels", 80, 24, 0, 480, Screen{80, 24, 160, 72, 0.75, false}},
		{"no height in pixels", 80, 24, 720, 0, Screen{80, 24, 160, 72, 0.75, false}},
	} {
		master, slave := ptyPair(t)
		resize(t, master, w.cols, w.rows, w.xpix, w.ypix)
		got, err := Measure(slave)
		if err != nil {
			t.Errorf("%s: %v", w.what, err)
			continue
		}
		if got.Cols != w.want.Cols || got.Rows != w.want.Rows || got.W != w.want.W || got.H != w.want.H ||
			got.Measured != w.want.Measured || math.Abs(got.Pixel-w.want.Pixel) > 1e-12 {
			t.Errorf("%s: got %+v, want %+v", w.what, got, w.want)
		}
	}
}
