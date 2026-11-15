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
// It draws the picture in bands of at most rows rows, one command each, so no
// single command runs long enough for the system to stop it; the inputs are
// made into buffers once for all the bands. Rows of zero or less is the whole
// picture at once.
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
	for first := 0; first < height; first += rows {
		down := min(rows, height-first)
		band := out[first*width*per : (first+down)*width*per]
		if err := k.readBand(stream, first, down, band); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// gridFits checks positive dimensions before allocation and C-int narrowing.
// The caller has already handled empty dimensions and nonpositive channels.
func gridFits(width, height, per int) bool {
	const mostNumbers = math.MaxInt / 4
	return width <= math.MaxInt32 && height <= math.MaxInt32 && per <= math.MaxInt32 &&
		width <= mostNumbers/per && height <= mostNumbers/(width*per)
}
