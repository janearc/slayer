package gpu

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/geom"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// topsRow is one node of the tops buffer: its low and high corners and its
// info, as ints.
type topsRow struct {
	low, high [3]float32
	info      [4]int32
}

// topsRows reads every node of a tops buffer, as the kernel's Node is laid
// out: low, high and info, four words each.
func topsRows(t *testing.T, tops packer) []topsRow {
	t.Helper()
	const words = 12
	if len(tops.bytes)%(words*4) != 0 {
		t.Fatalf("tops is %d bytes, not whole nodes", len(tops.bytes))
	}
	word := func(at int) uint32 {
		return binary.LittleEndian.Uint32(tops.bytes[at*4:])
	}
	rows := make([]topsRow, len(tops.bytes)/(words*4))
	for index := range rows {
		first := index * words
		for axis := range 3 {
			rows[index].low[axis] = math.Float32frombits(word(first + axis))
			rows[index].high[axis] = math.Float32frombits(word(first + 4 + axis))
		}
		for number := range 4 {
			rows[index].info[number] = int32(word(first + 8 + number))
		}
	}
	return rows
}

// sphereAt is a plain sphere at a place along x.
func sphereAt(encoded *scene, along float64) {
	placed := maths.NewSubspace(maths.V(along, 0, 0), maths.Identity, 1)
	encoded.sphere(placed,
		1, 0, false)
}

// conifersOf stands count default conifers along y, as the scene's own
// conifer does.
func conifersOf(t *testing.T, encoded *scene, count int) {
	t.Helper()
	for index := range count {
		at := maths.V(0, float64(index)*100, 0)
		placed := maths.NewSubspace(at, maths.Identity, 2)
		tree, err := newGrown(placed,
			manufacture.DefaultConiferParams(), nil)
		if err != nil {
			t.Fatal(err)
		}
		encoded.conifer(tree, 0)
	}
}

// TestAnalyticTreeNamesEverythingOnce checks that every sphere and conifer
// is the one leaf of the tree, planes stand before the tree, and every
// branch's children are inside the nodes written.
func TestAnalyticTreeNamesEverythingOnce(t *testing.T) {
	const spheres, conifers = 40, 3
	encoded := newScene()
	for index := range spheres {
		sphereAt(encoded, float64(index)*10)
	}
	placed := maths.NewSubspace(maths.V(0, -1, 0), maths.Identity, 1)
	encoded.plane(placed,
		0, false)
	conifersOf(t, encoded, conifers)
	if err := encoded.analyticTree(); err != nil {
		t.Fatal(err)
	}
	place := encoded.analyticAt
	rows := topsRows(t, encoded.tops)
	if place.loose != 1 || place.nodes != 2*(spheres+conifers)-1 ||
		len(rows) != place.first+place.loose+place.nodes {
		t.Fatalf("place %+v for %d rows", place, len(rows))
	}
	seen := map[[2]int32]int{}
	for index, row := range rows[place.first:] {
		leaf := row.info[1] != 0
		if leaf {
			seen[[2]int32{row.info[2], row.info[0]}]++
			continue
		}
		next := int(row.info[0])
		if index < place.loose || next < place.first+place.loose ||
			next+1 >= len(rows) {
			t.Fatalf("node %d names children at %d", index, next)
		}
	}
	for index := range int32(spheres + 1) {
		if seen[[2]int32{analyticSphere, index}] != 1 {
			t.Errorf("sphere %d is in %d leaves", index,
				seen[[2]int32{analyticSphere, index}])
		}
	}
	for index := range int32(conifers) {
		if seen[[2]int32{analyticConifer, index}] != 1 {
			t.Errorf("conifer %d is in %d leaves", index,
				seen[[2]int32{analyticConifer, index}])
		}
	}
}

// TestAnalyticTreeFollowsTheMeshNodes checks the nodes are written after
// whatever the tops already hold, and the place says where.
func TestAnalyticTreeFollowsTheMeshNodes(t *testing.T) {
	encoded := newScene()
	for range 2 {
		encoded.tops.row(nodeStruct, values{"low": {0}, "high": {0},
			"info": {0, 0}})
	}
	encoded.counts.tops = 2
	sphereAt(encoded, 5)
	if err := encoded.analyticTree(); err != nil {
		t.Fatal(err)
	}
	if encoded.analyticAt.first != 2 || len(topsRows(t, encoded.tops)) != 3 {
		t.Fatalf("place %+v", encoded.analyticAt)
	}
}

// TestAnalyticTreeOfNothingWritesNothing is the anti-test: a world with no
// spheres and no conifers adds no node and leaves the place nought, which
// the kernel reads as no work.
func TestAnalyticTreeOfNothingWritesNothing(t *testing.T) {
	encoded := newScene()
	if err := encoded.analyticTree(); err != nil {
		t.Fatal(err)
	}
	if len(encoded.tops.bytes) != 0 || encoded.analyticAt != (analyticPlace{}) {
		t.Fatalf("an empty world wrote %d bytes, place %+v",
			len(encoded.tops.bytes), encoded.analyticAt)
	}
}

// TestAnalyticTreeOfPlanesAloneHasNoTree checks that planes alone stand
// loose, with no nodes after them to walk.
func TestAnalyticTreeOfPlanesAloneHasNoTree(t *testing.T) {
	encoded := newScene()
	placed := maths.NewSubspace(maths.V(0, 0, 0), maths.Identity, 1)
	encoded.plane(placed, 0,
		false)
	if err := encoded.analyticTree(); err != nil {
		t.Fatal(err)
	}
	if encoded.analyticAt.loose != 1 || encoded.analyticAt.nodes != 0 {
		t.Fatalf("place %+v", encoded.analyticAt)
	}
}

// TestAnalyticSphereOfNoFiniteBoxStandsLoose checks that a sphere whose box
// is not within a float32 is tried each time and not put in the tree.
func TestAnalyticSphereOfNoFiniteBoxStandsLoose(t *testing.T) {
	encoded := newScene()
	placed := maths.NewSubspace(maths.V(0, 0, 0), maths.Identity, 1)
	encoded.sphere(placed,
		math.MaxFloat64, 0, false)
	sphereAt(encoded, 3)
	if err := encoded.analyticTree(); err != nil {
		t.Fatal(err)
	}
	if encoded.analyticAt.loose != 1 || encoded.analyticAt.nodes != 1 {
		t.Fatalf("place %+v", encoded.analyticAt)
	}
}

// TestAnalyticBoxIsGrownAtADistantPlace checks the margin: a unit sphere
// six thousand kilometres out is boxed wider than float32 rounds there.
func TestAnalyticBoxIsGrownAtADistantPlace(t *testing.T) {
	const out = 6e6
	box, ok := boxAround(maths.V(out, 0, 0), 1)
	if !ok {
		t.Fatal("a sphere out there has a box")
	}
	if out-box.Low[0] < 1+out*analyticReach*0.99 ||
		box.High[0]-out < 1+out*analyticReach*0.99 {
		t.Fatalf("box %+v is no wider than float32 at %g", box, out)
	}
}

// TestConiferReachHoldsEveryFace checks that the sphere the box is made
// from holds every vertex of the conifer manufacture makes, for plain and
// for deep and wide-leaning trees.
func TestConiferReachHoldsEveryFace(t *testing.T) {
	plain := manufacture.DefaultConiferParams()
	deep := plain
	deep.Depth, deep.BranchStart, deep.Sides = mostConiferDepth, 1.2, 12
	for _, params := range []manufacture.ConiferParams{plain, deep} {
		reach := coniferReach(params)
		for _, face := range manufacture.Conifer(geom.Vec3{}, params) {
			for _, corner := range face.V {
				far := math.Sqrt(corner[0]*corner[0] + corner[1]*corner[1] +
					corner[2]*corner[2])
				if far > reach {
					t.Fatalf("a vertex %g from the foot, past reach %g",
						far, reach)
				}
			}
		}
	}
}

// TestAnalyticKindsAreTheKernels checks the kinds in the kernel's source
// are the kinds the encoder writes, and that the walks it names are there.
func TestAnalyticKindsAreTheKernels(t *testing.T) {
	source, err := os.ReadFile("trace.metal")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"analytic_sphere = 0;", "analytic_conifer = 1;",
		"walk_nearest = 0;", "walk_shadow = 1;", "walk_moonlit = 2;"} {
		if !strings.Contains(string(source), line) {
			t.Errorf("trace.metal lacks %q", line)
		}
	}
	if analyticSphere != 0 || analyticConifer != 1 {
		t.Fatal("the encoder's kinds are not the kernel's")
	}
}

// TestKernelLoopsNoLongerTryEverySphere is the anti-test for the loops this
// replaces: no loop of the kernel runs over every sphere or every tree.
func TestKernelLoopsNoLongerTryEverySphere(t *testing.T) {
	source, err := os.ReadFile("trace.metal")
	if err != nil {
		t.Fatal(err)
	}
	for _, loop := range []string{"header->counts.x; index++",
		"header->more.w; index++", "header->counts.x; index++)"} {
		if strings.Contains(string(source), loop) {
			t.Errorf("trace.metal still has a loop %q", loop)
		}
	}
}
