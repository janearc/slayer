package placement

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/scene/thing"
)

// StagedWorld is a read-only working mesh of renderer-staged surfaces.
// The aggregate has no authored identity; the counts name resources not yet shown.
type StagedWorld struct {
	Mesh                   *DisplayMesh
	Volumes, Fields, Lamps int
}

// WorkingWorld translates a renderer-staged world at a scene time without drawing.
func WorkingWorld(world *thing.World, at time.Duration) (StagedWorld, error) {
	if world == nil {
		return StagedWorld{}, fmt.Errorf("staged world is unavailable")
	}
	builder := newDisplayBuilder()
	builder.heightFields = true
	for index, part := range world.Things {
		if part == nil {
			return StagedWorld{}, fmt.Errorf("staged surface %d is unavailable", index)
		}
		first := len(builder.components)
		if err := builder.thingAt(*part, at); err != nil {
			return StagedWorld{}, fmt.Errorf("staged surface %d: %w", index, err)
		}
		if author, has := world.AuthorOf(part); has {
			for i := first; i < len(builder.components); i++ {
				builder.components[i].author, builder.components[i].authored = author, true
			}
		}
	}
	mesh, err := builder.finish()
	if err != nil {
		return StagedWorld{}, err
	}
	return StagedWorld{Mesh: mesh, Volumes: len(world.Volumes), Fields: len(world.Fields), Lamps: len(world.Lamps)}, nil
}
