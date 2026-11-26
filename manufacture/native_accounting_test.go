package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestNativeMaturityCountsSharedLooks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		looks int
	}{
		// the walrus: the hide, and each side's head parts each with a
		// material of its own
		{"bay-seal", 3}, {"walrus-fresh-5", 9},
		{"orca-clean-05", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row, err := referenceNativeKind(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			made, err := MakeWithComplexity(row, Sow(17), "quick")
			if err != nil {
				t.Fatal(err)
			}
			var record Maturity
			record.measure(made)
			if record.Looks != tc.looks {
				t.Fatalf("looks = %d; want %d distinct native materials", record.Looks, tc.looks)
			}
			if record.Parts != len(made.native) {
				t.Fatalf("parts = %d, expected %d", record.Parts, len(made.native))
			}
		})
	}
}

func TestNativeFaceCountUsesStoredGroupGeometry(t *testing.T) {
	triangle, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	second := maths.NewSubspace(maths.V(2, 0, 0), maths.Identity, 1)
	third := maths.NewSubspace(maths.V(4, 0, 0), maths.Identity, 1)
	group, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: triangle},
		{Mesh: triangle, Space: second},
		{Mesh: triangle, Space: third},
	})
	if err != nil {
		t.Fatal(err)
	}
	skin, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	made := good{native: []thing.Thing{{Shape: group}, {Shape: skin}}}
	if group.FaceCount() != 3 || group.StoredFaces() != 1 {
		t.Fatal("fixture does not share a prototype")
	}
	if made.faces() != 2 {
		t.Fatalf("counted %d faces, want 2 stored faces", made.faces())
	}
}

func TestNativeBoundsWithoutExtentFollowLocalAndParentPlacement(t *testing.T) {
	near := maths.NewSubspace(maths.V(2, 0, 0), maths.Identity, 1)
	far := maths.NewSubspace(maths.V(-2, 1, 0), maths.Identity, 2)
	made := good{native: []thing.Thing{
		{Shape: shape.Sphere{Radius: 1}, Motion: thing.Still(near)},
		{Shape: shape.Sphere{Radius: .5}, Motion: thing.Still(far)},
	}}
	want := maths.Box{Min: maths.V(-3, -1, -1), Max: maths.V(3, 2, 1)}
	if got := made.Bounds(); got != want {
		t.Fatalf("bounds = %+v; want %+v", got, want)
	}
	at := maths.NewSubspace(maths.V(10, 0, 0), maths.Identity, 1)
	moved, err := made.placed(at)
	if err != nil {
		t.Fatal(err)
	}
	expected := maths.Box{Min: maths.V(7, -1, -1), Max: maths.V(13, 2, 1)}
	if got := moved.Bounds(); got != expected {
		t.Fatalf("moved bounds = %+v; want %+v", got, expected)
	}
	if got := made.Bounds(); got != want {
		t.Fatalf("placement mutated original bounds: %+v", got)
	}
	both, err := gather(made, moved)
	if err != nil {
		t.Fatal(err)
	}
	expected.Max.X = 13
	expected.Min.X = -3
	if got := both.Bounds(); got != expected {
		t.Fatalf("gathered bounds = %+v; want %+v", got, expected)
	}
}

func TestNativeUnboundedShapeRemainsConservativelyUnbounded(t *testing.T) {
	tilted := maths.NewSubspace(maths.V(1, 2, 3), maths.About(maths.XAxis, .4), 1)
	made := good{native: []thing.Thing{{Shape: shape.Plane{}, Motion: thing.Still(tilted)}}}
	if got := made.Bounds(); got != maths.Everywhere() {
		t.Fatalf("unbounded shape yielded %+v", got)
	}
}
