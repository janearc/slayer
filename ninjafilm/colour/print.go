package colour

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/grain"
)

// grainHeight is the height in the noise at which the grain is read.
var grainHeight = data.Picked("grain-height")

// Masked is a picture with the frame of the gauge cut into it. There are
// black bars above and below or on either side. What is left is aspect
// wide against tall, measured in square pixels.
//
// The pixel is the width over the height of the pixel of the picture.
// Pictures already that shape are left alone.
//
//ninja:vow practice slayer's pictures are masked to their gauge here
func Masked(img *image.RGBA, aspect, pixel float64) *image.RGBA {
	bounds := img.Bounds()
	across, down := bounds.Dx(), bounds.Dy()
	if pixel <= none {
		pixel = whole
	}
	out := clone(img)
	have := float64(across) * pixel / float64(down)
	switch {
	case aspect > have:
		// wider than the picture: bars above and below
		scaled := float64(down) * have / aspect
		keep := int(math.Round(scaled))
		top := (down - keep) / pair
		bars := bounds
		start := bounds.Min.Y + top
		bars.Min.Y, bars.Max.Y = start, start+keep
		blacken(out, bars)
	case aspect < have:
		// narrower: bars either side
		scaled := float64(across) * aspect / have
		keep := int(math.Round(scaled))
		left := (across - keep) / pair
		bars := bounds
		start := bounds.Min.X + left
		bars.Min.X, bars.Max.X = start, start+keep
		blacken(out, bars)
	}
	return out
}

// blacken makes black every pixel of a picture outside a rectangle.
func blacken(img *image.RGBA, keep image.Rectangle) {
	black := color.RGBA{A: byteMost}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !(image.Point{X: x, Y: y}).In(keep) {
				img.SetRGBA(x, y, black)
			}
		}
	}
}

// Grained is a picture with film grain on it. The clumps are soft and
// about size pixels across. Each lightens or darkens what is under it by
// up to amount, in linear light. They are the same clumps every time for
// a seed. Nought size or amount is the picture as it was.
//
// Clumps finer than a pixel average out inside it. The amount falls by
// the size. So a fine gauge at a coarse picture is nearly clean, which
// is what a fine gauge is for.
//
//ninja:vow practice slayer's pictures are grained to their gauge here
func Grained(img *image.RGBA, size, amount float64, seed uint64) *image.RGBA {
	bounds := img.Bounds()
	out := clone(img)
	if size <= none || amount <= none {
		return out
	}
	if size < whole {
		amount, size = amount*size, whole
	}
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			gx, gy := (float64(x)+half)/size, (float64(y)+half)/size
			noise := grain.Gradient(seed, gx, gy, grainHeight)
			shift := whole + amount*noise
			seen := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			red := Decode(seen.R) * shift
			green := Decode(seen.G) * shift
			blue := Decode(seen.B) * shift
			grained := encoded(red, green, blue)
			out.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, grained)
		}
	}
	return out
}

// encoded is three parts of linear light as an opaque pixel.
func encoded(red, green, blue float64) color.RGBA {
	return color.RGBA{R: Encode(red), G: Encode(green), B: Encode(blue),
		A: byteMost}
}

// clone is a new picture of the bounds of img holding its pixels. They
// are copied a row at a time from where each row starts. So a picture
// cut from a larger one copies whole. Its rows are the width of the
// larger one apart.
func clone(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	row := byteWidth * bounds.Dx()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		from := img.PixOffset(bounds.Min.X, y)
		to := out.PixOffset(bounds.Min.X, y)
		copy(out.Pix[to:to+row], img.Pix[from:from+row])
	}
	return out
}
