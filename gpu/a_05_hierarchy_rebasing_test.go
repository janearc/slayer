package gpu

import (
	"encoding/binary"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Local links are relative before packing and global afterwards. Decode the
// actual wire data with a bounded walk before any GPU can follow a bad cycle.
func TestPackedSharedHierarchyHasRebasedLinksAndEveryMemberExactlyOnce(t *testing.T) {
	mesh, err := shape.NewMesh([]maths.Vec{{}, maths.XAxis, maths.YAxis},
		[][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var members []shape.MeshMember
	for i := range 5 {
		members = append(members, shape.MeshMember{Mesh: mesh,
			Space: maths.Subspace{Origin: maths.V(float64(i)*3, 0, 0)}})
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	for _, origin := range []maths.Vec{maths.V(0, 0, -3), maths.V(0, 4, -3)} {
		world.Add(thing.Thing{Shape: group,
			Motion: thing.Still(maths.Subspace{Origin: origin})})
	}
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.counts.instances != 7 || len(encoded.groups) != 1 {
		t.Fatalf("two placements did not share five local members: %d instances, %d groups",
			encoded.counts.instances, len(encoded.groups))
	}
	word := func(data []byte, stride, record, field, lane int) int {
		t.Helper()
		at := record*stride + field + lane*4
		if record < 0 || at < 0 || at+4 > len(data) {
			t.Fatalf("packed index %d leaves its buffer", record)
		}
		return int(int32(binary.LittleEndian.Uint32(data[at:])))
	}
	instanceStride, nodeStride := instanceStruct.size(), nodeStruct.size()
	instanceInfo, nodeInfo := fieldOffset(instanceStruct, "info"), fieldOffset(nodeStruct, "info")
	nodeCount := len(encoded.tops.bytes) / nodeStride
	localRoot := word(encoded.instances.bytes, instanceStride, 0, instanceInfo, 3) - 1
	if localRoot <= 0 || localRoot >= nodeCount {
		t.Fatalf("local root is not after the global root: %d/%d", localRoot, nodeCount)
	}
	if other := word(encoded.instances.bytes, instanceStride, 1, instanceInfo, 3) - 1; other != localRoot {
		t.Fatal("world placements did not share their local hierarchy")
	}
	walk := func(root, firstInstance, pastInstance int) {
		t.Helper()
		pending := []int{root}
		visited, leaves := map[int]bool{}, map[int]bool{}
		for len(pending) > 0 {
			index := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if index < 0 || index >= nodeCount || visited[index] || len(visited) >= nodeCount {
				t.Fatalf("packed hierarchy cycles, repeats or leaves its buffer at node %d", index)
			}
			visited[index] = true
			first := word(encoded.tops.bytes, nodeStride, index, nodeInfo, 0)
			count := word(encoded.tops.bytes, nodeStride, index, nodeInfo, 1)
			if count == 0 {
				pending = append(pending, first, first+1)
				continue
			}
			if count < 0 || first < firstInstance || first+count > pastInstance {
				t.Fatalf("leaf range %d+%d leaves instances [%d,%d)", first, count, firstInstance, pastInstance)
			}
			for instance := first; instance < first+count; instance++ {
				if leaves[instance] {
					t.Fatalf("instance %d occurs twice", instance)
				}
				leaves[instance] = true
			}
		}
		if len(leaves) != pastInstance-firstInstance {
			t.Fatalf("hierarchy retained %d of %d instances", len(leaves), pastInstance-firstInstance)
		}
	}
	walk(0, 0, 2)
	walk(localRoot, 2, 7)
}
