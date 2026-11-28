package placement

import "github.com/janearc/slayer/maths"

// FarClippedBatches counts complete geometry groups whose staged bounds are
// beyond the working projection's far plane. A group straddling the plane
// remains in the scene and does not count as completely clipped.
func (scene *DisplayScene) FarClippedBatches() int {
	count := 0
	for _, batch := range scene.Batches {
		if batch.HasBounds && beyondDisplayFar(batch.Bounds, maths.Local(), scene.Eye) {
			count++
		}
	}
	return count
}

// A linear camera transform attains its depth extrema at the box corners.
func beyondDisplayFar(bounds maths.Box, place, eye maths.Subspace) bool {
	for _, x := range [...]float64{bounds.Min.X, bounds.Max.X} {
		for _, y := range [...]float64{bounds.Min.Y, bounds.Max.Y} {
			for _, z := range [...]float64{bounds.Min.Z, bounds.Max.Z} {
				depth := -eye.In(place.Out(maths.V(x, y, z))).Z
				if !displayFinite(depth) || depth <= DisplayFar {
					return false
				}
			}
		}
	}
	return true
}
