package thing

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// floatBytes is the bytes in a float64, which a height field holds its
// samples and slopes in.
const floatBytes = 8 // defined: a float64 is eight bytes

// The corners of the triangle the mutable mesh is made of, by csv.
const (
	storageCornersFile = "testdata/storage-corners.csv"
	storageCorners     = 3 // defined: a triangle has three corners
)

// Two material groups and direct placements may share one prototype.
// Their logical geometry must not become three copies of its storage: the
// bytes of the whole are the groups' own, and the mesh's once, which is
// the bytes of each counted alone less the mesh counted again for each
// group after the first.
func TestStorageCountsSharedPrototypeAcrossGroupsOnce(t *testing.T) {
	prototype := triangle(t)
	first, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: prototype, Space: maths.Local()},
		{Mesh: prototype, Space: maths.Local()}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: prototype, Space: maths.Local()}})
	if err != nil {
		t.Fatal(err)
	}
	got := Storage([]Thing{{Shape: first}, {Shape: second}, {Shape: first},
		{Shape: prototype}})
	faces := prototype.Storage().Faces
	referenced := first.FaceCount()*pair + second.FaceCount() + faces
	if got.UniqueMeshes != unity || got.UniqueGroups != pair ||
		got.Instances != first.MemberCount()+second.MemberCount() ||
		got.StoredFaces != faces || got.ReferencedFaces != referenced ||
		got.MutableMeshes != nought {
		t.Fatalf("prototype and placement accounting: %+v", got)
	}
	alone := Storage([]Thing{{Shape: prototype}}).OwnedGeometryBytes
	withFirst := Storage([]Thing{{Shape: first}}).OwnedGeometryBytes
	withSecond := Storage([]Thing{{Shape: second}}).OwnedGeometryBytes
	if want := withFirst + withSecond - alone; got.OwnedGeometryBytes != want {
		t.Fatalf("shared storage counted twice: %d bytes, want %d",
			got.OwnedGeometryBytes, want)
	}
	if got.OwnedGeometryBytes >= withFirst+withSecond+alone {
		t.Fatal("the shared mesh was counted for every placement")
	}
	if first.MemberCount() != pair || first.Member(nought).Mesh != prototype {
		t.Fatal("storage changed the assembly")
	}
}

func TestStorageFlagsMutableAndUnaccountedGeometry(t *testing.T) {
	var corners []maths.Vec
	for _, row := range readSheet(t, storageCornersFile) {
		corners = append(corners, keyedVec(t, row, ""))
	}
	face := [storageCorners]int{nought, unity, pair}
	made, err := shape.NewMesh(corners, [][storageCorners]int{face}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	got := Storage([]Thing{{Shape: made}, {Shape: shape.Plane{}},
		{Shape: unaccountedShape{}}})
	if got.MutableMeshes != unity || len(got.Unknown) != unity ||
		got.StoredFaces != unity {
		t.Fatalf("unknown storage claimed complete: %+v", got)
	}
}

// A height field is counted by its samples: as many float64 as it has
// samples, worked here from its grid. A field shaded by vertex slopes
// carries two arrays more, as long, and so is three times as big. Neither
// is named among the shapes it cannot count, and a field placed twice is
// counted once. The anti-test: a shape it cannot count adds no bytes.
func TestStorageCountsAHeightFieldsSamplesAndSlopes(t *testing.T) {
	flat := grid(t)
	_, side, _, _, _, _ := flat.Pack()
	samples := uint64(side * side)
	got := Storage([]Thing{{Shape: flat}, {Shape: flat}})
	if want := samples * floatBytes; got.OwnedGeometryBytes != want ||
		len(got.Unknown) != nought {
		t.Errorf("a flat field of %d samples: %+v, want %d bytes", samples,
			got, want)
	}
	shaded, err := flat.WithApproximateSmoothNormals()
	if err != nil {
		t.Fatal(err)
	}
	got = Storage([]Thing{{Shape: shaded}})
	arrays := uint64(sampleArrays + slopeArrays)
	if want := arrays * samples * floatBytes; got.OwnedGeometryBytes != want {
		t.Errorf("a shaded field of %d samples: %d bytes, want %d", samples,
			got.OwnedGeometryBytes, want)
	}
	unaccounted := Storage([]Thing{{Shape: unaccountedShape{}}})
	if unaccounted.OwnedGeometryBytes != nought {
		t.Errorf("an unaccounted shape added %d bytes",
			unaccounted.OwnedGeometryBytes)
	}
}

// emptyBox is the box whose two corners are both at the origin.
func emptyBox() maths.Box {
	origin := maths.V(nought, nought, nought)
	return maths.NewBox(origin, origin)
}

type unaccountedShape struct{}

func (unaccountedShape) Bounds() maths.Box          { return emptyBox() }
func (unaccountedShape) Distance(maths.Vec) float64 { return nought }
