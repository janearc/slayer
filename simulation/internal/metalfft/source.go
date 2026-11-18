// Package metalfft appends Fourier dispatches to a caller-owned Metal
// session, without committing or reading their results back.
package metalfft

import (
	_ "embed"
	"fmt"
)

// source is the kernels shared by standalone transforms and fluid steps.
//
//go:embed fft.metal
var source string

// Shape is an encoder's fixed grid and explicit maximum cell budget.
type Shape struct {
	Width, Height, MaxCells int
}

// Source is the shader source to combine with a caller's other kernels.
func Source() string { return source }

// Kernels is a fresh list of the functions the source provides.
func Kernels() []string {
	return []string{permutationKernel, rowKernel, columnKernel, scaleKernel}
}

// The kernel names are private to this source when a caller combines it
// with its own kernels in one library.
const (
	permutationKernel = "spectral_fft_permute"
	rowKernel         = "spectral_fft_row"
	columnKernel      = "spectral_fft_column"
	scaleKernel       = "spectral_fft_scale"
)

// These bounds follow the bridge's signed 32-bit dispatch dimensions,
// the shader's unsigned 32-bit cell indices, and float2's two components.
const (
	maximumDimension = 1<<31 - 1
	maximumCells     = 1<<32 - 1
	complexParts     = 2
	floatBytes       = 4
)

// Cells validates the caller's budget and every integer representation
// used by the bridge, buffers and shaders, before allocating anything.
func (shape Shape) Cells() (int, error) {
	for _, dimension := range []int{shape.Width, shape.Height} {
		if dimension < 2 || dimension&(dimension-1) != 0 {
			return 0, fmt.Errorf("metalfft: dimensions must be "+
				"powers "+
				"of two of at least 2: %d by %d", shape.Width,
				shape.Height)
		}
	}
	if shape.MaxCells <= 0 {
		return 0, fmt.Errorf("metalfft: cell budget must be positive")
	}
	maximumInt := int(^uint(0) >> 1)
	if shape.Width > maximumInt/shape.Height {
		return 0, fmt.Errorf("metalfft: cell count overflows int")
	}
	cells := shape.Width * shape.Height
	if cells > shape.MaxCells {
		return 0, fmt.Errorf("metalfft: %d cells exceed budget %d",
			cells, shape.MaxCells)
	}
	if shape.Width > maximumDimension || shape.Height > maximumDimension ||
		uint64(cells) > maximumCells ||
		cells > maximumInt/floatBytes/complexParts {
		return 0, fmt.Errorf("metalfft: shape exceeds device index " +
			"or buffer limits")
	}
	return cells, nil
}
