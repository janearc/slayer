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

// The places in a list of corners, and what a face is made of; each true
// by definition.
const (
	censusCorners = 3            // defined: the corners of a triangle
	censusFirst   = nought       // defined: the first place in a list
	censusSecond  = unity        // defined: the place after the first
	censusThird   = pair         // defined: the place after the second
	censusFourth  = pair + unity // defined: the place after the third
	censusNoFaces = nought       // defined: an analytic shape has none
)

// The csv files the census tests read.
const (
	censusThings = "testdata/census-things.csv"
	censusTotals = "testdata/census-totals.csv"
	censusKinds  = "testdata/census-analytic.csv"
)

// The rows and columns the census tests ask for by name.
const (
	censusKnown  = "known"
	censusHeight = "height-field"
	censusPlane  = "plane"
	censusSphere = "sphere"
	censusGrids  = "grids"
	censusGridAt = "grid-place"
	censusCount  = "count"
	censusFaces  = "faces"
)

// censusFaceKey is the key json gives a face count.
const censusFaceKey = "faces"

// censusGridSide is the test height field's samples a side.
func censusGridSide(t *testing.T) int {
	t.Helper()
	return pickedCount(t, "census-grid-side")
}

// censusSphereShape is the sphere the census tests count.
func censusSphereShape(t *testing.T) shape.Sphere {
	t.Helper()
	return shape.Sphere{Radius: picked(t, "census-sphere-radius")}
}

// censusMesh is a mesh of the corners and faces, or a stop.
func censusMesh(t *testing.T, corners []maths.Vec,
	faces [][censusCorners]int) *mesh.Mesh {
	t.Helper()
	made, err := mesh.New(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// triangle is a mesh of one face.
func triangle(t *testing.T) *mesh.Mesh {
	t.Helper()
	corners := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(nought, unity, nought)}
	faces := [][censusCorners]int{
		{censusFirst, censusSecond, censusThird}}
	return censusMesh(t, corners, faces)
}

// square is a mesh of two faces.
func square(t *testing.T) *mesh.Mesh {
	t.Helper()
	corners := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(unity, unity, nought),
		maths.V(nought, unity, nought)}
	faces := [][censusCorners]int{
		{censusFirst, censusSecond, censusThird},
		{censusFirst, censusThird, censusFourth}}
	return censusMesh(t, corners, faces)
}

// grid is a flat height field censusGridSide samples a side.
func grid(t *testing.T) *shape.HeightField {
	t.Helper()
	side := censusGridSide(t)
	heights := make([]float64, side*side)
	made, err := shape.NewHeightField(heights, side,
		picked(t, "census-grid-spacing"))
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
		{Shape: censusSphereShape(t)}, {Shape: shape.Plane{}},
		{Shape: grid(t)}, {Shape: one, Attrs: Attrs{Hidden: true}}},
		Volumes: []*Volume{{Medium: medium.Puff{}}, {Medium: medium.Puff{}}}}
}

// censusKnownTotals is the row of counts the known world must give.
func censusKnownTotals(t *testing.T) map[string]string {
	t.Helper()
	rows := readSheet(t, censusTotals)
	return rows.row(t, censusTotals, censusKnown)
}

// censusKindCount is the count in a named row of the analytic table.
func censusKindCount(t *testing.T, name string) int {
	t.Helper()
	rows := readSheet(t, censusKinds)
	return cellCount(t, rows.row(t, censusKinds, name), censusCount)
}

// censusHasNoFaces says a worked face count stands for a shape with none.
func censusHasNoFaces(want int) bool { return want < censusNoFaces }

// censusCheckFaces checks each placed thing's faces against its row.
func censusCheckFaces(t *testing.T, got Census) {
	t.Helper()
	for index, each := range readSheet(t, censusThings) {
		want := cellCount(t, each, censusFaces)
		row := got.Placed[index]
		if censusHasNoFaces(want) != (row.Faces == nil) ||
			(row.Faces != nil && *row.Faces != want) {
			t.Errorf("thing %d: faces %v, want %d", index, row.Faces, want)
		}
	}
}

// censusCheckGroup checks the group's kind, stored faces and instances.
func censusCheckGroup(t *testing.T, got Census, row map[string]string) {
	t.Helper()
	group := got.Placed[cellCount(t, row, "groupPlace")]
	if group.Kind != kindMeshGroup ||
		*group.StoredFaces != cellCount(t, row, "groupStored") ||
		group.Instances != cellCount(t, row, "groupInstances") {
		t.Errorf("the group: %+v", group)
	}
}

// censusCheckTotals checks the totals against the known world's row.
func censusCheckTotals(t *testing.T, total censusTotal,
	row map[string]string) {
	t.Helper()
	if total.FacesStaged != cellCount(t, row, "staged") ||
		total.FacesStored != cellCount(t, row, "stored") ||
		total.Instances != cellCount(t, row, "instances") ||
		total.Things != cellCount(t, row, "things") ||
		total.Hidden != cellCount(t, row, "hidden") ||
		total.HiddenFaces != cellCount(t, row, "hiddenFaces") {
		t.Errorf("the totals: %+v", total)
	}
}

// censusCheckVolumes checks the volumes are counted by kind.
func censusCheckVolumes(t *testing.T, got Census, row map[string]string) {
	t.Helper()
	want := kindCount{Kind: "medium.Puff", Count: cellCount(t, row, "puffs")}
	if len(got.Volumes) != cellCount(t, row, "volumeKinds") ||
		got.Volumes[censusFirst] != want {
		t.Errorf("the volumes: %+v", got.Volumes)
	}
}

// a world of known geometry counts exactly: each mesh its faces, the
// group its instanced and stored faces and its members, and the totals
// every placement (staged) and each distinct mesh once (stored).
func TestACensusOfKnownGeometryIsExact(t *testing.T) {
	got, err := Count(knownWorld(t))
	if err != nil {
		t.Fatal(err)
	}
	row := censusKnownTotals(t)
	censusCheckFaces(t, got)
	censusCheckGroup(t, got, row)
	censusCheckTotals(t, got.Total, row)
	first := got.Placed[censusFirst]
	if first.Material != "hull" || first.Kind != kindMesh {
		t.Errorf("the first thing: %+v", first)
	}
	censusCheckVolumes(t, got, row)
}

// censusAnalyticWorld is a sphere, a plane, a height field and a second
// sphere, in that order.
func censusAnalyticWorld(t *testing.T) *World {
	t.Helper()
	return &World{Things: []*Thing{{Shape: censusSphereShape(t)},
		{Shape: shape.Plane{}}, {Shape: grid(t)}, {Shape: shape.Sphere{}}}}
}

// the anti-test: an analytic shape is never given faces. The sphere, the
// plane and the height field are counted by kind, the height field says
// its grid, and none of them adds a face to the totals.
func TestAnAnalyticShapeIsNeverGivenFaces(t *testing.T) {
	got, err := Count(censusAnalyticWorld(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []kindCount{
		{Kind: kindHeightField, Count: censusKindCount(t, censusHeight)},
		{Kind: "shape.Plane", Count: censusKindCount(t, censusPlane)},
		{Kind: "shape.Sphere", Count: censusKindCount(t, censusSphere)}}
	if len(got.Analytic) != len(want) {
		t.Fatalf("the analytic shapes: %+v", got.Analytic)
	}
	for index := range want {
		if got.Analytic[index] != want[index] {
			t.Errorf("analytic %d: %+v, want %+v", index,
				got.Analytic[index], want[index])
		}
	}
	if got.Total.FacesStaged != censusNoFaces ||
		got.Total.FacesStored != censusNoFaces {
		t.Errorf("analytic shapes made faces: %+v", got.Total)
	}
	censusCheckGrids(t, got)
	text, err := json.Marshal(got.Placed)
	if err != nil || strings.Contains(string(text), censusFaceKey) {
		t.Errorf("an analytic row says faces: %s (%v)", text, err)
	}
}

// censusCheckGrids checks the one height field, its place and its side.
func censusCheckGrids(t *testing.T, got Census) {
	t.Helper()
	want := gridCount{Index: censusKindCount(t, censusGridAt),
		Samples: censusGridSide(t)}
	if len(got.Grids) != censusKindCount(t, censusGrids) ||
		got.Grids[censusFirst] != want {
		t.Errorf("the grids: %+v", got.Grids)
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
