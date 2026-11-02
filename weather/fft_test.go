package weather

import (
	"math"
	"math/cmplx"
	"testing"
)

// the fft of a row is the sum its definition says, term by term, and
// back again returns the row n times over.
func TestFFTIsTheDiscreteTransform(t *testing.T) {
	const n = 16
	row := make([]complex128, n)
	for i := range row {
		row[i] = complex(math.Sin(float64(3*i)+0.5), math.Cos(float64(i*i)))
	}
	want := make([]complex128, n)
	for k := range want {
		for i, x := range row {
			want[k] += x * cmplx.Exp(complex(0, -2*math.Pi*float64(k*i)/n))
		}
	}
	got := append([]complex128(nil), row...)
	fft(got, -1)
	for k := range want {
		if cmplx.Abs(got[k]-want[k]) > 1e-9 {
			t.Errorf("term %d: %v, want %v", k, got[k], want[k])
		}
	}
	fft(got, 1)
	for i := range row {
		if cmplx.Abs(got[i]/n-row[i]) > 1e-12 {
			t.Errorf("back %d: %v, want %v", i, got[i]/n, row[i])
		}
	}
}

// the upper half of a row's terms are the negative wavenumbers.
func TestWavenumbersWrapNegative(t *testing.T) {
	for k, want := range map[int]int{0: 0, 3: 3, 8: 8, 9: -7, 15: -1} {
		if got := wavenumber(k, 16); got != want {
			t.Errorf("term %d: %d, want %d", k, got, want)
		}
	}
}
