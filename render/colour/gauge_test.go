package colour

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// grey is a flat grey picture.
func grey(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	return img
}

// A wider frame cuts bars above and below, a narrower one either side,
// and the same shape cuts nothing; the bars are black and the middle
// is untouched.
func TestMasked(t *testing.T) {
	img := grey(100, 50) // 2 to 1 in square pixels
	wide := Masked(img, 4, 1)
	if wide.RGBAAt(50, 2).R != 0 || wide.RGBAAt(50, 25).R != 128 ||
		wide.RGBAAt(50, 47).R != 0 {
		t.Error("a wider frame did not bar above and below")
	}
	tall := Masked(img, 1, 1)
	if tall.RGBAAt(2, 25).R != 0 || tall.RGBAAt(50, 25).R != 128 ||
		tall.RGBAAt(97, 25).R != 0 {
		t.Error("a narrower frame did not bar either side")
	}
	same := Masked(img, 2, 1)
	if same.RGBAAt(0, 0).R != 128 || same.RGBAAt(99, 49).R != 128 {
		t.Error("the same shape cut something")
	}
	// a sextant pixel is three wide to four tall: the picture is
	// narrower than its pixel count says
	sext := Masked(img, 1.5, 0.75)
	if sext.RGBAAt(50, 0).R != 128 {
		t.Error("with sextant pixels a 1.5 frame should fit a 100 by 50")
	}
}

// Grain lightens and darkens in soft clumps of the size asked, by no
// more than the amount, the same for a seed; no grain is the picture.
func TestGrained(t *testing.T) {
	img := grey(60, 30)
	g := Grained(img, 3, 0.3, 7)
	seen := map[uint8]bool{}
	near, far := 0.0, 0.0
	for y := 0; y < 30; y++ {
		for x := 3; x < 60; x++ {
			v := g.RGBAAt(x, y).R
			seen[v] = true
			near += math.Abs(float64(v) - float64(g.RGBAAt(x-1, y).R))
			far += math.Abs(float64(v) - float64(g.RGBAAt(x-3, y).R))
			lin := Decode(v) / Decode(128)
			if lin < 0.69 || lin > 1.31 {
				t.Fatalf("grain moved a pixel to %v of what it was", lin)
			}
		}
	}
	// soft: a pixel is nearer its neighbour than the one three away
	if len(seen) < 10 || !(near < 0.6*far) {
		t.Errorf("%d levels; neighbours differ %.0f, three apart %.0f", len(seen), near, far)
	}
	if h := Grained(img, 3, 0.3, 7); h.RGBAAt(17, 9) != g.RGBAAt(17, 9) {
		t.Error("the grain changed between two asks")
	}
	if plain := Grained(img, 0, 0, 7); plain.RGBAAt(17, 9).R != 128 {
		t.Error("no grain changed the picture")
	}
}
