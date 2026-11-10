package maths

import "math"

// SegmentBox encloses two endpoints, expanded by a nonnegative margin on
// every axis. It also covers any linear interpolation between the endpoints.
func SegmentBox(from, to Vec, margin float64) Box {
	return Box{
		Min: V(math.Min(from.X, to.X)-margin,
			math.Min(from.Y, to.Y)-margin,
			math.Min(from.Z, to.Z)-margin),
		Max: V(math.Max(from.X, to.X)+margin,
			math.Max(from.Y, to.Y)+margin,
			math.Max(from.Z, to.Z)+margin),
	}
}
