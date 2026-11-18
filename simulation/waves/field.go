// Package waves evolves independent Fourier oscillators for linear waves.
package waves

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/simulation/spectral"
)

// State contains row-major complex field values and their time derivatives.
type State struct {
	Value []complex128
	Rate  []complex128
}

// Field holds one immutable periodic initial state and mode frequencies.
type Field struct {
	shape       spectral.Shape
	frequency   []float64
	initial     State
	spectrum    State
	initialized bool
}

// New copies a state and nonnegative angular frequencies in radians per second.
func New(shape spectral.Shape, frequency []float64, initial State) (*Field, error) {
	field, err := newWaveBase(shape, initial)
	if err != nil {
		return nil, err
	}
	if len(frequency) != len(field.spectrum.Value) {
		return nil, fmt.Errorf("waves: frequency length %d, want %d", len(frequency), len(field.spectrum.Value))
	}
	for i, omega := range frequency {
		if !waveFinite(omega) || omega < 0 {
			return nil, fmt.Errorf("waves: frequency %d must be finite and nonnegative", i)
		}
	}
	field.frequency = slices.Clone(frequency)
	field.initialized = true
	return field, nil
}

// newWaveBase validates the explicit grid budget through the public FFT
// before creating any other grid-sized storage.
func newWaveBase(shape spectral.Shape, initial State) (*Field, error) {
	q, err := spectral.Forward(shape, initial.Value)
	if err != nil {
		return nil, fmt.Errorf("waves: initial field: %w", err)
	}
	v, err := spectral.Forward(shape, initial.Rate)
	if err != nil {
		return nil, fmt.Errorf("waves: initial rate: %w", err)
	}
	return &Field{shape: shape,
		initial:  State{Value: slices.Clone(initial.Value), Rate: slices.Clone(initial.Rate)},
		spectrum: State{Value: q, Rate: v}}, nil
}

// At evaluates the initial state at finite elapsed seconds, including past time.
func (field *Field) At(seconds float64) (State, error) {
	if field == nil || !field.initialized {
		return State{}, fmt.Errorf("waves: nil or uninitialized field")
	}
	if !waveFinite(seconds) {
		return State{}, fmt.Errorf("waves: elapsed seconds must be finite")
	}
	if seconds == 0 {
		return State{Value: slices.Clone(field.initial.Value), Rate: slices.Clone(field.initial.Rate)}, nil
	}
	spectrum, err := field.evolvedModes(seconds)
	if err != nil {
		return State{}, err
	}
	return inverseWaveState(field.shape, spectrum)
}

// evolvedModes computes both independent Cauchy components without changing
// the stored spectra. Every coefficient and result must remain finite.
//
// A mode's drift sin(wt)/w and acceleration -w sin(wt) can be too small to
// hold as numbers by themselves while their products with a large spectrum
// are ordinary numbers. Where a coefficient or the phase is below the
// smallest normal number, the whole contribution is formed at once by
// scaled, so nothing is rounded away before the spectrum is applied.
func (field *Field) evolvedModes(seconds float64) (State, error) {
	result := State{Value: make([]complex128, len(field.frequency)), Rate: make([]complex128, len(field.frequency))}
	for i, omega := range field.frequency {
		cosine, drift, acceleration, err := waveModeFactors(omega, seconds)
		if err != nil {
			return State{}, fmt.Errorf("waves: mode %d: %w", i, err)
		}
		q, v := field.spectrum.Value[i], field.spectrum.Rate[i]
		drifted, accelerated := complex(drift, 0)*v, complex(acceleration, 0)*q
		if phase := omega * seconds; math.Abs(phase) < smallestNormal {
			// The phase has lost bits or all of itself, and its sine is
			// the phase: the drift is the time and the acceleration is
			// -w w t, each factor taken as it was given.
			drifted, accelerated = complex(seconds, 0)*v, scaled(q, 1, -omega, omega, seconds)
		} else if math.Abs(drift) < smallestNormal || math.Abs(acceleration) < smallestNormal {
			sine := math.Sin(phase)
			drifted, accelerated = scaled(v, omega, sine), scaled(q, 1, -omega, sine)
		}
		result.Value[i] = complex(cosine, 0)*q + drifted
		result.Rate[i] = accelerated + complex(cosine, 0)*v
		if !waveFiniteComplex(result.Value[i]) || !waveFiniteComplex(result.Rate[i]) {
			return State{}, fmt.Errorf("waves: mode %d evolved state overflows", i)
		}
	}
	return result, nil
}

// smallestNormal is the least positive number held at full precision.
const smallestNormal = 0x1p-1022

// scaled is value times the product of the factors, over divisor, with the
// powers of two kept apart until the end. No step between can overflow or
// underflow, so the result is lost only if it cannot itself be held. A
// divisor of one divides by nothing.
func scaled(value complex128, divisor float64, factors ...float64) complex128 {
	mantissa, exponent := 1.0, 0
	for _, factor := range factors {
		part, power := math.Frexp(factor)
		mantissa, exponent = mantissa*part, exponent+power
	}
	part, power := math.Frexp(divisor)
	mantissa, exponent = mantissa/part, exponent-power
	re, rePower := math.Frexp(real(value))
	im, imPower := math.Frexp(imag(value))
	return complex(math.Ldexp(re*mantissa, rePower+exponent),
		math.Ldexp(im*mantissa, imPower+exponent))
}

// waveModeFactors uses t*sinc(omega*t), so a tiny frequency does not erase
// a finite drift when the phase itself underflows to zero.
func waveModeFactors(omega, seconds float64) (float64, float64, float64, error) {
	phase := omega * seconds
	if !waveFinite(phase) {
		return 0, 0, 0, fmt.Errorf("phase overflows")
	}
	sine, cosine := math.Sincos(phase)
	sinc := 1.0
	if phase != 0 {
		sinc = sine / phase
	}
	drift, acceleration := seconds*sinc, -omega*sine
	if !waveFinite(drift) || !waveFinite(acceleration) {
		return 0, 0, 0, fmt.Errorf("mode coefficient overflows")
	}
	return cosine, drift, acceleration, nil
}

// inverseWaveState returns neither component if either inverse fails.
func inverseWaveState(shape spectral.Shape, spectrum State) (State, error) {
	q, err := spectral.Inverse(shape, spectrum.Value)
	if err != nil {
		return State{}, fmt.Errorf("waves: inverse field: %w", err)
	}
	v, err := spectral.Inverse(shape, spectrum.Rate)
	if err != nil {
		return State{}, fmt.Errorf("waves: inverse rate: %w", err)
	}
	return State{Value: q, Rate: v}, nil
}

// waveFinite excludes both infinities and NaN.
func waveFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// waveFiniteComplex checks both components before an ordered comparison.
func waveFiniteComplex(value complex128) bool {
	return waveFinite(real(value)) && waveFinite(imag(value))
}

// WaterConfig describes a constant-depth periodic water surface in SI units.
type WaterConfig struct {
	Shape                                           spectral.Shape
	WidthMetres, HeightMetres, DepthMetres, Gravity float64
}
