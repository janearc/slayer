package colour

import (
	"image"
	"image/color"
	"testing"

	"github.com/janearc/slayer/maths/noise"
)

// coloured is a picture whose pixels differ and whose three parts
// differ in each, so a grain taken to the wrong part or the wrong pixel
// shows. it is cut from a larger one at (3, 2), so its own corner is
// not the origin.
func coloured(w, h int) *image.RGBA {
	big := image.NewRGBA(image.Rect(0, 0, w+6, h+4))
	for y := 0; y < h+4; y++ {
		for x := 0; x < w+6; x++ {
			big.SetRGBA(x, y, color.RGBA{uint8(40 + 5*x), uint8(120 + 3*y), uint8(220 - 4*x - 2*y), 255})
		}
	}
	return big.SubImage(image.Rect(3, 2, 3+w, 2+h)).(*image.RGBA)
}

// grained is the picture grain makes of img, worked from the doc: the
// noise at the pixel's centre in units of the clump's size, at height
// one half, scaled by amount about one, and applied to each part in
// linear light.
func grained(img *image.RGBA, size, amount float64, seed uint64) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			k := 1 + amount*noise.Gradient(seed, (float64(x)+0.5)/size, (float64(y)+0.5)/size, 0.5)
			c := img.RGBAAt(b.Min.X+x, b.Min.Y+y)
			out.SetRGBA(b.Min.X+x, b.Min.Y+y, color.RGBA{
				Encode(Decode(c.R) * k), Encode(Decode(c.G) * k), Encode(Decode(c.B) * k), 255})
		}
	}
	return out
}

// within says two pictures are the same to a byte in each part: the
// grain is worked twice, and a fused multiply on one machine may round
// a byte over where another does not.
func within(t *testing.T, name string, got, want *image.RGBA) {
	t.Helper()
	if got.Bounds() != want.Bounds() {
		t.Fatalf("%s: bounds %v, want %v", name, got.Bounds(), want.Bounds())
	}
	diff := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g, w := got.RGBAAt(x, y), want.RGBAAt(x, y)
			if diff(g.R, w.R) > 1 || diff(g.G, w.G) > 1 || diff(g.B, w.B) > 1 || g.A != w.A {
				t.Errorf("%s: (%d, %d) is %v, want %v", name, x, y, g, w)
				return
			}
		}
	}
}

// grain is the noise at each pixel's centre, in clumps of size pixels,
// moving the light by up to amount: for a clump of a pixel or more the
// amount is as asked.
//
// for a finer one the clump is a pixel and the amount falls by the
// size, so that half a pixel at 0.4 is a pixel at 0.2. a size of 1.5
// keeps its own amount and clump.
func TestGrainFollowsTheNoiseAtEachPixelsCentre(t *testing.T) {
	img := coloured(12, 8)
	for _, c := range []struct {
		name             string
		size, amount     float64
		sizeIs, amountIs float64
	}{
		{"a clump of 2", 2, 0.3, 2, 0.3},
		{"a clump of 3", 3, 0.3, 3, 0.3},
		{"a clump of a pixel", 1, 0.3, 1, 0.3},
		{"a clump of 1.5", 1.5, 0.3, 1.5, 0.3},
		{"a clump of half a pixel", 0.5, 0.4, 1, 0.2},
		{"a clump of a quarter", 0.25, 0.8, 1, 0.2},
	} {
		within(t, c.name, Grained(img, c.size, c.amount, 11), grained(img, c.sizeIs, c.amountIs, 11))
	}
	// a different seed is a different field
	a, b := Grained(img, 2, 0.3, 11), Grained(img, 2, 0.3, 12)
	same := true
	for i := range a.Pix {
		same = same && a.Pix[i] == b.Pix[i]
	}
	if same {
		t.Error("two seeds made one grain")
	}
}

// no grain is the picture as it was, every part of every pixel, alpha among
// them: a zero size or amount, or a negative one, gives back what it was given,
// where grain draws every pixel opaque. the picture given is not changed by the
// grain made of it.
func TestNoGrainIsThePictureWithItsAlpha(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for i := 0; i < len(img.Pix); i += 4 {
		copy(img.Pix[i:], []byte{50, 100, 150, 200})
	}
	kept := append([]uint8(nil), img.Pix...)
	for _, c := range []struct{ size, amount float64 }{{0, 0.3}, {3, 0}, {0, 0}, {-1, 0.3}, {3, -0.3}} {
		out := Grained(img, c.size, c.amount, 7)
		for i := range out.Pix {
			if out.Pix[i] != kept[i] {
				t.Fatalf("size %v amount %v changed byte %d to %d", c.size, c.amount, i, out.Pix[i])
			}
		}
	}
	Grained(img, 2, 0.3, 7)
	for i := range img.Pix {
		if img.Pix[i] != kept[i] {
			t.Fatalf("grain changed the picture it was given at byte %d", i)
		}
	}
}
