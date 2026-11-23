package shape_test

import (
	"encoding/csv"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// What the tests of package shape_test count in, each true by definition.
// They stand beside nought, unity and pair of package shape, which these
// tests cannot reach.
const (
	groupNought     = 0  // defined: no amount
	groupUnity      = 1  // defined: one
	groupPair       = 2  // defined: two
	groupCorners    = 3  // defined: the corners of a face
	groupFloatBits  = 64 // defined: the bits in the float64 a cell is read as
	groupPrefixBase = 0  // defined by strconv: the base comes from the prefix
)

// The files the tests of package shape_test read. Every number a test
// chooses is in numbers.yaml, and a test asks for it by name. Points and
// rays are rows of a csv table, asked for by name.
const (
	groupNumbersFile = "testdata/numbers.yaml"
	groupVectorFile  = "testdata/group-vectors.csv"
)

// groupNumbers is numbers.yaml, read once.
var groupNumbers = sync.OnceValues(groupReadNumbers)

// groupReadNumbers reads numbers.yaml.
func groupReadNumbers() (numbers.Numbers, error) {
	raw, err := os.ReadFile(groupNumbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(groupNumbersFile, raw)
}

// groupPicked is a number the tests chose, by its name in numbers.yaml. A
// name the file does not hold stops the test.
func groupPicked(t *testing.T, name string) float64 {
	t.Helper()
	chosen, err := groupNumbers()
	if err != nil {
		t.Fatal(err)
	}
	number, err := chosen.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// groupPickedCount is a picked number that counts something, as an int.
func groupPickedCount(t *testing.T, name string) int {
	t.Helper()
	return int(groupPicked(t, name))
}

// groupRows reads a csv file whose first row names its columns: each other
// row is a map from those names to its cells.
func groupRows(t *testing.T, file string) []map[string]string {
	t.Helper()
	opened, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	all, err := csv.NewReader(opened).ReadAll()
	if err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	header, body := all[groupNought], all[groupUnity:]
	rows := make([]map[string]string, len(body))
	for place, line := range body {
		rows[place] = map[string]string{}
		for column, name := range header {
			rows[place][name] = line[column]
		}
	}
	return rows
}

// groupCell is the number in a column of a row. A cell that is no number
// stops the test.
func groupCell(t *testing.T, row map[string]string, column string) float64 {
	t.Helper()
	got, err := strconv.ParseFloat(row[column], groupFloatBits)
	if err != nil {
		t.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// groupCellCount is the whole number in a column of a row, or a stop.
func groupCellCount(t *testing.T, row map[string]string, column string) int {
	t.Helper()
	got, err := strconv.ParseInt(row[column], groupPrefixBase, groupFloatBits)
	if err != nil {
		t.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return int(got)
}

// groupVectors are the points and rays of one csv table. A row called name
// has its place in x, y, z and, for a ray, its direction in dirx, diry,
// dirz.
type groupVectors struct {
	test  *testing.T
	file  string
	table []map[string]string
}

// groupVectorsIn reads the csv table of vectors in file.
func groupVectorsIn(t *testing.T, file string) groupVectors {
	t.Helper()
	return groupVectors{test: t, file: file, table: groupRows(t, file)}
}

// row is the row whose name column is name, or a stop naming the file.
func (vectors groupVectors) row(name string) map[string]string {
	vectors.test.Helper()
	for _, each := range vectors.table {
		if each["name"] == name {
			return each
		}
	}
	vectors.test.Fatalf("%s: no row called %q", vectors.file, name)
	return nil
}

// point is the place of the row called name.
func (vectors groupVectors) point(name string) maths.Vec {
	vectors.test.Helper()
	row := vectors.row(name)
	return maths.V(groupCell(vectors.test, row, "x"),
		groupCell(vectors.test, row, "y"), groupCell(vectors.test, row, "z"))
}

// ray is the ray of the row called name: from its place, along its
// direction.
func (vectors groupVectors) ray(name string) maths.Ray {
	vectors.test.Helper()
	row := vectors.row(name)
	direction := maths.V(groupCell(vectors.test, row, "dirx"),
		groupCell(vectors.test, row, "diry"),
		groupCell(vectors.test, row, "dirz"))
	return maths.Ray{At: vectors.point(name), Dir: direction}
}

// groupTrianglePoints are the corners of the triangle the tests of this
// package build their meshes from.
func groupTrianglePoints(t *testing.T) []maths.Vec {
	t.Helper()
	vectors := groupVectorsIn(t, groupVectorFile)
	return []maths.Vec{vectors.point("triangle-first"),
		vectors.point("triangle-second"), vectors.point("triangle-third")}
}

// groupTriangleFaces is the one face of the triangle, its three corners in
// order.
func groupTriangleFaces() [][groupCorners]int {
	return [][groupCorners]int{{groupNought, groupUnity, groupPair}}
}

func groupTriangle(t *testing.T) *mesh.Mesh {
	t.Helper()
	triangle, err := mesh.New(groupTrianglePoints(t), groupTriangleFaces(),
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return triangle
}

// groupPlaced is the members of a group, each the triangle in a place of
// its own, and the one mesh whose faces are those of the members carried
// out of their places by hand.
func groupPlaced(t *testing.T) ([]shape.MeshMember, *mesh.Mesh) {
	t.Helper()
	triangle := groupTriangle(t)
	columns := groupPickedCount(t, "group-grid-columns")
	spacing := groupPicked(t, "group-grid-spacing")
	heights := groupPickedCount(t, "group-height-cycle")
	turn := groupPicked(t, "group-turn-step")
	scaleStep := groupPicked(t, "group-scale-step")
	var members []shape.MeshMember
	var points []maths.Vec
	var faces [][groupCorners]int
	for i := range groupPickedCount(t, "group-members") {
		origin := maths.V(float64(i%columns)*spacing,
			float64(i/columns)*spacing, float64(i%heights))
		turned := maths.About(maths.YAxis, float64(i)*turn)
		scale := groupUnity + float64(i%groupPair)*scaleStep
		space := maths.NewSubspace(origin, turned, scale)
		placedTriangle := shape.MeshMember{Mesh: triangle, Space: space}
		members = append(members, placedTriangle)
		first, second, third := triangle.Corners(groupNought)
		faces = append(faces, [groupCorners]int{len(points),
			len(points) + groupUnity, len(points) + groupPair})
		points = append(points, space.Out(first), space.Out(second),
			space.Out(third))
	}
	flat, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return members, flat
}

// groupAgrees checks that the group and the flat mesh give the same
// answers for ray, and says whether the ray met a face.
func groupAgrees(t *testing.T, group *shape.MeshGroup, flat *mesh.Mesh,
	ray maths.Ray) bool {
	t.Helper()
	far := groupPicked(t, "group-ray-far")
	tolerance := groupPicked(t, "group-hit-tolerance")
	groupAlong, groupHit := group.Hit(ray, groupNought, far)
	shaded, normal, hitNormal := group.HitNormal(ray, groupNought, far)
	shadedMoved := math.Abs(shaded-groupAlong) > tolerance
	if hitNormal != groupHit || groupHit && shadedMoved {
		t.Fatal("shading lookup changed the intersection")
	}
	flatAlong, flatHit := flat.Hit(ray, groupNought, far)
	flatMoved := math.Abs(groupAlong-flatAlong) > tolerance
	if groupHit != flatHit || groupHit && flatMoved {
		t.Fatalf("ray %v: group %g/%v, flat %g/%v", ray, groupAlong, groupHit,
			flatAlong, flatHit)
	}
	if !groupHit {
		return false
	}
	at := ray.Point(groupAlong)
	floor := groupPicked(t, "group-dot-floor")
	if normal.Dot(flat.Normal(at)) < floor {
		t.Fatal("hit normal differs from the transformed face")
	}
	if group.Normal(at).Dot(flat.Normal(at)) < floor {
		t.Fatal("transformed normal differs")
	}
	surface := groupPicked(t, "group-surface-tolerance")
	if math.Abs(group.Distance(at)) > surface {
		t.Fatal("hit point is not on group surface")
	}
	return true
}

// TestGroupMatchesExplicitlyTransformedFaces says that a group of placed
// members answers every ray as the one mesh of the same faces, put in their
// places by hand, does.
func TestGroupMatchesExplicitlyTransformedFaces(t *testing.T) {
	members, flat := groupPlaced(t)
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	vectors := groupVectorsIn(t, groupVectorFile)
	downward := vectors.point("downward")
	height := groupPicked(t, "group-ray-height")
	stop := groupPicked(t, "group-scan-stop")
	step := groupPicked(t, "group-scan-step")
	hits := groupNought
	// Avoid mathematical edge rays: rotating a ray versus rotating vertices
	// can round to opposite sides of the same zero-area silhouette boundary.
	for x := groupPicked(t, "group-scan-start-x"); x < stop; x += step {
		for y := groupPicked(t, "group-scan-start-y"); y < stop; y += step {
			ray := maths.Ray{At: maths.V(x, y, height), Dir: downward}
			if groupAgrees(t, group, flat, ray) {
				hits++
			}
		}
	}
	if hits < groupPickedCount(t, "group-least-hits") {
		t.Fatalf("only %d rays exercised group surfaces", hits)
	}
	if group.StoredFaces() != groupUnity ||
		group.FaceCount() != groupPickedCount(t, "group-members") {
		t.Fatal("group expanded shared geometry")
	}
	handedOut := group.Members()
	first := groupNought
	handedOut[first].Mesh = nil
	if group.Members()[first].Mesh == nil {
		t.Fatal("caller changed group members")
	}
}

// groupRefusal is a list of members a group refuses, and the words its
// refusal says.
type groupRefusal struct {
	why     string
	members []shape.MeshMember
	want    string
}

// groupRefusals are the lists of members the group refuses: none, a shape
// that is no mesh, a group, a placement that is no placement, and a mesh
// not made by NewMesh.
func groupRefusals(t *testing.T) []groupRefusal {
	t.Helper()
	triangle := groupTriangle(t)
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle}})
	if err != nil {
		t.Fatal(err)
	}
	ball := shape.Sphere{Radius: groupUnity}
	unbuilt := &shape.Mesh{Vertices: groupTrianglePoints(t),
		Faces: groupTriangleFaces()}
	negative := maths.Local()
	negative.Scale = -groupUnity
	nowhere := maths.Local()
	nowhere.Origin = maths.V(math.NaN(), groupNought, groupNought)
	return []groupRefusal{
		{"no members", nil, "member count"},
		{"a sphere", []shape.MeshMember{{Mesh: ball}}, "group needs meshes"},
		{"a group", []shape.MeshMember{{Mesh: group}}, "group needs meshes"},
		{"a negative scale", []shape.MeshMember{{Mesh: triangle,
			Space: negative}}, "placement"},
		{"a place that is no number", []shape.MeshMember{{Mesh: triangle,
			Space: nowhere}}, "placement"},
		{"a mesh literal, not made by NewMesh",
			[]shape.MeshMember{{Mesh: unbuilt}}, "made by NewMesh"},
	}
}

// TestGroupRejectsEmptyNestedAndInvalidTransforms says that a group refuses
// each list of members in groupRefusals, in the words of its row.
func TestGroupRejectsEmptyNestedAndInvalidTransforms(t *testing.T) {
	for _, row := range groupRefusals(t) {
		_, err := shape.NewMeshGroup(row.members)
		if err == nil || !strings.Contains(err.Error(), row.want) {
			t.Errorf("%s: error %v, want one saying %q", row.why, err, row.want)
		}
	}
}

// A member of each kind the group takes, in a place that is right, is
// accepted: the refusals above are for what is wrong with a member, and
// nothing else.
func TestGroupAcceptsABuiltMeshOfEachKind(t *testing.T) {
	built, err := shape.NewMesh(groupTrianglePoints(t), groupTriangleFaces(),
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range []shape.Shape{groupTriangle(t), built} {
		members := []shape.MeshMember{{Mesh: member}}
		if _, err := shape.NewMeshGroup(members); err != nil {
			t.Errorf("%T refused: %v", member, err)
		}
	}
}
