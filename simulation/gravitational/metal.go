package gravitational

import (
	"fmt"
	"github.com/janearc/slayer/simulation/waves"
)

// MetalField owns one paired plan for both physical strain polarizations.
// It has one caller at a time and must be closed when finished.
type MetalField struct {
	field *Field
	plan  *waves.MetalField
}

// OpenMetal prepares both polarizations and rates in one retained plan.
func (field *Field) OpenMetal() (*MetalField, error) {
	if field == nil || field.wave == nil {
		return nil, fmt.Errorf("gravitational: no initialized field")
	}
	plan, err := field.wave.OpenMetal()
	if err != nil {
		return nil, err
	}
	return &MetalField{field: field, plan: plan}, nil
}

// At evaluates the physical profiles using the paired device propagator.
// A physical regime refusal preserves the immutable plan for later queries.
func (field *MetalField) At(seconds float64) (Profiles, error) {
	if field == nil || field.field == nil || field.plan == nil {
		return Profiles{}, fmt.Errorf("gravitational: no live Metal field")
	}
	state, err := field.plan.At(seconds)
	if err != nil {
		return Profiles{}, err
	}
	return field.field.unpack(state)
}

// Close releases device resources and is idempotent, including on nil plans.
func (field *MetalField) Close() {
	if field != nil {
		field.plan.Close()
	}
}
