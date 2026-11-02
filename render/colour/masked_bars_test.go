package colour

import (
	"image"
	"image/color"
	"testing"
)

// marked is a picture whose every pixel is its own: red is the column,
// green the row, blue 77. nothing is black, so a bar shows, and a
// pixel the mask touched that it should not have is not the one it was.
func marked(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(10 + x), uint8(10 + y), 77, 255})
		}
	}
	return img
}

// a mask keeps a rectangle of the picture, to the pixel: what is in it
// is as it was, and what is out of it is black. the rectangle is worked
// by integer arithmetic on the shape. with have the picture's width
// over its height, in square pixels (width * pixel / height):
//
//	wider frame:   keep = round(h * have / aspect), bars (h - keep)/2 above
//	narrower:      keep = round(w * aspect / have), bars (w - keep)/2 beside
//
// all of these are 100 by 50, so have is 2 for square pixels.
func TestAMaskKeepsTheRectangleTheShapeLeaves(t *testing.T) {
	for _, c := range []struct {
		name           string
		aspect, pixel  float64
		x0, x1, y0, y1 int
	}{
		// have 2, aspect 4: keep = 50 * 2/4 = 25 rows, 25/2 = 12 above
		{"wider", 4, 1, 0, 100, 12, 37},
		// have 2, aspect 1: keep = 100 * 1/2 = 50 columns, 50/2 = 25 beside
		{"narrower", 1, 1, 25, 75, 0, 50},
		{"the same shape", 2, 1, 0, 100, 0, 50},
		// sextant pixels are 0.75 wide to 1 tall: have = 100*0.75/50 = 1.5
		{"the same shape in sextant pixels", 1.5, 0.75, 0, 100, 0, 50},
		// have 1.5, aspect 2.5: keep = 50 * 1.5/2.5 = 30 rows, 20/2 = 10 above
		{"wider in sextant pixels", 2.5, 0.75, 0, 100, 10, 40},
		// have 1.5, aspect 1: keep = round(100/1.5) = round(66.67) = 67
		// columns, 33/2 = 16 beside on the left and 17 on the right
		{"narrower in sextant pixels", 1, 0.75, 16, 83, 0, 50},
		// have 2, aspect 1.5: keep = 100 * 1.5/2 = 75 columns, 25/2 = 12 beside
		{"narrower, not square", 1.5, 1, 12, 87, 0, 50},
		// have 1.5, aspect 1.2: keep = 100 * 1.2/1.5 = 80 columns, 10 beside
		{"narrower in sextant pixels, not square", 1.2, 0.75, 10, 90, 0, 50},
		// a pixel of zero, or less, is read as square
		{"no pixel, narrower", 1, 0, 25, 75, 0, 50},
		{"no pixel, wider", 4, 0, 0, 100, 12, 37},
		{"a negative pixel, narrower", 1, -3, 25, 75, 0, 50},
	} {
		img := marked(100, 50)
		out := Masked(img, c.aspect, c.pixel)
		for y := 0; y < 50; y++ {
			for x := 0; x < 100; x++ {
				want := img.RGBAAt(x, y)
				if x < c.x0 || x >= c.x1 || y < c.y0 || y >= c.y1 {
					want = color.RGBA{0, 0, 0, 255}
				}
				if got := out.RGBAAt(x, y); got != want {
					t.Errorf("%s: (%d, %d) is %v, want %v", c.name, x, y, got, want)
					return
				}
			}
		}
	}
}
