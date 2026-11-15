package main

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/thing"
)

// volumeLightingPlan opts the scene into normalized single scattering.
// Omission preserves legacy lamp-only media and their authored brightness.
type volumeLightingPlan struct {
	Model   string            `yaml:"model" json:"model"`
	Samples int               `yaml:"sky-samples" json:"sky-samples"`
	Grid    *incidentGridPlan `yaml:"sky-grid" json:"sky-grid,omitempty"`
}

// incidentGridPlan fixes source nodes in the scene's anchored east/up/south
// frame, independent of the camera's floating render origin.
type incidentGridPlan struct {
	Low   [3]float64 `yaml:"low-metres" json:"low-metres"`
	High  [3]float64 `yaml:"high-metres" json:"high-metres"`
	Nodes [3]int     `yaml:"nodes" json:"nodes"`
}

// volumeIllumination checks the cross-field work rule after schema validation.
func (plan plan) volumeIllumination() (thing.VolumeIllumination, error) {
	if plan.VolumeLighting == nil {
		return thing.VolumeIllumination{}, nil
	}
	light, err := thing.NewVolumeIllumination(plan.VolumeLighting.Samples)
	if err != nil || plan.VolumeLighting.Grid == nil {
		return light, err
	}
	grid := plan.VolumeLighting.Grid
	checked, err := thing.NewIncidentGrid(maths.Box{Min: maths.V(grid.Low[0], grid.Low[1], grid.Low[2]),
		Max: maths.V(grid.High[0], grid.High[1], grid.High[2])}, grid.Nodes)
	if err != nil {
		return thing.VolumeIllumination{}, err
	}
	return light.WithGrid(checked)
}

// relativeIllumination carries a fixed source lattice into the current render
// origin without moving its physical nodes as the camera rises or rotates.
func (plan plan) relativeIllumination(origin frame.Relative) (thing.VolumeIllumination, error) {
	light, err := plan.volumeIllumination()
	if err != nil || light.Grid() == nil {
		return light, err
	}
	bounds, nodes := light.Grid().Parameters()
	low, err := origin.Point(bounds.Min)
	if err != nil {
		return thing.VolumeIllumination{}, err
	}
	high, err := origin.Point(bounds.Max)
	if err != nil {
		return thing.VolumeIllumination{}, err
	}
	grid, err := thing.NewIncidentGrid(maths.Box{Min: low, Max: high}, nodes)
	if err != nil {
		return thing.VolumeIllumination{}, err
	}
	return light.WithGrid(grid)
}
