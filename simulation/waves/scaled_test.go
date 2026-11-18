package waves

import (
	"math"
	"math/cmplx"
	"testing"
)

// TestScaledKeepsWhatItsStepsWouldLose checks scaled against products
// worked by hand in powers of ten, where multiplying the small factors
// first gives nought, and against the plain product where that is safe.
func TestScaledKeepsWhatItsStepsWouldLose(t *testing.T) {
	for _, c := range []struct {
		name    string
		value   complex128
		divisor float64
		factors []float64
		want    complex128
	}{
		{"two tiny factors and a large value", complex(1e306, -2e306), 1,
			[]float64{-1e-200, 1e-200}, complex(-1e-94, 2e-94)},
		{"a tiny quotient and a large value", complex(1e306, 0), 1e308,
			[]float64{1.25e-16}, complex(1.25e-18, 0)},
		{"three factors that pass through nought", complex(0, 1e290), 1,
			[]float64{-1e-200, 1e-200, 1e-150}, complex(0, -1e-260)},
		{"ordinary numbers", complex(3, -4), 4, []float64{2, 6}, complex(9, -12)},
		{"no factors and a divisor of one", complex(3, -4), 1, nil, complex(3, -4)},
	} {
		got := scaled(c.value, c.divisor, c.factors...)
		if cmplx.Abs(got-c.want) > 4e-16*cmplx.Abs(c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	if got := scaled(complex(1e306, 1e306), 1, 0, 1e300); got != 0 {
		t.Errorf("a factor of nought gave %v", got)
	}
	if got := scaled(complex(1e306, 0), 1, 1e300); !math.IsInf(real(got), 1) {
		t.Errorf("a product too large to hold gave %v, want infinity", got)
	}
}

// TestWaveAtNoFrequencyDriftsAndDoesNotAccelerate checks the mode that
// does not oscillate: its value moves by its rate times the time, and
// its rate is the rate it began with.
func TestWaveAtNoFrequencyDriftsAndDoesNotAccelerate(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	for i := range initial.Value {
		initial.Value[i], initial.Rate[i] = complex(3, 0), complex(0.5, 0)
	}
	field := openWave(t, make([]float64, 32), initial)
	got, err := field.At(8)
	if err != nil {
		t.Fatal(err)
	}
	for i := range got.Value {
		if cmplx.Abs(got.Rate[i]-complex(0.5, 0)) > 1e-15 {
			t.Fatalf("rate %d is %v: a mode with no frequency was accelerated", i, got.Rate[i])
		}
		if want := complex(7, 0); cmplx.Abs(got.Value[i]-want) > 2e-11*cmplx.Abs(want) {
			t.Fatalf("value %d is %v, want %v", i, got.Value[i], want)
		}
	}
}
