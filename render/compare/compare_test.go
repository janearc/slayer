package compare

import (
	"image"
	"image/color"
	"testing"
)

// flat is an image of one colour.
func flat(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

var clay = color.RGBA{220, 150, 124, 255}

// The same colour at both sizes is honest, and every cell is compared.
func TestTheSamePictureIsHonest(t *testing.T) {
	r := Cells(flat(24, 36, clay), flat(8, 9, clay), 3, 4)
	if !r.Honest() || r.Compared != 8*9 || r.Worst != 0 {
		t.Errorf("one colour at two sizes: %+v", r)
	}
}

// A terminal picture a shade off from the real one is caught.
func TestAPictureOfItsOwnIsCaught(t *testing.T) {
	off := color.RGBA{200, 150, 124, 255}
	if r := Cells(flat(24, 36, clay), flat(8, 9, off), 3, 4); r.Honest() {
		t.Errorf("a different red passed: %+v", r)
	}
}

// A cell whose real pixels hold an edge is left out rather than blamed on
// the terminal, which draws two colours a cell and cannot draw an edge.
func TestEdgesAreLeftOut(t *testing.T) {
	real := flat(24, 36, clay)
	for y := 0; y < 36; y++ {
		for x := 0; x < 3; x++ {
			real.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	r := Cells(real, flat(8, 9, clay), 3, 4)
	if r.Compared != 8*9-6*3 || !r.Honest() {
		t.Errorf("an edge down the first cells: %+v", r)
	}
}
