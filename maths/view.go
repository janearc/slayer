package maths

import "fmt"

// View constructs a finite camera placement facing a distinct target.
// Up must be measurable; LookAtUp defines the vertical-view fallback.
//
// A position, a target or an up that is no number, is infinite, or is too
// long for Len to measure, a part past about 1e154, is refused with an
// error, as is the distance from the target to the position when it is.
func View(from, target, up Vec) (*Subspace, error) {
	for _, vector := range []Vec{from, target, up, from.Sub(target)} {
		if unreadable(vector) {
			return nil, fmt.Errorf("view: a position or direction that " +
				"is no number, is infinite or is too long to measure")
		}
	}
	if unmeasured(up) || unmeasured(from.Sub(target)) {
		return nil, fmt.Errorf("view: needs an up " +
			"and a distinct target")
	}
	turn := LookAtUp(from, target, up)
	return &Subspace{Origin: from, Orientation: turn}, nil
}
