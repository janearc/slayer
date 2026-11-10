package maths

import (
	"fmt"
	"math"
)

// View constructs a finite camera placement facing a distinct target.
// Up must be measurable; LookAtUp defines the vertical-view fallback.
func View(from, target, up Vec) (*Subspace, error) {
	for _, vector := range []Vec{from, target, up, from.Sub(target)} {
		length := vector.Len()
		if math.IsNaN(length) || math.IsInf(length, nought) {
			return nil, fmt.Errorf("view: non-finite position " +
				"or direction")
		}
	}
	if unmeasured(up) || unmeasured(from.Sub(target)) {
		return nil, fmt.Errorf("view: needs an up " +
			"and a distinct target")
	}
	turn := LookAtUp(from, target, up)
	return &Subspace{Origin: from, Orientation: turn}, nil
}
