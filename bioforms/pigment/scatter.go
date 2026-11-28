package pigment

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Scatter distributes soft marks through a rest-space lattice. Spacing is
// the cell width, Radius is the largest mark radius, and Density is the
// chance that a cell carries a mark. Feather is the soft share of the radius.
type Scatter struct {
	Seed                              uint64
	Spacing, Radius, Density, Feather float64
	Region                            Mask
}

type scatterMask Scatter

// NewScatter validates a reusable mark field. A nil Region covers the full
// chart. A supplied region must be safe for concurrent render samples.
func NewScatter(spec Scatter) (Mask, error) {
	radius := spec.Radius / spec.Spacing
	if !positive(spec.Spacing) || !positive(spec.Radius) ||
		!(radius > 0 && radius <= .5) || !share(spec.Density) ||
		!share(spec.Feather) || spec.Feather == 0 {
		return nil, fmt.Errorf("pigment: scatter needs positive spacing, radius up to half a cell, and valid density and feather")
	}
	return scatterMask(spec), nil
}

func (field scatterMask) Coverage(rest maths.Vec, footprint float64) float64 {
	pixel := max(0, footprint) / field.Radius
	visibility := 1 - maths.Smooth(.65, 2.5, pixel)
	if !(visibility > 0) || field.Density == 0 {
		return 0
	}
	region := 1.0
	if field.Region != nil {
		region = field.Region.Coverage(rest, footprint)
		if !(region > 0) {
			return 0
		}
	}
	point := maths.V(rest.X/field.Spacing, rest.Y/field.Spacing, rest.Z/field.Spacing)
	const gridLimit = 1 << 50
	if !finite(point) || max(math.Abs(point.X), math.Abs(point.Y), math.Abs(point.Z)) > gridLimit {
		return 0
	}
	radius := field.Radius / field.Spacing
	widen := .25 * pixel * radius
	reach := radius + widen
	strength := 0.0
	for x := math.Floor(point.X - reach); x <= math.Floor(point.X+reach); x++ {
		for y := math.Floor(point.Y - reach); y <= math.Floor(point.Y+reach); y++ {
			for z := math.Floor(point.Z - reach); z <= math.Floor(point.Z+reach); z++ {
				h := scramble(field.Seed ^ uint64(int64(x))*73856093 ^
					uint64(int64(y))*19349663 ^ uint64(int64(z))*83492791)
				if fraction(h) >= field.Density {
					continue
				}
				sample := func() float64 {
					h = scramble(h + 1)
					return fraction(h)
				}
				size := radius * (.38 + .62*sample())
				delta := maths.V(point.X-x-sample(), point.Y-y-sample(), point.Z-z-sample())
				outer := size + widen
				inner := max(0, size*(1-field.Feather)-widen)
				share := size / outer
				mark := (1 - maths.Smooth(inner, outer, delta.Len())) * share * share
				strength = max(strength, mark)
			}
		}
	}
	return min(1, region) * strength * visibility
}
