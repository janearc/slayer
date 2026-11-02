package colour

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/maths/noise"
)

// Masked is a picture with the gauge's frame cut into it: black bars
// above and below, or either side, so that what is left is aspect wide
// against tall, measured in square pixels.
//
// pixel is the picture's own pixel's width over its height. A picture
// already that shape is left alone.
func Masked(img *image.RGBA, aspect, pixel float64) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if pixel <= 0 {
		pixel = 1
	}
	out := clone(img)
	have := float64(w) * pixel / float64(h)
	black := color.RGBA{0, 0, 0, 255}
	switch {
	case aspect > have: // wider than the picture: bars above and below
		keep := int(math.Round(float64(h) * have / aspect))
		top := (h - keep) / 2
		for y := 0; y < h; y++ {
			if y < top || y >= top+keep {
				for x := 0; x < w; x++ {
					out.SetRGBA(b.Min.X+x, b.Min.Y+y, black)
				}
			}
		}
	case aspect < have: // narrower: bars either side
		keep := int(math.Round(float64(w) * aspect / have))
		left := (w - keep) / 2
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if x < left || x >= left+keep {
					out.SetRGBA(b.Min.X+x, b.Min.Y+y, black)
				}
			}
		}
	}
	return out
}

// Grained is a picture with film grain on it: soft clumps about size
// pixels across, each lightening or darkening what is under it by up
// to amount, in linear light, the same clumps every time for a seed.
// Nought size or amount is the picture as it was.
//
// Clumps finer than a pixel average out inside it: the amount falls by
// the size, so a fine gauge at a coarse picture is nearly clean, which
// is what a fine gauge is for.
func Grained(img *image.RGBA, size, amount float64, seed uint64) *image.RGBA {
	b := img.Bounds()
	out := clone(img)
	if size <= 0 || amount <= 0 {
		return out
	}
	if size < 1 {
		amount, size = amount*size, 1
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			gx, gy := (float64(x)+0.5)/size, (float64(y)+0.5)/size
			k := 1 + amount*noise.Gradient(seed, gx, gy, 0.5)
			c := img.RGBAAt(b.Min.X+x, b.Min.Y+y)
			out.SetRGBA(b.Min.X+x, b.Min.Y+y, color.RGBA{
				Encode(Decode(c.R) * k),
				Encode(Decode(c.G) * k),
				Encode(Decode(c.B) * k), 255})
		}
	}
	return out
}

// clone is a new picture of img's bounds holding img's pixels, copied a
// row at a time from where each row starts, so a picture cut from a
// larger one, whose rows are the larger one's width apart, copies whole.
func clone(img *image.RGBA) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		from, to := img.PixOffset(b.Min.X, y), out.PixOffset(b.Min.X, y)
		copy(out.Pix[to:to+4*b.Dx()], img.Pix[from:from+4*b.Dx()])
	}
	return out
}
