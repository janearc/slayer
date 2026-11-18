package optics

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/simulation/internal/metalfft"
	"github.com/janearc/slayer/simulation/internal/metalfield"
)

// MetalField retains one field and its Metal propagation resources.
// It has one caller at a time and must be closed when finished.
type MetalField struct {
	field *Field
	plan  *metalfield.Plan
}

// OpenMetal prepares the immutable spectrum for Metal propagation.
func (field *Field) OpenMetal() (*MetalField, error) {
	if field == nil || !field.initialized {
		return nil, fmt.Errorf("optics: field is nil or uninitialized")
	}
	shape := field.config.Shape
	plan, err := metalfield.Open(metalfft.Shape{Width: shape.Width, Height: shape.Height, MaxCells: shape.MaxCells}, field.spectrum)
	if err != nil {
		return nil, err
	}
	return &MetalField{field: field, plan: plan}, nil
}

// At returns the propagated scalar field using one Metal batch.
func (field *MetalField) At(distanceMetres float64) ([]complex128, error) {
	if field == nil || field.field == nil {
		return nil, fmt.Errorf("optics: no live Metal field")
	}
	if err := field.plan.Usable(); err != nil {
		return nil, err
	}
	if !fieldFinite(distanceMetres) || distanceMetres < 0 {
		return nil, fmt.Errorf("optics: distance must be finite and nonnegative")
	}
	if distanceMetres == 0 {
		return append([]complex128(nil), field.field.initial...), nil
	}
	factors := make([]complex128, len(field.field.coefficients))
	for i, coefficient := range field.field.coefficients {
		transfer, err := phaseTransfer(coefficient.phaseRate, distanceMetres)
		if err != nil {
			return nil, fmt.Errorf("optics: mode %d: %w", i, err)
		}
		factors[i] = transfer
		if coefficient.evanescent {
			exponent := coefficient.decayRate * distanceMetres
			if !fieldFinite(exponent) {
				return nil, fmt.Errorf("optics: mode %d attenuation exponent overflows", i)
			}
			factors[i] *= complex(math.Exp(-exponent), 0)
		}
	}
	return field.plan.Evaluate(factors)
}

// Close releases the plan and is idempotent, including for nil receivers.
func (field *MetalField) Close() {
	if field == nil {
		return
	}
	field.plan.Close()
}
