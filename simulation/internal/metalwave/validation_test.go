package metalwave

import (
	"math"
	"testing"

	"github.com/janearc/slayer/simulation/internal/metalfft"
	"pgregory.net/rapid"
)

func validationShape() metalfft.Shape { return metalfft.Shape{Width: 2, Height: 2, MaxCells: 4} }
func validationFields() Fields {
	return Fields{Value: []complex128{1 + 2i, -3 + 4i, 1, -1i}, Rate: []complex128{2, -1i, 0, -3 + 4i}}
}

// Both component sums are independent and include real and imaginary parts.
func TestSpectralRangeCountsBothComponents(t *testing.T) {
	cells, bounds, err := checkedSpectra(validationShape(), validationFields())
	if err != nil || cells != 4 || bounds.value != 12 || bounds.rate != 10 {
		t.Fatalf("checked spectra: %d, %+v, %v", cells, bounds, err)
	}
}

// anti-test of TestSpectralRangeCountsBothComponents
// Invalid shapes, component lengths and finite-but-unrepresentable sums
// must be refused before opening a device or packing data.
func TestSpectralRangeRejectsInvalidOrUnrepresentableFields(t *testing.T) {
	for _, which := range []string{"shape", "budget", "short value", "short rate", "nan value", "infinite rate", "value range", "rate range", "sum overflow"} {
		t.Run(which, func(t *testing.T) {
			shape, fields := validationShape(), validationFields()
			switch which {
			case "shape":
				shape.Width = 3
			case "budget":
				shape.MaxCells = 1
			case "short value":
				fields.Value = fields.Value[:3]
			case "short rate":
				fields.Rate = fields.Rate[:3]
			case "nan value":
				fields.Value[0] = complex(math.NaN(), 0)
			case "infinite rate":
				fields.Rate[0] = complex(0, math.Inf(1))
			case "value range":
				fields.Value[0] = math.MaxFloat32
			case "rate range":
				fields.Rate[0] = math.MaxFloat32
			case "sum overflow":
				for i := range fields.Value {
					fields.Value[i] = complex(math.MaxFloat64, math.MaxFloat64)
				}
			}
			_, _, err := checkedSpectra(shape, fields)
			if err == nil {
				t.Fatal("invalid spectral input accepted")
			}
		})
	}
}

// Valid signed coefficients can advance either time direction.
func TestModeRangeAcceptsBoundedSignedMatrices(t *testing.T) {
	modes := []Mode{{1, 0, 0}, {0, 0.5, -2}, {0.75, -0.5, 2}, {1, -2, 0}}
	if err := checkedModes(modes, 4, amplitudes{12, 10}); err != nil {
		t.Fatal(err)
	}
}

// anti-test of TestModeRangeAcceptsBoundedSignedMatrices
// Bad coefficients and each independent output-sum overflow are refused.
func TestModeRangeRejectsInvalidAndOverflowingMatrices(t *testing.T) {
	for _, which := range []string{"count", "nan cosine", "infinite drift", "large acceleration", "subnormal", "value product", "rate product"} {
		t.Run(which, func(t *testing.T) {
			modes := []Mode{{1, 0, 0}, {1, 0, 0}, {1, 0, 0}, {1, 0, 0}}
			bounds := amplitudes{12, 10}
			switch which {
			case "count":
				modes = modes[:3]
			case "nan cosine":
				modes[0].Cosine = math.NaN()
			case "infinite drift":
				modes[0].Drift = math.Inf(-1)
			case "large acceleration":
				modes[0].Acceleration = math.MaxFloat64
			case "subnormal":
				modes[0].Drift = 1e-40
			case "value product":
				bounds = amplitudes{math.MaxFloat32 / 16, math.MaxFloat32 / 16}
				modes[0].Drift = 2
			case "rate product":
				bounds = amplitudes{math.MaxFloat32 / 16, math.MaxFloat32 / 16}
				modes[0].Acceleration = 2
			}
			if err := checkedModes(modes, 4, bounds); err == nil {
				t.Fatal("invalid mode matrix accepted")
			}
		})
	}
}

// Generated bounded inputs leave ample float32 range for both outputs.
func TestModeRangeAcceptsGeneratedSmallCoefficients(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mode := Mode{Cosine: rapid.Float64Range(-1, 1).Draw(t, "cosine"), Drift: rapid.Float64Range(-2, 2).Draw(t, "drift"), Acceleration: rapid.Float64Range(-8, 8).Draw(t, "acceleration")}
		// Avoid asking this positive property to admit a subnormal coefficient.
		for _, coefficient := range []*float64{&mode.Cosine, &mode.Drift, &mode.Acceleration} {
			if math.Abs(*coefficient) < 1e-10 {
				*coefficient = 0
			}
		}
		if err := checkedModes([]Mode{mode}, 1, amplitudes{12, 10}); err != nil {
			t.Fatal(err)
		}
	})
}

// anti-test of TestModeRangeAcceptsGeneratedSmallCoefficients
// One invalid matrix cannot hide behind otherwise valid generated entries.
func TestModeRangeRejectsAnInvalidLaterMatrix(t *testing.T) {
	modes := []Mode{{1, 0, 0}, {1, 0, 0}, {1, 0, 0}, {Cosine: 1, Acceleration: math.NaN()}}
	if err := checkedModes(modes, 4, amplitudes{12, 10}); err == nil {
		t.Fatal("invalid later coefficient was ignored")
	}
}

// anti-test of TestModeRangeAdmitsSeparatedCoefficientPeaksWithinBudget
// Different modes may attain the cosine and drift peaks. Bounds must
// combine those independent maxima before limiting the total spectrum.
func TestModeRangeCombinesIndependentCoefficientPeaks(t *testing.T) {
	modes := []Mode{{1, 0, 0}, {0, 2 / math.Pi, -math.Pi / 2}, {1, 0, 0}, {1, 0, 0}}
	if err := checkedModes(modes, 4, amplitudes{math.MaxFloat32 / 16, math.MaxFloat32 / 9}); err == nil {
		t.Fatal("independent coefficient peaks escaped the combined spectral sum bound")
	}
}

// anti-test of TestModeRangeCombinesIndependentCoefficientPeaks
// Separate peaks remain valid when both combined output bounds fit.
func TestModeRangeAdmitsSeparatedCoefficientPeaksWithinBudget(t *testing.T) {
	modes := []Mode{{1, 0, 0}, {0, 2 / math.Pi, -math.Pi / 2}, {1, 0, 0}, {1, 0, 0}}
	if err := checkedModes(modes, 4, amplitudes{math.MaxFloat32 / 64, math.MaxFloat32 / 64}); err != nil {
		t.Fatal(err)
	}
}
