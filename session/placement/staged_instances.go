package placement

import (
	"fmt"
	"reflect"
	"time"

	"github.com/janearc/slayer/arid"
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
	OmittedSeas            int
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
	if world == nil {
		return StagedInstances{}, fmt.Errorf("staged world is unavailable")
	}
	result := StagedInstances{Volumes: len(world.Volumes), Fields: len(world.Fields), Lamps: len(world.Lamps)}
	cache := make(map[stagedMeshKey]*DisplayMesh)
	batches := make(map[stagedBatchKey]int)
	var seas []stagedSea
	var add func(shape.Shape, maths.Subspace, maths.Light, thing.AuthorRef, bool) error
	add = func(solid shape.Shape, place maths.Subspace, colour maths.Light, author thing.AuthorRef, authored bool) error {
		if solid == nil || (reflect.ValueOf(solid).Kind() == reflect.Ptr && reflect.ValueOf(solid).IsNil()) || !shape.ValidPlacement(place) {
			return fmt.Errorf("staged instance has invalid geometry or placement")
		}
		if group, ok := solid.(*shape.MeshGroup); ok {
			for _, member := range group.Members() {
				if err := add(member.Mesh, member.Space.Within(place), colour, author, authored); err != nil {
					return err
				}
			}
			return nil
		}
		key := stagedMeshKey{shape: solid, colour: colour}
		comparable := reflect.TypeOf(solid).Comparable()
		var display *DisplayMesh
		if comparable {
			display = cache[key]
		}
		if display == nil {
			builder := newDisplayBuilder()
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
		colour := maths.Light{}
		if part.Material != nil {
			colour = part.Material.Colour
			if colour == (maths.Light{}) {
				colour = part.Material.Glow
			}
		}
		author, authored := world.AuthorOf(part)
		if sea, isSea := part.Shape.(*arid.Sea); isSea {
			if sea == nil || !shape.ValidPlacement(part.At(at)) {
				return StagedInstances{}, fmt.Errorf("staged sea %d is unavailable", index)
			}
			seas = append(seas, stagedSea{At: part.At(at), Colour: colour, Author: author, Authored: authored})
			continue
		}
		if err := add(part.Shape, part.At(at), colour, author, authored); err != nil {
			return StagedInstances{}, fmt.Errorf("staged surface %d: %w", index, err)
		}
	}
	if len(result.Batches) == 0 {
		return StagedInstances{}, fmt.Errorf("native working display contains no surfaces")
	}
	low, high, ok := stagedBounds(result.Batches)
	if len(seas) > 0 {
		for _, sea := range seas {
			if !ok {
				result.OmittedSeas++
				continue
			}
			batch, err := finiteSea(sea, low, high)
			if err != nil {
				return StagedInstances{}, err
			}
			proxy := []StagedBatch{batch}
			stagedBounds(proxy)
			result.Batches = append(result.Batches, proxy[0])
		}
	}
	return result, nil
}
