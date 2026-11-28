package placement

import (
	"fmt"
	"reflect"
	"time"

	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// StagedBatch places one shared mesh for one authored group.
type StagedBatch struct {
	Mesh       *DisplayMesh
	Places     []maths.Subspace
	Bounds     maths.Box
	HasBounds  bool
	Author     thing.AuthorRef
	Authored   bool
	pickBlocks []maths.Box
}

// StagedInstances holds staged geometry without expanding repeated meshes.
type StagedInstances struct {
	Batches                []StagedBatch
	Volumes, Fields, Lamps int
	GiantAtmospheres       int
	GlobeOutlines          []DisplaySphereOutline
	GlobeOutlinesOmitted   int
	OmittedSeas            int
	OmittedPlanes          int
	PlaneProxies           int // Number of read-only plane patches at the end of Batches.
	UnsupportedSurfaces    map[string]int
	MaterialLimits         MaterialLimits
	Sky, Haze, Glow        bool
}

type stagedMeshKey struct {
	shape  shape.Shape
	colour maths.Light
}

type stagedBatchKey struct {
	mesh     *DisplayMesh
	author   thing.AuthorRef
	authored bool
}

// WorkingInstances keeps shared mesh members and their actual transforms.
// It is a read-only staging representation, not a claim of editable picking.
func WorkingInstances(world *thing.World, at time.Duration) (StagedInstances, error) {
	return WorkingInstancesFrom(world, at, maths.Vec{})
}

// WorkingInstancesFrom uses the eye to frame infinite plane patches and
// to frame an open sea when no finite geometry supplies a preview extent.
func WorkingInstancesFrom(world *thing.World, at time.Duration, eye maths.Vec) (StagedInstances, error) {
	if world == nil {
		return StagedInstances{}, fmt.Errorf("staged world is unavailable")
	}
	result := StagedInstances{
		Volumes: len(world.Volumes), Fields: len(world.Fields), Lamps: len(world.Lamps),
		GiantAtmospheres: WorkingGiantAtmospheres(world),
		Sky:              world.Sky != nil, Haze: world.Haze.Density != 0, Glow: world.Glow != nil,
	}
	cache := make(map[stagedMeshKey]*DisplayMesh)
	batches := make(map[stagedBatchKey]int)
	var seas []stagedSea
	var planes []StagedBatch
	var add func(shape.Shape, maths.Subspace, *thing.Material, thing.AuthorRef, bool) error
	add = func(solid shape.Shape, place maths.Subspace, material *thing.Material, author thing.AuthorRef, authored bool) error {
		if solid == nil || (reflect.ValueOf(solid).Kind() == reflect.Ptr && reflect.ValueOf(solid).IsNil()) || !shape.ValidPlacement(place) {
			return fmt.Errorf("staged instance has invalid geometry or placement")
		}
		colour := maths.Light{}
		if material != nil {
			colour = material.Colour
			if colour == (maths.Light{}) {
				colour = material.Glow
			}
		}
		switch surfaceCoverage[reflect.TypeOf(solid)] {
		case finitePlaneProxy:
			batch, shown, err := planeAroundEye(place, eye, colour, material)
			if err != nil {
				return err
			}
			if shown {
				planes = append(planes, batch)
			} else {
				result.OmittedPlanes++
			}
			return nil
		case unclassifiedSurface, omittedSurface:
			if ball, ok := solid.(globe.Globe); ok {
				scale := place.Scale
				if scale == 0 {
					scale = 1
				}
				outline := DisplaySphereOutline{Centre: place.Origin, Radius: ball.Radius() * scale}
				if outline.Valid() {
					if len(result.GlobeOutlines) < GlobeOutlineLimit {
						result.GlobeOutlines = append(result.GlobeOutlines, outline)
					} else {
						result.GlobeOutlinesOmitted++
					}
				}
			}
			if result.UnsupportedSurfaces == nil {
				result.UnsupportedSurfaces = make(map[string]int)
			}
			result.UnsupportedSurfaces[reflect.TypeOf(solid).String()]++
			return nil
		case meanSeaProxy:
			seas = append(seas, stagedSea{At: place, Colour: colour, Author: author, Authored: authored})
			return nil
		case groupSurface:
			group := solid.(*shape.MeshGroup)
			for i := 0; i < group.MemberCount(); i++ {
				member := group.Member(i)
				if err := add(member.Mesh, member.Space.Within(place), material, author, authored); err != nil {
					return err
				}
			}
			return nil
		}
		result.MaterialLimits |= materialLimits(material)
		key := stagedMeshKey{shape: solid, colour: colour}
		comparable := reflect.TypeOf(solid).Comparable()
		var display *DisplayMesh
		if comparable {
			display = cache[key]
		}
		if display == nil {
			builder := newStagedDisplayBuilder(solid)
			builder.heightFields = true
			if err := builder.surface(solid, maths.Local(), colour); err != nil {
				return err
			}
			var err error
			display, err = builder.finish()
			if err != nil {
				return err
			}
			if comparable {
				cache[key] = display
			}
		}
		keyForBatch := stagedBatchKey{mesh: display, author: author, authored: authored}
		index, found := batches[keyForBatch]
		if !found {
			index = len(result.Batches)
			batches[keyForBatch] = index
			result.Batches = append(result.Batches, StagedBatch{Mesh: display, Author: author, Authored: authored})
		}
		result.Batches[index].Places = append(result.Batches[index].Places, place)
		return nil
	}
	for index, part := range world.Things {
		if part == nil {
			return StagedInstances{}, fmt.Errorf("staged surface %d is unavailable", index)
		}
		if part.Hidden() {
			continue
		}
		author, authored := world.AuthorOf(part)
		if err := add(part.Shape, part.At(at), part.Material, author, authored); err != nil {
			return StagedInstances{}, fmt.Errorf("staged surface %d: %w", index, err)
		}
	}
	low, high, ok := stagedBounds(result.Batches)
	for _, sea := range seas {
		var batch StagedBatch
		var err error
		if ok {
			batch, err = finiteSea(sea, low, high)
		} else {
			batch, err = seaAroundEye(sea, eye)
		}
		if err != nil {
			return StagedInstances{}, err
		}
		result.Batches = append(result.Batches, batch)
		stagedBounds(result.Batches[len(result.Batches)-1:])
	}
	stagedBounds(planes)
	result.PlaneProxies = len(planes)
	result.Batches = append(result.Batches, planes...)
	return result, nil
}
