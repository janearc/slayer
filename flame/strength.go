package flame

import (
	"math"
	"slices"
)

// WithStrengths returns an immutable state of the field with one strength
// in [0,1] per patch. Absorption follows strength and height its cube root;
// this is an authored display transfer, not a flame-height correlation.
// Bounds and spectral palettes stay shared. Zero-strength sources disappear.
func (field *Field) WithStrengths(strengths []float64) (*Field, error) {
	if len(strengths) != len(field.pieces) {
		return nil, errFlame
	}
	later := *field
	later.pieces = slices.Clone(field.pieces)
	for i, v := range strengths {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			return nil, errFlame
		}
		body := *later.pieces[i].Body
		body.fire.Absorption *= v
		body.fire.Height *= math.Cbrt(v)
		later.pieces[i].Body = &body
	}
	return &later, nil
}
