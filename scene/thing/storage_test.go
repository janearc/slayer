package thing

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"testing"
)

// Two material groups and direct placements may share one prototype.
// Their logical geometry must not become three copies of its storage.
func TestStorageCountsSharedPrototypeAcrossGroupsOnce(t *testing.T) {
	m, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	a, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: m, Space: maths.Local()}, {Mesh: m, Space: maths.Local()}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: m, Space: maths.Local()}})
	if err != nil {
		t.Fatal(err)
	}
	got := Storage([]Thing{{Shape: a}, {Shape: b}, {Shape: a}, {Shape: m}})
	if got.UniqueMeshes != 1 || got.UniqueGroups != 2 || got.Instances != 3 || got.StoredFaces != 1 || got.ReferencedFaces != 6 || got.MutableMeshes != 0 {
		t.Fatalf("prototype and placement accounting: %+v", got)
	}
	if got.OwnedGeometryBytes != m.Storage().Bytes+a.Storage().Bytes+b.Storage().Bytes {
		t.Fatalf("shared storage counted twice: %+v", got)
	}
	if a.MemberCount() != 2 || a.Member(0).Mesh != m {
		t.Fatal("storage changed the assembly")
	}
}

func TestStorageFlagsMutableAndUnaccountedGeometry(t *testing.T) {
	m, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	got := Storage([]Thing{{Shape: m}, {Shape: shape.Plane{}}, {Shape: unaccountedShape{}}})
	if got.MutableMeshes != 1 || len(got.Unknown) != 1 || got.StoredFaces != 1 {
		t.Fatalf("unknown storage claimed complete: %+v", got)
	}
}

type unaccountedShape struct{}

func (unaccountedShape) Bounds() maths.Box          { return maths.Box{} }
func (unaccountedShape) Distance(maths.Vec) float64 { return 0 }
