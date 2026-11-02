package sheet

import (
	"image"
	"image/color"
	"testing"
)

func flat(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

var red = color.RGBA{255, 0, 0, 255}

// Three frames at two to a row make two rows, each cell the size of the
// largest frame, a sextant frame drawn at its true shape.
func TestFramesAreLaidOutInRows(t *testing.T) {
	s := New(2)
	s.Add("one", flat(10, 10, red), 1)
	s.Add("two", flat(10, 10, red), 0.75) // 30 by 40 once grown
	s.Add("three", flat(5, 5, red), 1)
	got := s.Image().Bounds()
	want := image.Rect(0, 0, gap+2*(30+gap), gap+2*(40+label+gap))
	if got != want {
		t.Errorf("sheet is %v, wanted %v", got, want)
	}
	if c := s.Image().RGBAAt(gap, gap+40+label+gap); c != red {
		t.Errorf("the third frame's corner is %v", c)
	}
}

// Every frame's label is printed in the strip beneath it.
func TestLabelsArePrinted(t *testing.T) {
	s := New(1)
	s.Add("smoke 2", flat(80, 20, red), 1)
	img := s.Image()
	inked := 0
	for y := gap + 20; y < gap+20+label; y++ {
		for x := gap; x < gap+80; x++ {
			if img.RGBAAt(x, y) == ink {
				inked++
			}
		}
	}
	if inked == 0 {
		t.Error("the label strip is empty")
	}
}

// An empty sheet is only its margin, not an error.
func TestAnEmptySheet(t *testing.T) {
	if b := New(3).Image().Bounds(); b.Dx() != 2*gap || b.Dy() != gap {
		t.Errorf("an empty sheet is %v", b)
	}
}
