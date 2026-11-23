//go:build darwin

package gpu

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// The native ray-tracing plan and its instance-to-face mapping must be the
// same after streaming, including shared instances and modified geometry.
// Constructing an Acceleration is a CPU-only check; it does not open Metal.
func TestLowRetentionNativePlanAndFirsts(t *testing.T) {
	reference := shapeFan(t, 1, fanFaces, false)
	shared := groupMesh(t, 2)
	group := groupOf(t,
		shape.MeshMember{Mesh: reference},
		shape.MeshMember{Mesh: shared},
		shape.MeshMember{Mesh: shared, Space: maths.Subspace{Origin: maths.XAxis}},
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
		if !bytes.Equal(want.nativeFirsts(), got.nativeFirsts()) {
			t.Fatalf("frame %d: native firsts differ", frame)
		}
		wplan, wroot, err := want.accelerationPlan()
		if err != nil {
			t.Fatal(err)
		}
		gplan, groot, err := got.accelerationPlan()
		if err != nil {
			t.Fatal(err)
		}
		if wroot != groot || !reflect.DeepEqual(wplan, gplan) {
			t.Fatalf("frame %d: native acceleration plan differs", frame)
		}
		keeper.turn()
	}
}
