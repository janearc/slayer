package waves

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/janearc/slayer/simulation/spectral"
)

// NewWater prepares finite-depth linear surface-gravity waves. It refuses
// a net periodic surface rate and removes only a roundoff-sized rate mean.
func NewWater(config WaterConfig, initial State) (*Field, error) {
	if err := waterUnitsError(config); err != nil {
		return nil, err
	}
	field, err := newWaveBase(config.Shape, initial)
	if err != nil {
		return nil, err
	}
	if err := field.removeWaterRateMean(); err != nil {
		return nil, err
	}
	field.frequency, err = waterFrequencies(config)
	if err != nil {
		return nil, err
	}
	field.initialized = true
	return field, nil
}

// waterUnitsError checks SI periods, depth and gravity before grid allocation.
func waterUnitsError(config WaterConfig) error {
	for _, part := range []struct {
		name  string
		value float64
	}{
		{"width", config.WidthMetres}, {"height", config.HeightMetres},
		{"depth", config.DepthMetres}, {"gravity", config.Gravity}} {
		if !waveFinite(part.value) || part.value <= 0 {
			return fmt.Errorf("waves: water %s must be finite and positive", part.name)
		}
	}
	return nil
}

// removeWaterRateMean enforces the unforced periodic mass constraint.
// The picked relative allowance covers a tiny transform cancellation mean.
func (field *Field) removeWaterRateMean() error {
	var sum float64
	for _, rate := range field.initial.Rate {
		sum += math.Abs(real(rate)) + math.Abs(imag(rate))
		if !waveFinite(sum) {
			return fmt.Errorf("waves: water rate component sum overflows")
		}
	}
	const meanRoundoff = 64 * 0x1p-52
	if cmplx.Abs(field.spectrum.Rate[0]) > meanRoundoff*sum {
		return fmt.Errorf("waves: periodic water requires zero mean surface rate")
	}
	if field.spectrum.Rate[0] == 0 {
		return nil
	}
	field.spectrum.Rate[0] = 0
	rate, err := spectral.Inverse(field.shape, field.spectrum.Rate)
	if err != nil {
		return fmt.Errorf("waves: project water rate mean: %w", err)
	}
	field.initial.Rate = rate
	return nil
}

// waterFrequencies maps each signed two-dimensional mode to its physical
// finite-depth gravity frequency, including an exactly zero mean mode.
func waterFrequencies(config WaterConfig) ([]float64, error) {
	stepX, stepY := 2*math.Pi/config.WidthMetres, 2*math.Pi/config.HeightMetres
	if !waveFinite(stepX) || !waveFinite(stepY) {
		return nil, fmt.Errorf("waves: water domain wavenumber overflows")
	}
	values := make([]float64, config.Shape.Width*config.Shape.Height)
	for y := range config.Shape.Height {
		for x := range config.Shape.Width {
			kx := stepX * float64(waterSignedMode(x, config.Shape.Width))
			ky := stepY * float64(waterSignedMode(y, config.Shape.Height))
			k := math.Hypot(kx, ky)
			if !waveFinite(k) {
				return nil, fmt.Errorf("waves: transverse water wavenumber overflows")
			}
			omega := waterFrequency(k, config.DepthMetres, config.Gravity)
			if !waveFinite(omega) {
				return nil, fmt.Errorf("waves: water frequency overflows")
			}
			values[y*config.Shape.Width+x] = omega
		}
	}
	return values, nil
}

// waterFrequency avoids the unnecessary g*k product in deep water and the
// k*depth underflow in the shallow limit. Tanh safely saturates at infinity.
func waterFrequency(k, depth, gravity float64) float64 {
	if k == 0 {
		return 0
	}
	kh := k * depth
	if kh <= 1 {
		correction := 1.0
		if kh != 0 {
			correction = math.Sqrt(math.Tanh(kh) / kh)
		}
		return k * (math.Sqrt(gravity) * math.Sqrt(depth)) * correction
	}
	return (math.Sqrt(gravity) * math.Sqrt(k)) * math.Sqrt(math.Tanh(kh))
}

// waterSignedMode gives the signed Fourier bin on one periodic axis.
func waterSignedMode(index, size int) int {
	if index > size/2 {
		return index - size
	}
	return index
}
