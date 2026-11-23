//go:build darwin

package metal

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"math"
	"runtime"
)

// what Grid refuses
var (
	errNoNumbers = errors.New("metal: a pixel needs a number at least")
	errTooBig    = errors.New("metal: an input over two gigabytes")
	errGridSize  = errors.New("metal: grid dimensions overflow native or host sizes")
)

// Grid runs the kernel once for every pixel of a width by height picture and
// gives back per numbers a pixel, row by row. The inputs are the kernel's
// buffers from 0, as given; the picture is its next buffer, and the band's
// first row the one after that.
//
// It draws the picture in bands, one command each; the inputs are made into
// buffers once for all the bands. rows is the most rows a band holds, which
// sizes the band buffers; zero or less lets a band hold the whole picture. How
// many rows each band does hold is decided by the bands' times
// (band_heights.go): the first is one row, and each after it at most twice
// the last, unless FixBands fixes them.
func (k *Kernel) Grid(width, height, rows, per int,
	inputs [][]byte) ([]float32, error) {
	if k.ctx == nil {
		return nil, errors.New("metal: the kernel is closed")
	}
	if per <= 0 {
		return nil, errNoNumbers
	}
	for _, input := range inputs {
		if len(input) > math.MaxInt32 {
			return nil, errTooBig
		}
	}
	if width <= 0 || height <= 0 {
		return []float32{}, nil
	}
	// Check the product by division before allocating or narrowing
	// dimensions to C ints.
	if !gridFits(width, height, per) {
		return nil, errGridSize
	}
	out := make([]float32, width*height*per)
	if rows <= 0 || rows > height {
		rows = height
	}
	// the streamed path copies each input straight into its retained
	// Metal buffer, with no stop in C memory on the way
	stream, err := k.openBands(width, height, rows, per, inputs)
	if err != nil {
		return nil, err
	}
	defer C.slayer_metal_grid_close(stream)
	// k is not read after k.ctx, so without this its finalizer could
	// free the context while the gpu still draws with it
	defer runtime.KeepAlive(k)
	place := func(first, down int) []float32 {
		return out[first*width*per : (first+down)*width*per]
	}
	drawing := k.drawingOf(stream, height, rows)
	if err := drawing.draw(place, keptInPlace); err != nil {
		return nil, err
	}
	return out, nil
}

// keptInPlace is the consumer of a band already copied where it belongs.
func keptInPlace(int, int, []float32) error { return nil }

// gridFits checks positive dimensions before allocation and C-int narrowing.
// The caller has already handled empty dimensions and nonpositive channels.
func gridFits(width, height, per int) bool {
	const mostNumbers = math.MaxInt / 4
	return width <= math.MaxInt32 && height <= math.MaxInt32 && per <= math.MaxInt32 &&
		width <= mostNumbers/per && height <= mostNumbers/(width*per)
}
