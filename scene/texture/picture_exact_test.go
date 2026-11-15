package texture

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// oldAt is Picture.At as it was written, each tap through the image's At and
// a power per channel.
func oldAt(pic Picture, p maths.Vec) maths.Light {
	if pic.Image == nil || pic.Image.Bounds().Empty() {
		return maths.Light{}
	}
	d := p.Unit()
	lon := math.Atan2(d.Z, d.X) + pic.Turn
	lat := math.Asin(math.Max(-1, math.Min(1, d.Y)))
	u := lon/(2*math.Pi) + 0.5
	u -= math.Floor(u)
	v := 0.5 - lat/math.Pi
	b := pic.Image.Bounds()
	w, h := b.Dx(), b.Dy()
	x, y := u*float64(w)-0.5, v*float64(h)-0.5
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	sample := func(px, py int) maths.Light {
		px = ((px % w) + w) % w
		py = max(0, min(h-1, py))
		r, g, bl, _ := pic.Image.At(b.Min.X+px, b.Min.Y+py).RGBA()
		return maths.RGB(srgb.ToLinear(float64(r)/65535),
			srgb.ToLinear(float64(g)/65535),
			srgb.ToLinear(float64(bl)/65535))
	}
	top := sample(x0, y0).Scale(1 - fx).Add(sample(x0+1, y0).Scale(fx))
	bottom := sample(x0, y0+1).Scale(1 - fx).
		Add(sample(x0+1, y0+1).Scale(fx))
	l := top.Scale(1 - fy).Add(bottom.Scale(fy))
	if pic.Gain != 0 {
		l = l.Scale(pic.Gain)
	}
	return l
}

// randomImages is one image of each kind the picture may be given, of
// seeded random pixels, each with a bounds that does not start at zero.
func randomImages(r *rand.Rand) map[string]image.Image {
	rect := image.Rect(3, 5, 3+37, 5+19)
	rgba, nrgba, gray := image.NewRGBA(rect), image.NewNRGBA(rect), image.NewGray(rect)
	rgba64, ycc := image.NewRGBA64(rect), image.NewYCbCr(rect, image.YCbCrSubsampleRatio420)
	pal := image.NewPaletted(rect, color.Palette{color.RGBA{10, 20, 30, 255},
		color.RGBA{200, 100, 50, 255}, color.RGBA{0, 0, 0, 0}, color.RGBA{255, 255, 255, 255}})
	for _, pix := range [][]uint8{rgba.Pix, nrgba.Pix, gray.Pix, rgba64.Pix, ycc.Y, ycc.Cb, ycc.Cr} {
		for i := range pix {
			pix[i] = uint8(r.Intn(256))
		}
	}
	for i := 0; i < len(rgba.Pix); i += 4 {
		// premultiplied: no channel above alpha
		a := rgba.Pix[i+3]
		for c := 0; c < 3; c++ {
			rgba.Pix[i+c] = uint8(int(rgba.Pix[i+c]) * int(a) / 255)
		}
	}
	for i := 0; i < len(rgba64.Pix); i += 8 {
		for c := 0; c < 3; c++ { // alpha high byte 255 keeps colour valid
			rgba64.Pix[i+6], rgba64.Pix[i+7] = 0xff, 0xff
		}
	}
	for i := range pal.Pix {
		pal.Pix[i] = uint8(r.Intn(4))
	}
	return map[string]image.Image{"RGBA": rgba, "NRGBA": nrgba, "Gray": gray,
		"RGBA64": rgba64, "YCbCr": ycc, "Paletted": pal}
}

// every tap reads as the old arithmetic read it, bit for bit, in every
// image type, at seeded random directions, turns and gains.
func TestPictureAtIsTheOldArithmeticBitForBit(t *testing.T) {
	r := rand.New(rand.NewSource(20260910))
	for name, img := range randomImages(r) {
		for n := 0; n < 20000; n++ {
			pic := Picture{Image: img, Turn: r.Float64()*8 - 4}
			if n%3 == 0 {
				pic.Gain = r.Float64() * 3
			}
			p := maths.V(r.NormFloat64(), r.NormFloat64(), r.NormFloat64()).Scale(r.Float64()*9 + 0.1)
			got, want := pic.At(p), oldAt(pic, p)
			if math.Float64bits(got.R) != math.Float64bits(want.R) ||
				math.Float64bits(got.G) != math.Float64bits(want.G) ||
				math.Float64bits(got.B) != math.Float64bits(want.B) {
				t.Fatalf("%s at %v: %+v, was %+v", name, p, got, want)
			}
		}
	}
}

// the tables hold what the curve gives for every part.
func TestPictureTablesAreTheCurve(t *testing.T) {
	for i := 0; i < 65536; i++ {
		if math.Float64bits(linear16()[i]) != math.Float64bits(srgb.ToLinear(float64(i)/65535)) {
			t.Fatalf("linear16[%d]", i)
		}
	}
	for i := 0; i < 256; i++ {
		if math.Float64bits(linear8[i]) != math.Float64bits(srgb.ToLinear(float64(i*0x101)/65535)) {
			t.Fatalf("linear8[%d]", i)
		}
	}
}
