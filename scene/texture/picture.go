package texture

import (
	"image"
	"math"
	"sync"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// Picture is an image wrapped round the y axis as the map of a sphere: an
// equirectangular picture, longitude across and latitude down, as nasa
// publishes the blue marble and the black marble.
//
// At a direction from the centre it is the picture's colour there, read as srgb
// and given back as linear light, blended between the four nearest pixels. Turn
// moves the picture's left edge round the sphere, in radians, and Gain scales
// the light, zero read as one.
//
// It is wrapped mirrored: its longitude grows from +X toward +Z, where the
// package's spin, and the earth's, run from +X toward -Z, so a map's west lands
// east and the world seen is its mirror. Nothing here turns it back. A
// picture of no pixels is dark.
type Picture struct {
	Image image.Image
	Turn  float64
	Gain  float64
}

// pointing is the point's direction, one unit long, with each square rounded
// before the squares are summed. Left to itself the compiler may fold one of
// the multiplies into an add, and which one it folds depends on what the sum
// is inlined into, so the last bit of a direction differed from one caller to
// the next and from one chip to the next. A picture's place must not: the same
// point is the same texel whoever asks. The zero vector stays zero.
func pointing(point maths.Vec) maths.Vec {
	length := math.Sqrt(float64(point.X*point.X) + float64(point.Y*point.Y) +
		float64(point.Z*point.Z))
	if length == nought {
		return point
	}
	return point.Scale(unity / length)
}

// mapped is where a direction falls on the picture, across and down, each
// from nought to one.
func (pic Picture) mapped(point maths.Vec) (across, down float64) {
	direction := pointing(point)
	longitude := math.Atan2(direction.Z, direction.X) + pic.Turn
	latitude := math.Asin(clamped(direction.Y))
	across = longitude/fullTurn + half
	across -= math.Floor(across)
	return across, half - latitude/math.Pi
}

// At is the picture's light in the direction of the point.
func (pic Picture) At(point maths.Vec) maths.Light {
	if pic.Image == nil || pic.Image.Bounds().Empty() {
		return noPaint
	}
	across, down := pic.mapped(point)
	bounds := pic.Image.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	x, y := across*float64(width)-half, down*float64(height)-half
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	texels := texelSource{pixels: reader(pic.Image), bounds: bounds}
	top := texels.at(x0, y0).Scale(unity - fx).
		Add(texels.at(x0+unity, y0).Scale(fx))
	bottom := texels.at(x0, y0+unity).Scale(unity - fx).
		Add(texels.at(x0+unity, y0+unity).Scale(fx))
	light := top.Scale(unity - fy).Add(bottom.Scale(fy))
	if pic.Gain != nought {
		light = light.Scale(pic.Gain)
	}
	return light
}

// texelSource is a picture's texels: its pixels, read as linear light, by
// their place counted from the picture's own corner.
type texelSource struct {
	pixels pixelReader
	bounds image.Rectangle
}

// at is the texel at a column and row, the columns wrapped round the
// picture and the rows held to its first and last.
func (texels texelSource) at(column, row int) maths.Light {
	width, height := texels.bounds.Dx(), texels.bounds.Dy()
	column = ((column % width) + width) % width
	row = max(nought, min(height-unity, row))
	red, green, blue := texels.pixels.linear(texels.bounds.Min.X+column,
		texels.bounds.Min.Y+row)
	return maths.RGB(red, green, blue)
}

// the numbers Water is judged by, each picked and held in
// picked-numbers.yaml
var (
	// waterOverRed is how many times more blue than red the sea is.
	waterOverRed = pickedNumber("water-blue-over-red")
	// waterOverGreen is how many times more blue than green the sea is.
	waterOverGreen = pickedNumber("water-blue-over-green")
)

// Water is whether the picture is water in the direction of the point: blue
// over red and green, as the sea is in a map of the earth by day.
func (pic Picture) Water(point maths.Vec) bool {
	colour := pic.At(point)
	return colour.B > colour.R*waterOverRed &&
		colour.B > colour.G*waterOverGreen
}

// byteValues is defined: the values a byte holds.
const byteValues = 256

// shortValues is defined: the values a 16 bit part holds.
const shortValues = 65536

// shortMost is defined: the largest 16 bit part.
const shortMost = 65535

// byteToShort is defined: the factor that makes a byte the 16 bit part
// with the same byte twice, as color.Color widens one.
const byteToShort = 0x101

// linear8 is linear light for each srgb byte, as the 16-bit part the
// byte makes, so the table agrees with linear16 where the two overlap.
var linear8 = linearBytes()

// linearBytes is the table of linear light for each srgb byte.
func linearBytes() (table [byteValues]float64) {
	for value := range table {
		table[value] = srgb.ToLinear(float64(value*byteToShort) / shortMost)
	}
	return table
}

// linearShorts is the table of linear light for each 16-bit srgb part.
func linearShorts() *[shortValues]float64 {
	var table [shortValues]float64
	for value := range table {
		table[value] = srgb.ToLinear(float64(value) / shortMost)
	}
	return &table
}

// linear16 is linear light for each 16-bit srgb part, filled on first use.
// A color.Color that returns a channel above 65535 breaks the color.Color
// contract and is not supported: the table has no entry for it.
var linear16 = sync.OnceValue(linearShorts)

// pixelReader reads an image's pixels as linear light, each channel what
// srgb.ToLinear makes of the image's own 16-bit part over 65535.
type pixelReader interface {
	linear(x, y int) (red, green, blue float64)
}

// rgbaPixels are the pixels of an image.RGBA, read without boxing a
// colour.
type rgbaPixels struct{ image *image.RGBA }

// linear is the pixel at x and y, by the byte table.
func (pixels rgbaPixels) linear(x, y int) (float64, float64, float64) {
	colour := pixels.image.RGBAAt(x, y)
	return linear8[colour.R], linear8[colour.G], linear8[colour.B]
}

// nrgbaPixels are the pixels of an image.NRGBA, read without boxing a
// colour.
type nrgbaPixels struct{ image *image.NRGBA }

// linear is the pixel at x and y, by the 16-bit table.
func (pixels nrgbaPixels) linear(x, y int) (float64, float64, float64) {
	red, green, blue, _ := pixels.image.NRGBAAt(x, y).RGBA()
	table := linear16()
	return table[red], table[green], table[blue]
}

// grayPixels are the pixels of an image.Gray, read without boxing a
// colour.
type grayPixels struct{ image *image.Gray }

// linear is the pixel at x and y, by the byte table.
func (pixels grayPixels) linear(x, y int) (float64, float64, float64) {
	value := linear8[pixels.image.GrayAt(x, y).Y]
	return value, value, value
}

// anyPixels are the pixels of any other image, read through its At.
type anyPixels struct{ image image.Image }

// linear is the pixel at x and y, by the 16-bit table.
func (pixels anyPixels) linear(x, y int) (float64, float64, float64) {
	red, green, blue, _ := pixels.image.At(x, y).RGBA()
	table := linear16()
	return table[red], table[green], table[blue]
}

// reader is a way to read an image's pixels as linear light. the common
// image types are read without boxing a colour; any other is read
// through its At, and every path converts by a table.
func reader(img image.Image) pixelReader {
	switch typed := img.(type) {
	case *image.RGBA:
		return rgbaPixels{typed}
	case *image.NRGBA:
		return nrgbaPixels{typed}
	case *image.Gray:
		return grayPixels{typed}
	}
	return anyPixels{img}
}
