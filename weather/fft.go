// Package weather models a thin atmospheric layer on a rotating sphere. Storms
// inject vorticity, friction removes energy, and the flow advects atmospheric
// bands.
//
// Jets and vortices emerge from the dynamics rather than being placed by hand,
// following the inverse cascade of two-dimensional rotating turbulence (Vallis
// and Maltrud 1993; Cho and Polvani 1996).
package weather

import (
	"math"
	"math/cmplx"
)

// fft transforms a row of complex numbers in place, its length a power
// of two: forward with a sign of minus one, back with one, unscaled.
// cooley and tukey's, by halves, the row first put in bit-reversed
// order.
func fft(row []complex128, sign float64) {
	n := len(row)
	for i, k := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; k&bit != 0; bit >>= 1 {
			k ^= bit
		}
		k ^= bit
		if i < k {
			row[i], row[k] = row[k], row[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		turn := cmplx.Exp(complex(0, sign*2*math.Pi/float64(size)))
		for start := 0; start < n; start += size {
			w := complex(1, 0)
			for k := range size / 2 {
				near, far := row[start+k], row[start+k+size/2]*w
				row[start+k], row[start+k+size/2] = near+far, near-far
				w *= turn
			}
		}
	}
}

// wavenumber is the turns round the circle of a row's k'th term, by
// sign: the upper half of a row of n are the negative ones.
func wavenumber(k, n int) int {
	if k > n/2 {
		return k - n
	}
	return k
}
