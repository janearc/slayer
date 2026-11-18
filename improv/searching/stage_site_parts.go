package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// siteBlocks batches scene-specific surfaces by material without thousands
// of separate acceleration structures.
type siteBlocks struct {
	vertices []maths.Vec
	faces    [][3]int
}

// box adds one box of the given size at a position to the batch.
func (b *siteBlocks) box(at, size maths.Vec) {
	first := len(b.vertices)
	for _, v := range []maths.Vec{
		maths.V(0, 0, 0), maths.V(size.X, 0, 0),
		maths.V(size.X, size.Y, 0), maths.V(0, size.Y, 0),
		maths.V(0, 0, size.Z), maths.V(size.X, 0, size.Z),
		maths.V(size.X, size.Y, size.Z), maths.V(0, size.Y, size.Z),
	} {
		b.vertices = append(b.vertices, v.Add(at))
	}
	for _, f := range [][3]int{{0, 2, 1}, {0, 3, 2}, {4, 5, 6}, {4, 6, 7}, {0, 1, 5}, {0, 5, 4}, {3, 7, 6}, {3, 6, 2}, {0, 4, 7}, {0, 7, 3}, {1, 2, 6}, {1, 6, 5}} {
		b.faces = append(b.faces, [3]int{first + f[0], first + f[1], first + f[2]})
	}
}

// place adds the batched faces to the world as one mesh with material.
func (b *siteBlocks) place(s *heardStage, material *thing.Material) error {
	if len(b.faces) == 0 {
		return nil
	}
	mesh, err := shape.NewMesh(b.vertices, b.faces, nil, false)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: mesh, Material: material})
	return nil
}

// placeSeparate retains the topology of independent construction parts.
// Adjacent boxes may touch without becoming one non-manifold welded solid.
// The existing group hierarchy batches them without changing winding checks.
func (b *siteBlocks) placeSeparate(s *heardStage, material *thing.Material) error {
	var members []shape.MeshMember
	for first := 0; first < len(b.faces); {
		low, high := b.faces[first][0], b.faces[first][0]
		end := first
		for end < len(b.faces) {
			f := b.faces[end]
			nextLow := min(f[0], f[1], f[2])
			if end > first && nextLow > high {
				break
			}
			low = min(low, nextLow)
			high = max(high, f[0], f[1], f[2])
			end++
		}
		faces := make([][3]int, end-first)
		for i, f := range b.faces[first:end] {
			faces[i] = [3]int{f[0] - low, f[1] - low, f[2] - low}
		}
		mesh, err := shape.NewMesh(b.vertices[low:high+1], faces, nil, false)
		if err != nil {
			return err
		}
		members = append(members, shape.MeshMember{Mesh: mesh, Space: maths.Local()})
		first = end
	}
	if len(members) == 0 {
		return nil
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: group, Material: material})
	return nil
}
