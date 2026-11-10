//go:build darwin

package gpu

import (
	"encoding/binary"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Native IDs must address the same pooled faces and shared members as the
// software instance records, even with distinct groups and reused meshes.
func TestNativeFaceLookupAgreesWithPackedInstancesAcrossSharedGroups(t *testing.T) {
	triangleAt := func(x float64) *shape.Mesh {
		t.Helper()
		made, err := shape.NewMesh([]maths.Vec{
			maths.V(x, 0, 0), maths.V(x+1, 0, 0), maths.V(x, 1, 0),
		}, [][3]int{{0, 1, 2}}, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	shared := triangleAt(0)
	a, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: triangleAt(2)}, {Mesh: triangleAt(4)},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: triangleAt(6)}, {Mesh: triangleAt(8)}, {Mesh: shared},
	})
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shared})
	world.Add(thing.Thing{Shape: a,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(-10, 0, 0)})})
	world.Add(thing.Thing{Shape: b,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(10, 0, 0)})})
	world.Add(thing.Thing{Shape: a,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 10, 0)})})
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.groups) != 2 || len(encoded.standings) != 4 ||
		encoded.counts.instances != 9 || len(encoded.faces.triangles) != 5 {
		t.Fatal("fixture lost distinct groups, repeated placement or pooled mesh reuse")
	}
	native := encoded.nativeFirsts()
	// Instance is six float4 transform lanes and one int4 info lane.
	// Node is two float4 bounds and one int4 info lane.
	const instanceStride, instanceInfo = 7 * 16, 6 * 16
	const nodeStride, nodeInfo = 3 * 16, 2 * 16
	if len(native) != 9*4 || len(encoded.instances.bytes) != 9*instanceStride {
		t.Fatal("native and software instance tables have different lengths")
	}
	// This bounded wire walk makes no assumption about leaf size or
	// split order. It reads the actual minimum leaf index in a subtree.
	firstLeaf := func(data []byte, root int) int {
		t.Helper()
		count := len(data) / nodeStride
		pending := []int{root}
		visited := map[int]bool{}
		first := -1
		for len(pending) > 0 {
			index := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if index < 0 || index >= count || visited[index] {
				t.Fatalf("packed tree repeats or leaves its buffer at %d", index)
			}
			visited[index] = true
			at := index*nodeStride + nodeInfo
			start := int(int32(binary.LittleEndian.Uint32(data[at:])))
			held := int(int32(binary.LittleEndian.Uint32(data[at+4:])))
			if held == 0 {
				pending = append(pending, start, start+1)
				continue
			}
			if held < 0 || start < 0 {
				t.Fatal("packed tree has a negative leaf range")
			}
			if first < 0 || start < first {
				first = start
			}
		}
		if first < 0 {
			t.Fatal("nonempty fixture has no leaf")
		}
		return first
	}
	for row := range 9 {
		at := row*instanceStride + instanceInfo
		meshRoot := int(int32(binary.LittleEndian.Uint32(encoded.instances.bytes[at:])))
		groupRoot := int(int32(binary.LittleEndian.Uint32(encoded.instances.bytes[at+12:]))) - 1
		var want int
		if groupRoot >= 0 {
			want = firstLeaf(encoded.tops.bytes, groupRoot)
			if want < 4 || want >= 9 {
				t.Fatal("world group does not address retained local members")
			}
		} else {
			want = firstLeaf(encoded.nodes.bytes, meshRoot)
			if want < 0 || want >= 5 {
				t.Fatal("mesh does not address its pooled triangles")
			}
		}
		got := int(int32(binary.LittleEndian.Uint32(native[row*4:])))
		if got != want {
			t.Fatalf("native instance %d starts at %d, packed hierarchy starts at %d",
				row, got, want)
		}
	}
}
