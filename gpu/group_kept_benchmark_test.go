package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// BenchmarkGroupedMeshEncoding measures one retained prototype at 64
// placements, excluding initial mesh construction and all device work.
func BenchmarkGroupedMeshEncoding(b *testing.B) {
	const side = 32
	points := make([]maths.Vec, 0, (side+1)*(side+1))
	faces := make([][3]int, 0, side*side*2)
	for y := 0; y <= side; y++ {
		for x := 0; x <= side; x++ {
			corner := maths.V(float64(x), float64(y), 0)
			points = append(points, corner)
		}
	}
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			a := y*(side+1) + x
			faces = append(faces, [3]int{a, a + 1, a + side + 1}, [3]int{a + 1, a + side + 2, a + side + 1})
		}
	}
	prototype, err := mesh.New(points, faces, nil, false)
	if err != nil {
		b.Fatal(err)
	}
	members := make([]shape.MeshMember, 64)
	for i := range members {
		corner := maths.V(float64(i*40), 0, 0)
		members[i] = shape.MeshMember{Mesh: prototype, Space: maths.NewSubspace(
			corner, maths.Identity, 1)}
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		b.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group})
	for _, retained := range []bool{false, true} {
		name := "grow"
		if retained {
			name = "retain"
		}
		b.Run(name, func(b *testing.B) {
			var keeper *keptTrees
			if retained {
				keeper = newKeptTrees()
				if _, err := encodeKept(world, nil, maths.Subspace{}, nil, 0, 8, 8, keeper); err != nil {
					b.Fatal(err)
				}
				keeper.turn()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := encodeKept(world, nil, maths.Subspace{}, nil, 0, 8, 8, keeper); err != nil {
					b.Fatal(err)
				}
				if keeper != nil {
					keeper.turn()
				}
			}
		})
	}
}
