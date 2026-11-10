package oceantex

import (
	"math"
	"math/cmplx"
)

// fft is the radix-2 fast Fourier transform of [CT65], in place. The
// length must be a power of two. Forward uses exp(-i...), inverse
// exp(+i...); neither is scaled.
func fft(a []complex128, inverse bool) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	sign := -1.0
	if inverse {
		sign = 1.0
	}
	for size := 2; size <= n; size <<= 1 {
		step := cmplx.Rect(1, sign*2*math.Pi/float64(size))
		for start := 0; start < n; start += size {
			w := complex(1, 0)
			for k := 0; k < size/2; k++ {
				u := a[start+k]
				v := a[start+k+size/2] * w
				a[start+k] = u + v
				a[start+k+size/2] = u - v
				w *= step
			}
		}
	}
}

// fft2 is fft over the rows and then the columns of an n by n grid.
func fft2(a []complex128, n int, inverse bool) {
	for row := 0; row < n; row++ {
		fft(a[row*n:(row+1)*n], inverse)
	}
	col := make([]complex128, n)
	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			col[y] = a[y*n+x]
		}
		fft(col, inverse)
		for y := 0; y < n; y++ {
			a[y*n+x] = col[y]
		}
	}
}
