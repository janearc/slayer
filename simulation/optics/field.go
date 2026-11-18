// Package optics propagates a monochromatic scalar field by its angular
// spectrum, with the longitudinal carrier factored out.
package optics

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/simulation/spectral"
)

// Config describes a periodic field grid and its physical length and
// wavelength, all in metres.
type Config struct {
	Shape            spectral.Shape
	WidthMetres      float64
	HeightMetres     float64
	WavelengthMetres float64
}

// Field is one immutable scalar field at a single wavelength.
type Field struct {
	config       Config
	initial      []complex128
	spectrum     []complex128
	coefficients []modeCoefficient
	initialized  bool
}

type modeCoefficient struct {
	phaseRate  float64
	decayRate  float64
	evanescent bool
}

// New copies samples and prepares the field for propagation at its
// configured wavelength. Invalid units, grid, or samples return no field.
func New(config Config, samples []complex128) (*Field, error) {
	if !fieldFinite(config.WidthMetres) || config.WidthMetres <= 0 {
		return nil, fmt.Errorf("width must be finite and positive: %g",
			config.WidthMetres)
	}
	if !fieldFinite(config.HeightMetres) || config.HeightMetres <= 0 {
		return nil, fmt.Errorf("height must be finite and positive: %g",
			config.HeightMetres)
	}
	if !fieldFinite(config.WavelengthMetres) || config.WavelengthMetres <= 0 {
		return nil, fmt.Errorf("wavelength must be finite and positive: %g",
			config.WavelengthMetres)
	}

	wavenumber := twoPi / config.WavelengthMetres
	if !fieldFinite(wavenumber) || wavenumber <= 0 {
		return nil, fmt.Errorf("wavelength gives a non-finite wavenumber")
	}
	spectrum, err := spectral.Forward(config.Shape, samples)
	if err != nil {
		return nil, fmt.Errorf("transform initial field: %w", err)
	}

	widthFrequency := twoPi / config.WidthMetres
	heightFrequency := twoPi / config.HeightMetres
	if !fieldFinite(widthFrequency) || !fieldFinite(heightFrequency) {
		return nil, fmt.Errorf("grid periods give non-finite transverse wavenumbers")
	}
	coefficients := make([]modeCoefficient, len(spectrum))
	for y := range config.Shape.Height {
		ky := heightFrequency * float64(signedMode(y, config.Shape.Height))
		if !fieldFinite(ky) {
			return nil, fmt.Errorf("row %d has a non-finite transverse wavenumber", y)
		}
		for x := range config.Shape.Width {
			kx := widthFrequency * float64(signedMode(x, config.Shape.Width))
			transverse := math.Hypot(kx, ky)
			if !fieldFinite(kx) || !fieldFinite(transverse) {
				return nil, fmt.Errorf("mode (%d,%d) has a non-finite transverse wavenumber", x, y)
			}
			coefficient, err := modeCoefficientFor(transverse, wavenumber)
			if err != nil {
				return nil, fmt.Errorf("mode (%d,%d): %w", x, y, err)
			}
			coefficients[y*config.Shape.Width+x] = coefficient
		}
	}

	return &Field{
		config:  config,
		initial: append([]complex128(nil), samples...), spectrum: spectrum,
		coefficients: coefficients, initialized: true,
	}, nil
}

// At returns a new copy of the field propagated by a nonnegative distance in
// metres. A nil field, invalid distance, or non-finite result is refused.
func (field *Field) At(distanceMetres float64) ([]complex128, error) {
	if field == nil || !field.initialized {
		return nil, fmt.Errorf("field is nil or uninitialized")
	}
	if !fieldFinite(distanceMetres) || distanceMetres < 0 {
		return nil, fmt.Errorf("distance must be finite and nonnegative: %g",
			distanceMetres)
	}
	if distanceMetres == 0 {
		return append([]complex128(nil), field.initial...), nil
	}

	values := append([]complex128(nil), field.spectrum...)
	for index, coefficient := range field.coefficients {
		transfer, err := phaseTransfer(coefficient.phaseRate, distanceMetres)
		if err != nil {
			return nil, fmt.Errorf("mode %d phase: %w", index, err)
		}
		if coefficient.evanescent {
			exponent := coefficient.decayRate * distanceMetres
			if !fieldFinite(exponent) {
				return nil, fmt.Errorf("mode %d attenuation exponent overflows", index)
			}
			amplitude := math.Exp(-exponent)
			transfer *= complex(amplitude, 0)
		}
		if !fieldFiniteComplex(transfer) {
			return nil, fmt.Errorf("mode %d has a non-finite propagation coefficient", index)
		}
		values[index] *= transfer
		if !fieldFiniteComplex(values[index]) {
			return nil, fmt.Errorf("mode %d overflowed during propagation", index)
		}
	}
	result, err := spectral.Inverse(field.config.Shape, values)
	if err != nil {
		return nil, fmt.Errorf("inverse transform propagated field: %w", err)
	}
	if err := checkFieldFinite(result); err != nil {
		return nil, fmt.Errorf("propagated field: %w", err)
	}
	return result, nil
}

// Intensity returns the squared magnitude of each sample. A non-finite input
// or an intensity that overflows is refused with no result.
func Intensity(samples []complex128) ([]float64, error) {
	result := make([]float64, len(samples))
	for index, sample := range samples {
		if !fieldFiniteComplex(sample) {
			return nil, fmt.Errorf("sample %d is not finite: %v", index, sample)
		}
		realPart, imaginaryPart := real(sample), imag(sample)
		realPower := realPart * realPart
		imaginaryPower := imaginaryPart * imaginaryPart
		if !fieldFinite(realPower) || !fieldFinite(imaginaryPower) {
			return nil, fmt.Errorf("sample %d intensity overflows", index)
		}
		power := realPower + imaginaryPower
		if !fieldFinite(power) {
			return nil, fmt.Errorf("sample %d intensity overflows", index)
		}
		result[index] = power
	}
	return result, nil
}

const twoPi = 2 * math.Pi

// modeCoefficientFor separates relative phase from evanescent attenuation.
func modeCoefficientFor(transverse, wavenumber float64) (modeCoefficient, error) {
	if transverse <= wavenumber {
		ratio := transverse / wavenumber
		root := math.Sqrt((1 - ratio) * (1 + ratio))
		kz := wavenumber * root
		// (kz-k) is written as the algebraically equivalent ratio to
		// avoid subtracting nearly equal wavenumbers at small angles.
		phaseRate := -wavenumber * ratio * ratio / (1 + root)
		if !fieldFinite(kz) || !fieldFinite(phaseRate) {
			return modeCoefficient{}, fmt.Errorf("propagating mode has non-finite phase rate")
		}
		return modeCoefficient{phaseRate: phaseRate}, nil
	}
	ratio := wavenumber / transverse
	root := math.Sqrt((1 - ratio) * (1 + ratio))
	decayRate := transverse * root
	if !fieldFinite(decayRate) {
		return modeCoefficient{}, fmt.Errorf("evanescent mode has non-finite decay rate")
	}
	return modeCoefficient{
		phaseRate: -wavenumber, decayRate: decayRate, evanescent: true,
	}, nil
}

// signedMode maps a Fourier bin to its signed periodic integer mode.
func signedMode(index, size int) int {
	if index > size/2 {
		return index - size
	}
	return index
}

// phaseTransfer checks the relative phase, then lets Sincos apply accurate
// large-argument reduction before either backend converts its factors.
func phaseTransfer(rate, distance float64) (complex128, error) {
	phase := rate * distance
	if !fieldFinite(phase) {
		return 0, fmt.Errorf("phase overflows float64")
	}
	sine, cosine := math.Sincos(phase)
	return complex(cosine, sine), nil
}

// checkFieldFinite identifies any non-finite propagated sample.
func checkFieldFinite(values []complex128) error {
	for index, value := range values {
		if !fieldFiniteComplex(value) {
			return fmt.Errorf("sample %d is not finite", index)
		}
	}
	return nil
}

// fieldFinite excludes both infinities and NaN.
func fieldFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// fieldFiniteComplex requires both complex components to be finite.
func fieldFiniteComplex(value complex128) bool {
	return fieldFinite(real(value)) && fieldFinite(imag(value))
}
