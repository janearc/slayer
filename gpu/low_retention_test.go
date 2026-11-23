package gpu

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// The low-retention route must feed the same bytes to the device across
// changing frames, while retaining no CPU mesh trees or CPU scene pool for the next frame.
func TestLowRetentionMatchesFullEncodeAcrossFrames(t *testing.T) {
	flat := shapeFan(t, 1, fanFaces, false)
	keeper := newKeptTrees()
	eye := maths.NewSubspace(maths.V(0, 0, fanBack), maths.Identity, 1)
	for frame := range fanFrames {
		world := fanWorld(t, flat, frame)
		at := time.Duration(frame) * fanStep
		t.Setenv("SLAYER_LOW_RETENTION", "")
		want, err := encodeKept(world, nil, eye, nil, at, fanAcross, fanDown, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("SLAYER_LOW_RETENTION", "1")
		got, err := encodeKept(world, nil, eye, nil, at, fanAcross, fanDown, keeper)
		if err != nil {
			t.Fatal(err)
		}
		if !sameInputs(got, want) {
			t.Fatalf("frame %d differs", frame)
		}
		if got.keeper != nil || len(got.pending) != 0 {
			t.Fatal("scene kept a tree or pending geometry")
		}
		keeper.turn()
		if keeper.held() != 0 || keeper.pool != nil {
			t.Fatal("keeper retained geometry")
		}
	}
}

// Low retention must release each grown tree as it is pooled, rather than
// keeping an entire scene's trees in the pending list.
func TestLowRetentionPoolsEachTreeImmediately(t *testing.T) {
	t.Setenv("SLAYER_LOW_RETENTION", "1")
	encoded := newScene()
	encoded.lowRetention = true
	first := meshTree(slayersFan(t, 3))
	second := meshTree(slayersFan(t, 4))
	p1 := encoded.pend(pendingMesh{tree: first})
	if len(encoded.pending) != 0 || len(encoded.faces.triangles) != len(first.triangles) || p1.root != 0 {
		t.Fatal("first tree not immediately pooled")
	}
	p2 := encoded.pend(pendingMesh{tree: second})
	if len(encoded.pending) != 0 || len(encoded.faces.triangles) != len(first.triangles)+len(second.triangles) || p2.root != len(first.nodes) {
		t.Fatal("second tree not pooled at next root")
	}
}

// Grouped and repeated meshes keep their instance ordering and react to a
// mutable reference mesh's changed geometry just as a fresh encode does.
func TestLowRetentionGroupedInstancesAndChangedGeometry(t *testing.T) {
	reference := shapeFan(t, 1, fanFaces, false)
	shared := groupMesh(t, 2)
	group := groupOf(t,
		shape.MeshMember{Mesh: reference},
		shape.MeshMember{Mesh: shared, Space: maths.Subspace{Origin: maths.XAxis}},
		shape.MeshMember{Mesh: shared, Space: maths.Subspace{Origin: maths.YAxis}},
	)
	world := worldWithGroup(group)
	keeper := newKeptTrees()
	for frame := range 3 {
		if frame == 2 {
			reference.Vertices[1] = reference.Vertices[1].Add(maths.V(0, 0, fanRise))
		}
		t.Setenv("SLAYER_LOW_RETENTION", "")
		want := encodeGroupFrame(t, world, nil)
		t.Setenv("SLAYER_LOW_RETENTION", "1")
		got := encodeGroupFrame(t, world, keeper)
		requireSameInputs(t, want, got)
		keeper.turn()
		if keeper.held() != 0 || keeper.pool != nil {
			t.Fatalf("frame %d retained CPU trees or pool", frame)
		}
	}
}
