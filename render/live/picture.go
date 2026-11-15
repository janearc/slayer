package live

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/janearc/slayer/render/frame"
)

// canvas is the pane's pixels with a picture laid in its place: black
// bars where it does not fill, and the picture resampled onto the
// rectangle the placing gives it.
func canvas(picture image.Image, place placing) *image.RGBA {
	out := image.NewRGBA(image.Rect(none, none, place.paneW, place.paneH))
	black := image.NewUniform(color.RGBA{A: math.MaxUint8})
	draw.Draw(out, out.Bounds(), black, image.Point{}, draw.Src)
	resample(picture, out, place.shown)
	return out
}

// resample lays from onto the rectangle into of out: each pixel of the
// rectangle the mean of the pixels of from it covers, or the nearest
// one where it covers less than one. A coarse picture grows as blocks;
// a fine one shrinks without losing what lies between its samples.
func resample(from image.Image, out *image.RGBA, into image.Rectangle) {
	source := from.Bounds()
	spanX := float64(source.Dx()) / float64(max(one, into.Dx()))
	spanY := float64(source.Dy()) / float64(max(one, into.Dy()))
	for y := into.Min.Y; y < into.Max.Y; y++ {
		top := source.Min.Y + int(float64(y-into.Min.Y)*spanY)
		bottom := max(top+one, source.Min.Y+
			int(float64(y-into.Min.Y+one)*spanY))
		for x := into.Min.X; x < into.Max.X; x++ {
			left := source.Min.X + int(float64(x-into.Min.X)*spanX)
			right := max(left+one, source.Min.X+
				int(float64(x-into.Min.X+one)*spanX))
			out.SetRGBA(x, y, meanOf(from, image.Rect(left, top,
				min(right, source.Max.X), min(bottom, source.Max.Y))))
		}
	}
}

// meanOf is the mean colour of a rectangle of a picture, opaque.
func meanOf(from image.Image, area image.Rectangle) color.RGBA {
	var red, green, blue, count uint64
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			r, g, b, _ := from.At(x, y).RGBA()
			red, green, blue = red+uint64(r), green+uint64(g),
				blue+uint64(b)
			count++
		}
	}
	count = max(count, one)
	return color.RGBA{R: eight(red / count), G: eight(green / count),
		B: eight(blue / count), A: math.MaxUint8}
}

// eight is a sixteen bit channel as eight bits.
func eight(channel uint64) uint8 {
	return uint8(channel >> eightBits)
}

// eightBits is how far a sixteen bit channel shifts to eight. Defined.
const eightBits = 8

// scaled is a copy of light with its colour multiplied by factor, its
// coverage and depth as they were: an exposure changed after the light
// was drawn.
func scaled(light *frame.Frame, factor float64) *frame.Frame {
	out := &frame.Frame{W: light.W, H: light.H,
		Pix: make([]frame.Pixel, len(light.Pix))}
	gain := float32(factor)
	for i, pixel := range light.Pix {
		pixel.R, pixel.G, pixel.B = pixel.R*gain, pixel.G*gain,
			pixel.B*gain
		out.Pix[i] = pixel
	}
	return out
}

// gainOf is what a count of thirds of a stop multiplies light by.
func gainOf(thirds int) float64 {
	return math.Pow(stopBase, float64(thirds)/thirdsInStop)
}
