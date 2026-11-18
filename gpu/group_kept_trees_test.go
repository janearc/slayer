package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func groupMesh(t *testing.T, scale float64) *mesh.Mesh {
	t.Helper()
	points := []maths.Vec{maths.V(-scale, -scale, 0), maths.V(scale, -scale, 0), maths.V(0, scale, 0)}
	made, err := mesh.New(points, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func groupOf(t *testing.T, members ...shape.MeshMember) *shape.MeshGroup {
	t.Helper()
	made, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func worldWithGroup(group *shape.MeshGroup) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group})
	return world
}

func requireSameInputs(t *testing.T, one, two *scene) {
	t.Helper()
	first, second := one.inputs(), two.inputs()
	if len(first) != len(second) {
		t.Fatalf("input counts differ: %d and %d", len(first), len(second))
	}
	for index := range first {
		if string(first[index]) != string(second[index]) {
			t.Fatalf("input %d differs", index)
		}
	}
}

func encodeGroupFrame(t *testing.T, world *thing.World, keeper *keptTrees) *scene {
	t.Helper()
	encoded, err := encodeKept(world, nil, maths.Subspace{}, nil, 0, 8, 8, keeper)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestGroupedImmutableMeshesReuseKeptTreesAndPreservePackedInputs(t *testing.T) {
	shared, added := groupMesh(t, 1), groupMesh(t, 2)
	firstWorld := worldWithGroup(groupOf(t, shape.MeshMember{Mesh: shared}))
	changedWorld := worldWithGroup(groupOf(t,
		shape.MeshMember{Mesh: shared},
		shape.MeshMember{Mesh: added, Space: maths.Subspace{Origin: maths.XAxis}},
	))
	keeper := newKeptTrees()

	encodeGroupFrame(t, firstWorld, keeper)
	if len(keeper.used) != 1 || keeper.served != 0 {
		t.Fatalf("first frame used %d trees and served %d", len(keeper.used), keeper.served)
	}
	keeper.turn()
	if keeper.held() != 1 {
		t.Fatalf("first frame retained %d trees", keeper.held())
	}

	served := keeper.served
	got := encodeGroupFrame(t, changedWorld, keeper)
	if keeper.served-served != 1 || len(keeper.used) != 2 {
		t.Fatalf("changed group served %d trees and used %d", keeper.served, len(keeper.used))
	}
	want := encodeGroupFrame(t, changedWorld, nil)
	requireSameInputs(t, want, got)
	keeper.turn()
	if keeper.held() != 2 {
		t.Fatalf("changed frame retained %d trees", keeper.held())
	}

	served = keeper.served
	last := encodeGroupFrame(t, firstWorld, keeper)
	fresh := encodeGroupFrame(t, firstWorld, nil)
	if keeper.served-served != 1 || len(keeper.used) != 1 {
		t.Fatalf("last frame served %d trees and used %d", keeper.served, len(keeper.used))
	}
	requireSameInputs(t, fresh, last)
	keeper.turn()
	if keeper.held() != 1 {
		t.Fatalf("last frame retained %d stale trees", keeper.held())
	}
}

func TestGroupedMutableReferenceMeshTracksChangesAndKeepsSmoothRefusal(t *testing.T) {
	reference, err := shape.NewMesh(
		[]maths.Vec{{}, maths.XAxis, maths.YAxis}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group := groupOf(t, shape.MeshMember{Mesh: reference})
	world := worldWithGroup(group)
	keeper := newKeptTrees()
	first := encodeGroupFrame(t, world, keeper)
	keeper.turn()
	encodeGroupFrame(t, world, keeper)
	if keeper.served != 1 || keeper.held() != 1 {
		t.Fatalf("unchanged reference mesh served %d and retained %d trees", keeper.served, keeper.held())
	}
	keeper.turn()

	reference.Vertices[1] = maths.V(2, 0, 0)
	served := keeper.served
	changed := encodeGroupFrame(t, world, keeper)
	if keeper.served != served {
		t.Fatal("changed mesh reused its previous face tree")
	}
	requireSameInputs(t, encodeGroupFrame(t, world, nil), changed)
	firstInputs, changedInputs := first.inputs(), changed.inputs()
	same := true
	for i := range firstInputs {
		same = same && string(firstInputs[i]) == string(changedInputs[i])
	}
	if same {
		t.Fatal("changing a group member's vertices did not change the inputs")
	}

	reference.Smooth = true
	reference.Normals = []maths.Vec{maths.ZAxis, maths.ZAxis, maths.ZAxis}
	if _, err := encodeKept(world, nil, maths.Subspace{}, nil, 0, 8, 8, keeper); err == nil ||
		!strings.Contains(err.Error(), "a smooth reference mesh in a group") {
		t.Fatalf("mutated smooth group member refusal changed: %v", err)
	}
}
