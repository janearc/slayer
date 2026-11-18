package waves

import (
	"fmt"
	"slices"

	"github.com/janearc/slayer/simulation/internal/metalfft"
	"github.com/janearc/slayer/simulation/internal/metalwave"
)

// MetalField retains the two initial spectra and Metal propagation resources.
// It has one caller at a time and must be closed when finished.
type MetalField struct {
	field *Field
	plan  *metalwave.Plan
}

// OpenMetal prepares the immutable Cauchy spectra for paired propagation.
func (field *Field) OpenMetal() (*MetalField, error) {
	if field == nil || !field.initialized {
		return nil, fmt.Errorf("waves: no initialized field")
	}
	shape := field.shape
	plan, err := metalwave.Open(metalfft.Shape{Width: shape.Width, Height: shape.Height, MaxCells: shape.MaxCells}, metalwave.Fields{Value: field.spectrum.Value, Rate: field.spectrum.Rate})
	if err != nil {
		return nil, err
	}
	return &MetalField{field: field, plan: plan}, nil
}

// At returns value and rate using one committed Metal batch.
func (field *MetalField) At(seconds float64) (State, error) {
	if field == nil || field.field == nil {
		return State{}, fmt.Errorf("waves: no live Metal field")
	}
	if err := field.plan.Usable(); err != nil {
		return State{}, err
	}
	if !waveFinite(seconds) {
		return State{}, fmt.Errorf("waves: elapsed seconds must be finite")
	}
	if seconds == 0 {
		return State{Value: slices.Clone(field.field.initial.Value), Rate: slices.Clone(field.field.initial.Rate)}, nil
	}
	modes, err := field.field.metalModes(seconds)
	if err != nil {
		return State{}, err
	}
	result, err := field.plan.Evaluate(modes)
	if err != nil {
		return State{}, err
	}
	return State{Value: result.Value, Rate: result.Rate}, nil
}

// metalModes retains float64 phase evaluation before converting coefficients.
func (field *Field) metalModes(seconds float64) ([]metalwave.Mode, error) {
	modes := make([]metalwave.Mode, len(field.frequency))
	for i, omega := range field.frequency {
		cosine, drift, acceleration, err := waveModeFactors(omega, seconds)
		if err != nil {
			return nil, fmt.Errorf("waves: mode %d: %w", i, err)
		}
		modes[i] = metalwave.Mode{Cosine: cosine, Drift: drift, Acceleration: acceleration}
	}
	return modes, nil
}

// Close releases resources and is harmless on nil or closed plans.
func (field *MetalField) Close() {
	if field != nil {
		field.plan.Close()
	}
}
