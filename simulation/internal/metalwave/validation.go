// Package metalwave composes two retained Cauchy spectra and their inverses.
package metalwave

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/simulation/internal/metalfft"
)

// Fields holds two row-major complex Cauchy components.
type Fields struct{ Value, Rate []complex128 }

// Mode holds real coefficients of one oscillator evolution matrix.
type Mode struct{ Cosine, Drift, Acceleration float64 }

// amplitudes bounds unnormalized sums before any native memory is touched.
type amplitudes struct{ value, rate float64 }

// sumLimit leaves headroom for float32 products, additions and inverse sums.
const sumLimit = math.MaxFloat32 / 8

// minimumNormal excludes coefficients a device may flush to zero.
const minimumNormal = 0x1p-126

// checkedSpectra validates the device grid and both initial field ranges.
func checkedSpectra(shape metalfft.Shape, initial Fields) (int, amplitudes, error) {
	cells, err := shape.Cells()
	if err != nil {
		return 0, amplitudes{}, err
	}
	if cells > int(^uint(0)>>1)/16 {
		return 0, amplitudes{}, fmt.Errorf("metalwave: coefficient buffer exceeds int byte range")
	}
	if len(initial.Value) != cells || len(initial.Rate) != cells {
		return 0, amplitudes{}, fmt.Errorf("metalwave: Cauchy component lengths must match the grid")
	}
	q, err := componentSum(initial.Value)
	if err != nil {
		return 0, amplitudes{}, err
	}
	v, err := componentSum(initial.Rate)
	if err != nil {
		return 0, amplitudes{}, err
	}
	return cells, amplitudes{q, v}, nil
}

// componentSum checks both parts and bounds every unnormalized partial sum.
func componentSum(values []complex128) (float64, error) {
	var sum float64
	for i, value := range values {
		if !finite(real(value)) || !finite(imag(value)) {
			return 0, fmt.Errorf("metalwave: non-finite component at %d", i)
		}
		sum += math.Abs(real(value)) + math.Abs(imag(value))
		if sum > sumLimit {
			return 0, fmt.Errorf("metalwave: spectrum exceeds conservative float32 sum range")
		}
	}
	return sum, nil
}

// checkedModes validates every coefficient and both combined output bounds.
// The nonnegative finite amplitudes come from checkedSpectra.
func checkedModes(modes []Mode, cells int, bounds amplitudes) error {
	if cells <= 0 || len(modes) != cells {
		return fmt.Errorf("metalwave: mode coefficient count must match the grid")
	}
	var maxima [3]float64
	for i, mode := range modes {
		for component, coefficient := range [3]float64{mode.Cosine, mode.Drift, mode.Acceleration} {
			if err := coefficientError(coefficient); err != nil {
				return fmt.Errorf("metalwave: mode %d: %w", i, err)
			}
			maxima[component] = math.Max(maxima[component], math.Abs(coefficient))
		}
	}
	valueBound := bounds.value*maxima[0] + bounds.rate*maxima[1]
	rateBound := bounds.value*maxima[2] + bounds.rate*maxima[0]
	if valueBound > sumLimit || rateBound > sumLimit {
		return fmt.Errorf("metalwave: combined spectral products exceed conservative float32 range")
	}
	return nil
}

// coefficientError refuses non-finite, overflowing or nonzero subnormal
// factors before packing. Zero remains a valid physical coefficient.
func coefficientError(value float64) error {
	absolute := math.Abs(value)
	if !finite(value) || absolute > math.MaxFloat32 || (absolute > 0 && absolute < minimumNormal) {
		return fmt.Errorf("coefficient is outside the checked float32 range")
	}
	return nil
}

// finite excludes both infinities and NaN.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
