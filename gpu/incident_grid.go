package gpu

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// incidentGrid owns one reserved RGB lattice in the ordinary volume payload.
// Samples are vertices, including the boundary, rather than clamped cell means.
type incidentGrid struct {
	low, high maths.Vec
	nodes     [3]int
	first     int
}

// reserveIncidentGrid validates float node separation before allocating or
// compiling. Small domains far from the render origin must not collapse into
// duplicate float32 positions and claim resolution they do not have.
func (encoded *scene) reserveIncidentGrid(light thing.VolumeIllumination) error {
	grid := light.Grid()
	if grid == nil {
		return nil
	}
	bounds, nodes := grid.Parameters()
	for axis, low := range []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z} {
		high := []float64{bounds.Max.X, bounds.Max.Y, bounds.Max.Z}[axis]
		if !fits(low) || !fits(high) {
			return fmt.Errorf("gpu: incident grid exceeds finite float32 coordinates")
		}
		a, b := float32(low), float32(high)
		if !fits(float64(b-a)) || b <= a {
			return fmt.Errorf("gpu: incident grid loses its float32 extent")
		}
		previous := a
		for index := 1; index < nodes[axis]; index++ {
			value := a + float32(index)/float32(nodes[axis]-1)*(b-a)
			if value <= previous {
				return fmt.Errorf("gpu: incident grid nodes collapse in float32")
			}
			previous = value
		}
	}
	bytes := nodes[0] * nodes[1] * nodes[2] * 3 * 4
	if len(encoded.boxes.bytes) > math.MaxInt32-bytes {
		return fmt.Errorf("gpu: incident grid exceeds volume buffer byte limit")
	}
	encoded.incident = &incidentGrid{low: bounds.Min, high: bounds.Max, nodes: nodes, first: len(encoded.boxes.bytes) / 4}
	encoded.boxes.bytes = append(encoded.boxes.bytes, make([]byte, bytes)...)
	return nil
}

// incidentHeader emits a disabled range too, keeping every named ABI field
// initialized for trace, smoke preparation and the environment preparation pass.
func (encoded *scene) incidentHeader(given values) {
	given["sky_low"], given["sky_high"], given["sky_grid"] = nil, nil, wholes(0, 0, 0, -1)
	if grid := encoded.incident; grid != nil {
		given["sky_low"], given["sky_high"] = point(grid.low), point(grid.high)
		given["sky_grid"] = wholes(grid.nodes[0], grid.nodes[1], grid.nodes[2], grid.first)
	}
}
