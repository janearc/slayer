// Package frame stores rendered linear light, coverage, and first-hit depth in
// a pixel grid. It is the handoff between tracing a scene and developing or
// displaying the resulting image.
//
// A frame is a plain container. It checks where a pixel goes, not what the
// pixel holds.
package frame

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// What a frame counts in, each defined.
const (
	// none is no pixels, no place and no row. It is defined as zero.
	none = 0
	// endless is the sign math.Inf takes for plus infinity. It is defined
	// by that function.
	endless = 1
	// pixelFloats is the float32 values in a Pixel: R, G, B, Cover and
	// Depth. It is defined by the type.
	pixelFloats = 5
	// floatBytes is the bytes in a float32. It is defined by the type:
	// 32 bits of 8.
	floatBytes = 4
	// pixelBytes is the memory of one Pixel. It is defined by the type.
	pixelBytes = pixelFloats * floatBytes
	// mostPixels is the most pixels whose memory, in bytes, still fits in
	// an int. A frame of more cannot be sized. It is defined by the
	// arithmetic.
	mostPixels = math.MaxInt / pixelBytes
)

// Pixel stores linear RGB, coverage, and first-hit depth. New initializes miss
// depth to +Inf; a zero-value Pixel instead has depth zero, meaning a hit at
// the camera.
//
// Light has no ceiling. Cover runs from 0 to 1, as the part of the pixel that
// something covers. Depth is the distance to what the ray met, or +Inf for
// nothing. Code that reads a pixel assumes this. The type does not enforce
// it, and Set stores what it is given.
type Pixel struct {
	R, G, B float32 //ninja:vow name a colour is r, g, b at every call site
	Cover   float32
	Depth   float32
}

// Light returns the pixel's RGB as maths.Light.
func (pixel Pixel) Light() maths.Light {
	return maths.RGB(float64(pixel.R), float64(pixel.G), float64(pixel.B))
}

// Frame stores pixels in row-major order from the top left: the pixel at
// column x and row y is Pix[y*W+x]. New makes a frame whose W and H are not
// negative and whose Pix holds W times H pixels.
//
// A Frame built as a literal holds that only if its builder keeps it. At and
// Set trust W, H and Pix. A Pix shorter than W times H panics with an index
// error, not the refusal of a place past the edge. A wrong W reads the wrong
// pixel and says nothing.
//
// A frame holds only W, H and Pix, and a pixel is a whole value in one slice.
// Goroutines may call Set on different places of one frame at once, and At on
// any place. Two calls on one place, one of them a Set, race. So does
// changing W, H or Pix while others use the frame.
//
// A frame of no width or no height is allowed. It holds no pixels, so every
// place in it is past the edge.
type Frame struct {
	W, H int //ninja:vow name width and height are W, H at every call site
	Pix  []Pixel
}

var (
	errSize    = errors.New("frame: a frame is no size below nought")
	errOutside = errors.New("frame: a pixel past the frame's edge")
	errTooBig  = errors.New("frame: more pixels than a frame can be sized for")
)

// New allocates a width by height frame with black light, zero coverage, and
// infinite depth. A size of zero is allowed. A size below zero panics with
// errSize. Storage is width times height pixels, twenty bytes each.
//
// A pair whose memory, in bytes, does not fit in an int panics with errTooBig,
// since the product would wrap and leave W and H larger than Pix. A pair
// below that is not refused here, even when the machine cannot hold it: the
// allocator fails, by a runtime panic or by ending the process. A caller that
// takes sizes from a file or a person must bound them first.
func New(width, height int) *Frame {
	if width < none || height < none {
		panic(errSize)
	}
	if height > none && width > mostPixels/height {
		panic(errTooBig)
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

// At returns the pixel at column x and row y. Out-of-bounds coordinates panic
// with errOutside rather than indexing into another row.
func (grid *Frame) At(x, y int) Pixel {
	if grid.outside(x, y) {
		panic(errOutside)
	}
	return grid.Pix[y*grid.W+x]
}

// Set stores pixel at column x and row y, exactly as given. Out-of-bounds
// coordinates panic with errOutside.
//
// Set does not look at the pixel. NaN, an infinity, a negative light, a Cover
// outside 0 to 1 and a NaN or negative Depth are all stored. Keeping a pixel
// as Pixel describes it is the caller's. Where it is checked today is in the
// callers: raw.Write refuses non-finite light when it saves a frame.
//
// Set may be called from several goroutines at once on different places of
// one frame. Frame says what races.
func (grid *Frame) Set(x, y int, pixel Pixel) {
	if grid.outside(x, y) {
		panic(errOutside)
	}
	grid.Pix[y*grid.W+x] = pixel
}
