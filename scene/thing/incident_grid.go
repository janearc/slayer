package thing

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// IncidentGrid is a bounded world-space lattice of incident sky radiance means.
// It caches illumination, never density or opacity. The renderer recomputes it
// each frame; outside its domain, transport uses direct angular integration.
type IncidentGrid struct {
	bounds maths.Box
	nodes  [3]int
}

// NewIncidentGrid checks the complete allocation and its geometric domain.
// The 262144-node cap bounds this optional RGB payload to three MiB; it is a
// work limit, not an accuracy guarantee. Coordinates are in world metres.
func NewIncidentGrid(bounds maths.Box, nodes [3]int) (*IncidentGrid, error) {
	count := 1
	for axis, low := range []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z} {
		high := []float64{bounds.Max.X, bounds.Max.Y, bounds.Max.Z}[axis]
		if math.IsNaN(low) || math.IsNaN(high) || math.IsInf(low, 0) || math.IsInf(high, 0) || high <= low || math.IsInf(high-low, 0) ||
			nodes[axis] < 2 || nodes[axis] > 128 || count > 262144/nodes[axis] {
			return nil, fmt.Errorf("thing: invalid incident grid domain or node budget")
		}
		count *= nodes[axis]
	}
	return &IncidentGrid{bounds: bounds, nodes: nodes}, nil
}

// Parameters returns independent values, preserving checked grid ownership.
func (grid *IncidentGrid) Parameters() (maths.Box, [3]int) { return grid.bounds, grid.nodes }

// WithGrid requests native sky-source interpolation for this lighting model.
// CPU rendering remains a direct reference. The grid has no temporal history.
func (light VolumeIllumination) WithGrid(grid *IncidentGrid) (VolumeIllumination, error) {
	if light.samples == 0 || grid == nil || grid.nodes[0] < 2 {
		return VolumeIllumination{}, fmt.Errorf("thing: an incident grid needs checked isotropic sky lighting")
	}
	light.grid = grid
	return light, nil
}

// Grid returns immutable grid ownership, or nil for direct integration.
func (light VolumeIllumination) Grid() *IncidentGrid { return light.grid }
