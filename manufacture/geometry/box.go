package geometry

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// GrowBox expands a box to include each point.
func GrowBox(box maths.Box, points []maths.Vec) maths.Box {
	for _, p := range points {
		box.Min = maths.V(math.Min(box.Min.X, p.X), math.Min(box.Min.Y, p.Y), math.Min(box.Min.Z, p.Z))
		box.Max = maths.V(math.Max(box.Max.X, p.X), math.Max(box.Max.Y, p.Y), math.Max(box.Max.Z, p.Z))
	}
	return box
}
