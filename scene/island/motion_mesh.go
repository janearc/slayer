package island

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/terrain"
)

// transitionVertex retains endpoint positions and shading vectors in the fixed
// world frame. Map buckets and constructed meshes are outside this payload.
type transitionVertex struct{ from, to, normalFrom, normalTo maths.Vec }

const transitionValueBytes = int64(unsafe.Sizeof(transitionVertex{}) + unsafe.Sizeof(lattice{}))

// terrainTransition owns at most two endpoint meshes and their common lattice.
// Both endpoints use the same source and frame; only sampling detail differs.
type terrainTransition struct {
	from, target Prepared
	patches      []terrain.Patch
	values       map[lattice]transitionVertex
	faces        [][3]int
	started      time.Duration
	side         int
	smooth       bool
	reported     bool
}

// transition refuses a common refinement that exceeds either declared limit.
// Endpoint interpolation uses actual stitched triangles, not new source reads.
func (motion *Motion) transition(ctx context.Context, from, target Prepared, at time.Duration) (*terrainTransition, error) {
	a, err := motion.ground.sampler(from)
	if err != nil {
		return nil, err
	}
	b, err := motion.ground.sampler(target)
	if err != nil {
		return nil, err
	}
	patches, err := commonPatches(motion.ground.root, a.leaves, b.leaves, motion.patches)
	if err != nil {
		return nil, err
	}
	made := &terrainTransition{from: from, target: target, patches: patches,
		values: make(map[lattice]transitionVertex), started: at, side: a.side,
		smooth: motion.ground.detail.NormalLongitude > 0}
	for _, patch := range patches {
		step := (patch.MaxColumn - patch.MinColumn) / (made.side - 1)
		for row := range made.side {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for column := range made.side {
				key := lattice{patch.MinColumn + column*step, patch.MinRow + row*step}
				if _, found := made.values[key]; found {
					continue
				}
				if len(made.values) >= motion.ground.detail.Points ||
					int64(len(made.values)) >= motion.bytes/transitionValueBytes {
					return nil, fmt.Errorf("island: terrain transition vertex or payload allowance exhausted")
				}
				var value transitionVertex
				value.from, value.normalFrom, err = a.at(key)
				if err != nil {
					return nil, err
				}
				value.to, value.normalTo, err = b.at(key)
				if err != nil {
					return nil, err
				}
				made.values[key] = value
			}
		}
	}
	for row := range made.side - 1 {
		for column := range made.side - 1 {
			near := row*made.side + column
			far := near + made.side
			made.faces = append(made.faces, [3]int{near, far + 1, near + 1}, [3]int{near, far, far + 1})
		}
	}
	return made, nil
}

// prepare builds the displayed interpolation in the current render origin.
// Normal vectors are blended before the mesh constructor normalizes them;
// subdivision and shadow-terminator interpolation still need visual acceptance.
func (change *terrainTransition) prepare(ctx context.Context, origin frame.Relative, share float64) (Prepared, error) {
	result := Prepared{Patches: change.patches, Origin: origin.Origin(),
		Limited: change.from.Limited || change.target.Limited, Samples: len(change.values),
		Transition: &TransitionRecord{Share: share, FromPatches: len(change.from.Patches),
			TargetPatches: len(change.target.Patches), PayloadBytes: int64(len(change.values)) * transitionValueBytes}}
	if !change.reported {
		result.NormalStencils, result.NormalHeightQueries = change.target.NormalStencils, change.target.NormalHeightQueries
	}
	for _, patch := range change.patches {
		points := make([]maths.Vec, change.side*change.side)
		var normals []maths.Vec
		if change.smooth {
			normals = make([]maths.Vec, len(points))
		}
		step := (patch.MaxColumn - patch.MinColumn) / (change.side - 1)
		for row := range change.side {
			if err := ctx.Err(); err != nil {
				return Prepared{}, err
			}
			for column := range change.side {
				key := lattice{patch.MinColumn + column*step, patch.MinRow + row*step}
				value := change.values[key]
				world, normal := value.from, value.normalFrom
				if share >= 1 {
					world, normal = value.to, value.normalTo
				} else if share > 0 {
					world = world.Add(value.to.Sub(world).Scale(share))
					normal = normal.Add(value.normalTo.Sub(normal).Scale(share))
				}
				point, err := origin.Point(world)
				if err != nil {
					return Prepared{}, err
				}
				index := row*change.side + column
				points[index] = point
				if normals != nil {
					normals[index] = normal
				}
			}
		}
		surface, err := mesh.New(points, change.faces, normals, change.smooth)
		if err != nil {
			return Prepared{}, err
		}
		result.Surfaces = append(result.Surfaces, surface)
	}
	change.reported = true
	return result, nil
}
