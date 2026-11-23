package shape_test

import (
	"encoding/csv"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// What this file counts in, each true by definition.
const (
	groupAxisNought       = 0   // defined: no amount
	groupAxisUnity        = 1   // defined: one
	groupAxisHalf         = 0.5 // defined: one half
	groupAxisFloatBits    = 64  // defined: the bits in a float64
	groupAxisHeaderRows   = 1   // defined: the row of a table naming columns
	groupAxisHeightColumn = 1   // defined: the place of a height in its row
)

// The files this file reads.
const (
	groupAxisNumbersFile = "testdata/numbers.yaml"
	groupAxisLayersFile  = "testdata/group-axis-layers.csv"
)

// groupAxisBook is numbers.yaml, read once.
var groupAxisBook = sync.OnceValues(groupAxisReadBook)

// groupAxisReadBook reads numbers.yaml.
func groupAxisReadBook() (numbers.Numbers, error) {
	raw, err := os.ReadFile(groupAxisNumbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(groupAxisNumbersFile, raw)
}

// groupAxisPicked is a number this file chose, by its name in
// numbers.yaml. A name the file does not hold stops the test.
func groupAxisPicked(t *testing.T, name string) float64 {
	t.Helper()
	book, err := groupAxisBook()
	if err != nil {
		t.Fatal(err)
	}
	number, err := book.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// groupAxisLayers is the heights of the layers, in the order they are
// handed to the group, read from the table.
func groupAxisLayers(t *testing.T) []float64 {
	t.Helper()
	opened, err := os.Open(groupAxisLayersFile)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	rows, err := csv.NewReader(opened).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var heights []float64
	for _, row := range rows[groupAxisHeaderRows:] {
		height, err := strconv.ParseFloat(row[groupAxisHeightColumn],
			groupAxisFloatBits)
		if err != nil {
			t.Fatal(err)
		}
		heights = append(heights, height)
	}
	return heights
}

// groupAxisMembers is one triangle at each of the layers' heights.
func groupAxisMembers(t *testing.T) []shape.MeshMember {
	t.Helper()
	triangle := groupTriangle(t)
	var members []shape.MeshMember
	for _, height := range groupAxisLayers(t) {
		space := maths.Local()
		space.Origin = maths.V(groupAxisNought, groupAxisNought, height)
		members = append(members, shape.MeshMember{Mesh: triangle,
			Space: space})
	}
	return members
}

// groupAxisPlane fails unless a plane faces up, and a ray along it meets
// nothing.
func groupAxisPlane(t *testing.T) {
	t.Helper()
	plane := shape.Plane{}
	far := maths.V(groupAxisPicked(t, "group-axis-plane-x"), groupAxisNought,
		groupAxisPicked(t, "group-axis-plane-z"))
	if normal := plane.Normal(far); normal != maths.Up {
		t.Fatalf("plane normal: %v", normal)
	}
	along := maths.Ray{At: maths.V(groupAxisNought, groupAxisUnity,
		groupAxisNought), Dir: maths.XAxis}
	reach := groupAxisPicked(t, "group-axis-parallel-reach")
	if _, hit := plane.Hit(along, groupAxisNought, reach); hit {
		t.Fatal("parallel ray met the plane")
	}
}

// TestGroupSearchKeepsTheNearestLayerAlongItsLongestAxis says that layers
// handed to a group out of order are each found by a ray from just above
// it, at the distance to its top face.
func TestGroupSearchKeepsTheNearestLayerAlongItsLongestAxis(t *testing.T) {
	members := groupAxisMembers(t)
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	offset := groupAxisPicked(t, "group-axis-ray-offset")
	reach := groupAxisPicked(t, "group-axis-reach")
	for layer := range len(members) {
		ray := maths.Ray{At: maths.V(offset, offset,
			float64(layer)+groupAxisHalf), Dir: maths.ZAxis.Neg()}
		distance, normal, hit := group.HitNormal(ray, groupAxisNought, reach)
		if !hit || distance != groupAxisHalf || normal != maths.ZAxis {
			t.Fatalf("layer %d: distance %g normal %v hit %v",
				layer, distance, normal, hit)
		}
	}
	groupAxisPlane(t)
}
