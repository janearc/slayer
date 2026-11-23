// This file assembles a metre-scale shore and its optical water surface.

package arid

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Water describes a sampled sea around the cove's local mean water plane.
type Water struct {
	Name          string
	Sea           oceantex.Sea
	Extinction    maths.Light // RGB extinction per metre
	Extent        float64
	Collar        float64
	Samples       int
	Seed          uint64
	BoundedEdge   bool // limit the continuation's boundary-cell extrapolation
	Composition   *materials.WaterComposition
	OpticalSource string
}

// Cove keeps the shoreline and water recipe while a caller stages shots.
type Cove struct {
	world *thing.World
	shore []*thing.Thing
	water Water
	look  *thing.Material
}

// New accepts positioned shore geometry such as terrain, seabed or piers.
// A nil water recipe leaves the relief exposed for a dry study.
func New(shore []thing.Thing, water *Water) (*Cove, error) {
	made := &Cove{world: thing.NewWorld()}
	for _, feature := range shore {
		made.shore = append(made.shore, &feature)
	}
	if water != nil {
		medium, err := thing.NewClearMedium(water.Name, water.Extinction, 1)
		if err != nil {
			return nil, err
		}
		if water.Composition != nil {
			medium, err = medium.WithWaterComposition(water.Composition, water.OpticalSource)
			if err != nil {
				return nil, err
			}
		}
		made.water = *water
		made.look = &thing.Material{
			Name: water.Name, Through: 1, Gloss: 1, Index: 1.333, ClearMedium: medium,
			Source: water.OpticalSource, WaterComposition: water.Composition,
		}
	}
	return made, nil
}

// World is the render world. Lights, sky and haze belong to the shot.
func (made *Cove) World() *thing.World { return made.world }

// Advance stages the shore and water at one time. Occupants can place
// themselves using the same height field as the visible water. They are
// optional and receive nil water in a dry study.
func (made *Cove) Advance(at time.Duration, occupants func(*thing.World, *shape.HeightField) error) error {
	next := thing.NewWorld()
	var field *shape.HeightField
	if made.look != nil {
		water := made.water
		var err error
		seconds := max(0, at.Seconds())
		field, err = NewHeightField(HeightFieldConfig{
			Sea: water.Sea, Extent: water.Extent, Samples: water.Samples,
			Seed: water.Seed, Seconds: seconds,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", water.Name, err)
		}
		var surface *InfiniteMeanContinuation
		if water.BoundedEdge {
			surface, err = NewBoundedInfiniteMeanContinuation(field, 0, water.Collar)
		} else {
			surface, err = NewInfiniteMeanContinuation(field, 0, water.Collar)
		}
		if err != nil {
			return err
		}
		surface, err = surface.WithApproximateSmoothNormals()
		if err != nil {
			return err
		}
		boundary := &thing.Thing{Shape: surface, Material: made.look}
		next.OceanSurface = &thing.OceanSurface{
			Boundary: boundary, ExtentMetres: water.Extent, MetresPerUnit: 1,
			InfiniteMeanPlane: true, InterfaceEvents: 4, SunLampIndex: -1, FillLampIndex: -1,
		}
	}
	if occupants != nil {
		if err := occupants(next, field); err != nil {
			return err
		}
	}
	if next.OceanSurface != nil {
		next.Things = append(next.Things, next.OceanSurface.Boundary)
	}
	made.world.Things = append(next.Things, made.shore...)
	made.world.OceanSurface = next.OceanSurface
	return nil
}
