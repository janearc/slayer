package thing

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// The counts an incident grid is made of.
const (
	// incidentAxes is defined: the three axes of space, x, y and z.
	incidentAxes = 3
	// incidentNone is defined: no amount.
	incidentNone = 0
	// incidentUnit is defined: one, the count of nodes before any axis is
	// taken into it.
	incidentUnit = 1
	// incidentFirst is defined: the first place in a list, the x axis.
	incidentFirst = 0
	// incidentNodesFewest is defined: two nodes, the ends of one cell,
	// which an axis needs to span an interval.
	incidentNodesFewest = 2
)

// incidentNodesMost is the most nodes an axis may have. It is read from
// picks.yaml.
var incidentNodesMost = int(pick("incident-grid-nodes-most"))

// incidentNodeBudget is the most nodes a grid may have in all. It is read
// from picks.yaml.
var incidentNodeBudget = int(pick("incident-grid-nodes-total-most"))

// IncidentGrid is a bounded world-space lattice of incident sky radiance means.
// It caches illumination, never density or opacity. The renderer recomputes it
// each frame; outside its domain, transport uses direct angular integration.
type IncidentGrid struct {
	bounds maths.Box
	nodes  [incidentAxes]int
}

// NewIncidentGrid checks the complete allocation and its geometric domain.
// The 262144-node cap bounds this optional RGB payload to three MiB; it is a
// work limit, not an accuracy guarantee. Coordinates are in world metres.
func NewIncidentGrid(bounds maths.Box,
	nodes [incidentAxes]int) (*IncidentGrid, error) {
	count := incidentUnit
	lows := []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z}
	highs := []float64{bounds.Max.X, bounds.Max.Y, bounds.Max.Z}
	for axis, low := range lows {
		high := highs[axis]
		if !incidentSpans(low, high) || nodes[axis] < incidentNodesFewest ||
			nodes[axis] > incidentNodesMost ||
			count > incidentNodeBudget/nodes[axis] {
			return nil, fmt.Errorf("thing: invalid incident grid domain or node budget")
		}
		count *= nodes[axis]
	}
	return &IncidentGrid{bounds: bounds, nodes: nodes}, nil
}

// incidentSpans reports whether a low and a high end are finite numbers
// with a finite, positive distance between them.
func incidentSpans(low, high float64) bool {
	if math.IsNaN(low) || math.IsNaN(high) {
		return false
	}
	if math.IsInf(low, incidentNone) || math.IsInf(high, incidentNone) {
		return false
	}
	return high > low && !math.IsInf(high-low, incidentNone)
}

// Parameters returns independent values, preserving checked grid ownership.
func (grid *IncidentGrid) Parameters() (maths.Box, [incidentAxes]int) {
	return grid.bounds, grid.nodes
}

// WithGrid requests native sky-source interpolation for this lighting model.
// CPU rendering remains a direct reference. The grid has no temporal history.
func (light VolumeIllumination) WithGrid(
	grid *IncidentGrid) (VolumeIllumination, error) {
	if light.samples == incidentNone || grid == nil ||
		grid.nodes[incidentFirst] < incidentNodesFewest {
		return VolumeIllumination{}, fmt.Errorf("thing: an incident grid needs checked isotropic sky lighting")
	}
	light.grid = grid
	return light, nil
}

// Grid returns immutable grid ownership, or nil for direct integration.
func (light VolumeIllumination) Grid() *IncidentGrid { return light.grid }
