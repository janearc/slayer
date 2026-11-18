package thing

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
)

// gridSide is the test height field's samples a side.
const gridSide = 3

// triangle is a mesh of one face.
func triangle(t *testing.T) *mesh.Mesh {
	t.Helper()
	made, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// square is a mesh of two faces.
func square(t *testing.T) *mesh.Mesh {
	t.Helper()
	made, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(1, 1, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2},
		{0, 2, 3}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// grid is a flat height field gridSide samples a side.
func grid(t *testing.T) *shape.HeightField {
	t.Helper()
	made, err := shape.NewHeightField(make([]float64, gridSide*gridSide),
		gridSide, 1)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// knownWorld is two meshes of one and two faces, a group of the
// triangle three times and the square once, a sphere, a plane, a height
// field and a hidden triangle, with two puffs of mist.
func knownWorld(t *testing.T) *World {
	t.Helper()
	one, two := triangle(t), square(t)
	group, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: one, Space: maths.Local()}, {Mesh: one, Space: maths.Local()},
		{Mesh: one, Space: maths.Local()}, {Mesh: two, Space: maths.Local()}})
	if err != nil {
		t.Fatal(err)
	}
	return &World{Things: []*Thing{{Shape: one,
		Material: &Material{Name: "hull"}}, {Shape: two}, {Shape: group},
		{Shape: shape.Sphere{Radius: 1}}, {Shape: shape.Plane{}},
		{Shape: grid(t)}, {Shape: one, Attrs: Attrs{Hidden: true}}},
		Volumes: []*Volume{{Medium: medium.Puff{}}, {Medium: medium.Puff{}}}}
}

// a world of known geometry counts exactly: each mesh its faces, the
// group its instanced and stored faces and its members, and the totals
// every placement (staged) and each distinct mesh once (stored).
func TestACensusOfKnownGeometryIsExact(t *testing.T) {
	got, err := Count(knownWorld(t))
	if err != nil {
		t.Fatal(err)
	}
	faces := []int{1, 2, 5, -1, -1, -1, 1}
	for index, want := range faces {
		row := got.Placed[index]
		if (want < 0) != (row.Faces == nil) ||
			(row.Faces != nil && *row.Faces != want) {
			t.Errorf("thing %d: faces %v, want %d", index, row.Faces, want)
		}
	}
	group := got.Placed[2]
	if group.Kind != kindMeshGroup || *group.StoredFaces != 3 ||
		group.Instances != 4 {
		t.Errorf("the group: %+v", group)
	}
	total := got.Total
	if total.FacesStaged != 9 || total.FacesStored != 3 ||
		total.Instances != 4 || total.Things != 7 || total.Hidden != 1 ||
		total.HiddenFaces != 1 {
		t.Errorf("the totals: %+v", total)
	}
	if got.Placed[0].Material != "hull" || got.Placed[0].Kind != kindMesh {
		t.Errorf("the first thing: %+v", got.Placed[0])
	}
	if len(got.Volumes) != 1 || got.Volumes[0] !=
		(KindCount{Kind: "medium.Puff", Count: 2}) {
		t.Errorf("the volumes: %+v", got.Volumes)
	}
}

// the anti-test: an analytic shape is never given faces. The sphere, the
// plane and the height field are counted by kind, the height field says
// its grid, and none of them adds a face to the totals.
func TestAnAnalyticShapeIsNeverGivenFaces(t *testing.T) {
	world := &World{Things: []*Thing{{Shape: shape.Sphere{Radius: 1}},
		{Shape: shape.Plane{}}, {Shape: grid(t)}, {Shape: shape.Sphere{}}}}
	got, err := Count(world)
	if err != nil {
		t.Fatal(err)
	}
	want := []KindCount{{Kind: kindHeightField, Count: 1},
		{Kind: "shape.Plane", Count: 1}, {Kind: "shape.Sphere", Count: 2}}
	if len(got.Analytic) != len(want) {
		t.Fatalf("the analytic shapes: %+v", got.Analytic)
	}
	for index := range want {
		if got.Analytic[index] != want[index] {
			t.Errorf("analytic %d: %+v, want %+v", index,
				got.Analytic[index], want[index])
		}
	}
	if got.Total.FacesStaged != 0 || got.Total.FacesStored != 0 {
		t.Errorf("analytic shapes made faces: %+v", got.Total)
	}
	if len(got.Grids) != 1 || got.Grids[0] != (GridCount{Index: 2,
		Samples: gridSide}) {
		t.Errorf("the grids: %+v", got.Grids)
	}
	text, err := json.Marshal(got.Placed)
	if err != nil || strings.Contains(string(text), "faces") {
		t.Errorf("an analytic row says faces: %s (%v)", text, err)
	}
}

// the anti-test: a world that is missing, or holds a nil thing or a nil
// volume, is refused by place and not counted as nothing.
func TestACensusRefusesWhatIsMissing(t *testing.T) {
	for _, each := range []struct {
		world *World
		said  string
	}{
		{nil, "no world"},
		{&World{Things: []*Thing{nil}}, "thing 0 is nil"},
		{&World{Volumes: []*Volume{nil}}, "volume 0 is nil"},
	} {
		if _, err := Count(each.world); err == nil ||
			!strings.Contains(err.Error(), each.said) {
			t.Errorf("want %q, got %v", each.said, err)
		}
	}
}
