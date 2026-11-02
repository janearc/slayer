// Package frame stores rendered linear light, coverage, and first-hit depth in
// a pixel grid. It is the handoff between tracing a scene and developing or
// displaying the resulting image.
package frame

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// Pixel stores linear RGB, coverage, and first-hit depth. New initializes miss
// depth to +Inf; a zero-value Pixel instead has depth zero, meaning a hit at
// the camera.
type Pixel struct {
	R, G, B float32 // linear light, no ceiling
	Cover   float32 // 0 to 1: how much of the pixel something covers
	Depth   float32 // distance to what the ray met; +Inf for nothing
}

// Light returns the pixel’s RGB as maths.Light.
func (p Pixel) Light() maths.Light {
	return maths.Light{R: float64(p.R), G: float64(p.G), B: float64(p.B)}
}

// Frame stores pixels in row-major order from the top left.
type Frame struct {
	W, H int
	Pix  []Pixel
}

var (
	errSize    = errors.New("frame: a frame is no size below nought")
	errOutside = errors.New("frame: a pixel past the frame's edge")
)

// New allocates a w by h frame with black light, zero coverage, and infinite
// depth. Negative dimensions panic with errSize. Storage grows with w times h.
func New(w, h int) *Frame {
	if w < 0 || h < 0 {
		panic(errSize)
	}
	f := &Frame{W: w, H: h, Pix: make([]Pixel, w*h)}
	inf := float32(math.Inf(1))
	for i := range f.Pix {
		f.Pix[i].Depth = inf
	}
	return f
}

// At returns the pixel at column x and row y. Out-of-bounds coordinates panic
// with errOutside rather than indexing into another row.
func (f *Frame) At(x, y int) Pixel {
	if uint(x) >= uint(f.W) || uint(y) >= uint(f.H) {
		panic(errOutside)
	}
	return f.Pix[y*f.W+x]
}

// Set stores p at column x and row y. Out-of-bounds coordinates panic with
// errOutside.
func (f *Frame) Set(x, y int, p Pixel) {
	if uint(x) >= uint(f.W) || uint(y) >= uint(f.H) {
		panic(errOutside)
	}
	f.Pix[y*f.W+x] = p
}
