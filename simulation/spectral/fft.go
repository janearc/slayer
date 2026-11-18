// Package spectral transforms finite complex grids with two-dimensional
// Fourier transforms.
package spectral

import (
	"fmt"
	"math"
	"math/cmplx"
)

// Shape is the size of a row-major grid and the maximum cell budget for a
// transform.
type Shape struct {
	Width, Height, MaxCells int
}

// Forward returns the unscaled, negative-exponent two-dimensional discrete
// Fourier transform of values. It refuses invalid shapes, over-budget grids,
// wrong-sized inputs, non-finite inputs, and arithmetic that produces
// non-finite outputs without modifying values.
func Forward(shape Shape, values []complex128) ([]complex128, error) {
	return transform(shape, values, false)
}

// Inverse returns the positive-exponent two-dimensional discrete Fourier
// transform of values, normalized by the grid's cell count. It refuses
// invalid shapes, over-budget grids, wrong-sized inputs, non-finite inputs,
// and arithmetic that produces non-finite outputs without modifying values.
func Inverse(shape Shape, values []complex128) ([]complex128, error) {
	return transform(shape, values, true)
}

// transform validates and copies input before applying both separable axes.
func transform(grid Shape, values []complex128, inverse bool) ([]complex128, error) {
	cells, err := checkedShape(grid)
	if err != nil {
		return nil, err
	}
	if err := checkInput(values, cells); err != nil {
		return nil, err
	}

	result := append([]complex128(nil), values...)
	sign := -1.0
	if inverse {
		sign = 1
	}
	for row := range grid.Height {
		line := result[row*grid.Width : (row+1)*grid.Width]
		if inverse {
			scale(line, float64(grid.Width))
		}
		fft(line, sign)
		if err := checkFinite(line); err != nil {
			return nil, fmt.Errorf("row %d: %w", row, err)
		}
	}

	column := make([]complex128, grid.Height)
	for x := range grid.Width {
		for y := range grid.Height {
			column[y] = result[y*grid.Width+x]
		}
		if inverse {
			scale(column, float64(grid.Height))
		}
		fft(column, sign)
		if err := checkFinite(column); err != nil {
			return nil, fmt.Errorf("column %d: %w", x, err)
		}
		for y := range grid.Height {
			result[y*grid.Width+x] = column[y]
		}
	}
	return result, nil
}

// checkedShape validates dimensions, the caller's cell budget, and the
// multiplication used to find the row-major cell count.
func checkedShape(grid Shape) (int, error) {
	if grid.Width < 2 || !powerOfTwo(grid.Width) {
		return 0, fmt.Errorf("width %d must be a power of two of at least 2",
			grid.Width)
	}
	if grid.Height < 2 || !powerOfTwo(grid.Height) {
		return 0, fmt.Errorf("height %d must be a power of two of at least 2",
			grid.Height)
	}
	if grid.MaxCells <= 0 {
		return 0, fmt.Errorf("maximum cell budget must be positive: %d",
			grid.MaxCells)
	}
	if grid.Width > int(^uint(0)>>1)/grid.Height {
		return 0, fmt.Errorf("grid cell count overflows int: %d by %d",
			grid.Width, grid.Height)
	}
	cells := grid.Width * grid.Height
	if cells > grid.MaxCells {
		return 0, fmt.Errorf("grid has %d cells, over budget %d",
			cells, grid.MaxCells)
	}
	return cells, nil
}

// powerOfTwo accepts only positive radix-two dimensions.
func powerOfTwo(value int) bool {
	return value >= 2 && value&(value-1) == 0
}

// checkInput validates the exact row-major input and every complex component.
func checkInput(values []complex128, cells int) error {
	if len(values) != cells {
		return fmt.Errorf("got %d input values, want %d", len(values), cells)
	}
	return checkFinite(values)
}

// checkFinite refuses the first non-finite complex sample.
func checkFinite(values []complex128) error {
	for index, value := range values {
		if !finiteComplex(value) {
			return fmt.Errorf("value %d is not finite: %v", index, value)
		}
	}
	return nil
}

// finiteComplex requires both complex components to be finite.
func finiteComplex(value complex128) bool {
	return !math.IsNaN(real(value)) && !math.IsInf(real(value), 0) &&
		!math.IsNaN(imag(value)) && !math.IsInf(imag(value), 0)
}

// scale normalizes an axis before its inverse butterflies run.
func scale(values []complex128, divisor float64) {
	for index := range values {
		values[index] /= complex(divisor, 0)
	}
}

// fft is the in-place radix-two transform of one row or column. Its caller
// checks that the length is a power of two of at least two.
func fft(values []complex128, sign float64) {
	n := len(values)
	for index, reversed := 1, 0; index < n; index++ {
		bit := n >> 1
		for ; reversed&bit != 0; bit >>= 1 {
			reversed ^= bit
		}
		reversed ^= bit
		if index < reversed {
			values[index], values[reversed] = values[reversed], values[index]
		}
	}
	for size := 2; ; size <<= 1 {
		step := cmplx.Exp(complex(0, sign*2*math.Pi/float64(size)))
		for start := 0; start < n; start += size {
			turn := complex(1, 0)
			for offset := range size / 2 {
				near := values[start+offset]
				far := values[start+offset+size/2] * turn
				values[start+offset] = near + far
				values[start+offset+size/2] = near - far
				turn *= step
			}
		}
		if size == n {
			return
		}
	}
}
