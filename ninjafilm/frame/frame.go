// Package frame stores linear light, coverage, and depth in a pixel grid.
// Renderers supply these frames to the film and development stages.
package frame

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// what a frame counts in, each defined
const (
	// none is no pixels and no place. It is defined as zero.
	none = 0
	// endless is math.Inf's code for infinity above all numbers. It is
	// defined by that function.
	endless = 1
)

// Pixel is what came through one pixel of a frame. Pixels made other than by
// New have a Depth of zero. That reads as met right at the eye. One that met
// nothing says +Inf.
type Pixel struct {
	//ninja:vow gate a value, whose parts are the light
	R, G, B float32 //ninja:vow name a colour is r, g, b at every call site
	//ninja:vow gate renderers set how much of the pixel is covered
	Cover float32
	//ninja:vow gate renderers set the distance to what the ray met
	Depth float32
}

// NewPixel is a pixel of light, cover and depth.
func NewPixel(red, green, blue, cover, depth float32) Pixel {
	return Pixel{R: red, G: green, B: blue, Cover: cover, Depth: depth}
}

// Light is the pixel's light as a Light.
func (pixel Pixel) Light() maths.Light {
	return maths.Light{R: float64(pixel.R), G: float64(pixel.G),
		B: float64(pixel.B)}
}

// Frame is a grid of pixels, row by row from the top left.
type Frame struct {
	//ninja:vow gate renderers read the size they were given
	W, H int //ninja:vow name the width and height a renderer was given
	//ninja:vow gate renderers fill the rows in place
	Pix []Pixel
}

var (
	errSize    = errors.New("frame: a frame is no size below nought")
	errOutside = errors.New("frame: a pixel past the frame's edge")
)

// New is a frame of width by height pixels that have met nothing. They have no
// light and no cover and are infinitely far. Sizes less than zero are mistakes
// the caller made. They panic with errSize. Frames hold width times height
// pixels in memory.
func New(width, height int) *Frame {
	if width < none || height < none {
		panic(errSize)
	}
	pixels := make([]Pixel, width*height)
	made := &Frame{W: width, H: height, Pix: pixels}
	far := float32(math.Inf(endless))
	for index := range made.Pix {
		made.Pix[index].Depth = far
	}
	return made
}

// outside is whether a column and row are past the edge of the frame.
func (grid *Frame) outside(x, y int) bool {
	return uint(x) >= uint(grid.W) || uint(y) >= uint(grid.H)
}

// At is the pixel at column x, row y. Places past the edge of the frame
// panic with errOutside, rather than reading the next row's.
func (grid *Frame) At(x, y int) Pixel {
	if grid.outside(x, y) {
		panic(errOutside)
	}
	return grid.Pix[y*grid.W+x]
}

// Set puts a pixel at column x, row y. Places past the edge of the frame
// panic with errOutside, rather than writing the next row's.
func (grid *Frame) Set(x, y int, put Pixel) {
	if grid.outside(x, y) {
		panic(errOutside)
	}
	grid.Pix[y*grid.W+x] = put
}
