package mesh

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files of this file's fixtures, each row named in its first
// column.
const (
	walkBallRadiiFile       = "testdata/walk-ball-radii.csv"
	walkCostRadiiFile       = "testdata/walk-cost-radii.csv"
	walkTwoNearCornersFile  = "testdata/walk-two-near-corners.csv"
	walkTwoNearFacesFile    = "testdata/walk-two-near-faces.csv"
	walkTwoNearPointFile    = "testdata/walk-two-near-point.csv"
	walkBeatenFile          = "testdata/walk-beaten.csv"
	walkSquareFacesFile     = "testdata/walk-square-faces.csv"
	walkOverSquareFile      = "testdata/walk-over-square.csv"
	walkStragglerPlacesFile = "testdata/walk-straggler-places.csv"
	walkStragglerNearFile   = "testdata/walk-straggler-nearest.csv"
	walkBoxBoundsFile       = "testdata/walk-box-bounds.csv"
	walkBoxPointsFile       = "testdata/walk-box-points.csv"
)

// walkNoFace is the face of nothing found yet.
const walkNoFace = -1 // defined: faces are counted from zero, so one before it

// walkTally counts the nodes entered and faces tried that it is told of.
type walkTally struct{ nodes, faces int }

// note counts a piece of work of the kind given.
func (tally *walkTally) note(kind int) {
	if kind == entered {
		tally.nodes++
	} else {
		tally.faces++
	}
}

// walkStopCounting stops the package's counting of work.
func walkStopCounting() { counted = nil }

// work counts the nodes entered and faces tried by whatever action does.
func work(action func()) (nodes, faces int) {
	tally := &walkTally{}
	counted = tally.note
	defer walkStopCounting()
	action()
	return tally.nodes, tally.faces
}

// walkNearest is a mesh and a point it is asked the nearest face to.
type walkNearest struct {
	made  *Mesh
	point maths.Vec
}

// ask finds the nearest face and keeps nothing of it.
func (asking walkNearest) ask() { asking.made.nearest(asking.point) }

// brute is the nearest face by asking every face, the first of two as
// near: the face, its weights and how far, squared.
func brute(made *Mesh, point maths.Vec) (int, weights, float64) {
	best, at, squared := walkNoFace, weights{}, math.Inf(plusInfinity)
	for i := range made.faces {
		on := made.closest(i, point)
		gap := made.pointAt(i, on).Sub(point)
		if got := gap.Dot(gap); got < squared {
			best, at, squared = i, on, got
		}
	}
	return best, at, squared
}

// walkRandom is a random source seeded by the two numbers of the names
// given.
func walkRandom(test testing.TB, firstName, secondName string) *rand.Rand {
	test.Helper()
	seedFirst := uint64(picked(test, firstName))
	seedSecond := uint64(picked(test, secondName))
	return rand.New(rand.NewPCG(seedFirst, seedSecond))
}

// walkColumn is the numbers of a column of a csv file, a row each.
func walkColumn(test testing.TB, file, column string) []float64 {
	test.Helper()
	var numbers []float64
	for _, row := range read(test, file) {
		numbers = append(numbers, float(test, row, column))
	}
	return numbers
}

// shell is points on spheres round the origin, the radii given, from a
// fixed sequence.
func shell(test testing.TB, count int, radii ...float64) []maths.Vec {
	test.Helper()
	random := walkRandom(test, "walk-sphere-seed-first",
		"walk-sphere-seed-second")
	var made []maths.Vec
	for i := range count {
		direction := maths.V(random.NormFloat64(), random.NormFloat64(),
			random.NormFloat64()).Unit()
		radius := radii[i%len(radii)]
		made = append(made, direction.Scale(radius))
	}
	return made
}

// walkRowPlaces is the places along a row of faces, each a power of two
// in two runs, the second run stood farther along.
func walkRowPlaces(test testing.TB) []float64 {
	test.Helper()
	doublings := pickedCount(test, "walk-row-doublings")
	extra := picked(test, "walk-row-growth")
	var places []float64
	for k := range pickedCount(test, "walk-row-faces") {
		scale := math.Ldexp(unity, k%doublings)
		places = append(places, scale*(unity+extra*float64(k/doublings)))
	}
	return places
}

// walkRowAgrees is that the tree finds the nearest face of a row of
// faces of every size, as asking every face does, for random points.
func walkRowAgrees(t *testing.T) {
	width := picked(t, "cut-narrow")
	corners, faces := strips(nought, width, walkRowPlaces(t)...)
	row := mustMesh(t, corners, faces, nil, false)
	random := walkRandom(t, "walk-row-seed-first", "walk-row-seed-second")
	reach := math.Ldexp(unity, pickedCount(t, "walk-row-reach-doublings"))
	spread := picked(t, "walk-row-spread")
	for range pickedCount(t, "walk-points") {
		point := maths.V(random.Float64()*reach, random.NormFloat64()*spread,
			random.NormFloat64()*spread)
		wantFace, wantAt, wantSquared := brute(row, point)
		face, at, squared := row.nearest(point)
		if face != wantFace || at != wantAt || squared != wantSquared {
			t.Fatalf("at %v the tree gives %d, asking all %d", point,
				face, wantFace)
		}
	}
}

// the nearest face by the tree is the nearest by asking every face,
// for points inside the ball of faces, on it and outside, and for faces
// of every size in a row.
func TestTheTreeFindsTheNearestFaceAskingEveryFace(t *testing.T) {
	subdivisions := pickedCount(t, "walk-ball-subdivisions")
	made, _, _ := ball(t, subdivisions)
	count := pickedCount(t, "walk-points")
	radii := walkColumn(t, walkBallRadiiFile, "radius")
	for _, point := range shell(t, count, radii...) {
		wantFace, wantAt, wantSquared := brute(made, point)
		face, at, squared := made.nearest(point)
		if face != wantFace || at != wantAt || squared != wantSquared {
			t.Fatalf("at %v the tree gives %d %v %v, asking all %d %v %v",
				point, face, at, squared, wantFace, wantAt, wantSquared)
		}
	}
	walkRowAgrees(t)
}

// walkWeights is the weights in the columns weight-first, weight-second
// and weight-third of a csv row.
func walkWeights(test testing.TB, row map[string]string) weights {
	test.Helper()
	return weights{float(test, row, "weight-first"),
		float(test, row, "weight-second"), float(test, row, "weight-third")}
}

// of two faces as near, the first is the nearest, whichever the tree
// reaches first: faces 1 and 2 are in the tree's left side and face 0,
// the first, alone in its right, and a point halfway between 0 and 1
// is a unit from both.
func TestOfTwoAsNearTheTreeFindsTheFirstFace(t *testing.T) {
	corners := vectorsIn(t, walkTwoNearCornersFile)
	faces := facesInFile(t, walkTwoNearFacesFile)
	made := mustMesh(t, corners, faces, nil, false)
	made.ensureTree()
	if got := shape(made.tree, nought); got != "([1 2] [0])" {
		t.Fatalf("the tree is %s", got)
	}
	row := read(t, walkTwoNearPointFile)[nought]
	want := walkWeights(t, row)
	face, at, squared := made.nearest(nearVectorAt(t, row, ""))
	if face != nought || squared != unity || at != want {
		t.Errorf("nearest %d %v %v, not 0 %v 1", face, at, squared, want)
	}
}

// a tie goes to the earlier face, and a nearer face beats a later one:
// the same distance from a later face does not, from an earlier does.
func TestAFaceBeatsTheBestOnlyIfNearerOrAsNearAndEarlier(t *testing.T) {
	bestFace := pickedCount(t, "walk-best-face")
	bestSquared := picked(t, "walk-best-squared")
	for _, row := range read(t, walkBeatenFile) {
		squared, face := float(t, row, "squared"), whole(t, row, "face")
		want := whole(t, row, "want") == unity
		best := found{face: bestFace, squared: bestSquared}
		if got := best.beaten(squared, face); got != want {
			t.Errorf("%v away, face %d against face 3 at 1: %v", squared,
				face, got)
		}
		met := Met{face: bestFace, along: bestSquared}
		if got := met.beaten(squared, face); got != want {
			t.Errorf("%v along, face %d against face 3 at 1: %v", squared,
				face, got)
		}
	}
}

// walkStragglers is that a row whose root is a leaf of two and whose
// last face is alone in its leaf is searched whole.
func walkStragglers(t *testing.T) {
	places := walkColumn(t, walkStragglerPlacesFile, "place")
	corners, faces := strips(nought, picked(t, "cut-wide"), places...)
	row := mustMesh(t, corners, faces, nil, false)
	row.ensureTree()
	if got := shape(row.tree, nought); got != "(([0 1] [2 3]) [4])" {
		t.Fatalf("the tree is %s", got)
	}
	for _, each := range read(t, walkStragglerNearFile) {
		point, want := nearVectorAt(t, each, ""), whole(t, each, "face")
		if face, _, _ := row.nearest(point); face != want {
			t.Errorf("at %v the nearest is %d, not %d", point, face, want)
		}
	}
}

// a mesh of one leaf, or of a leaf of one face, is searched whole: the
// root is a leaf of two, and the straggler is alone in its leaf.
func TestLeavesOfOneAndTwoAreSearched(t *testing.T) {
	height := picked(t, "walk-square-height")
	faces := facesInFile(t, walkSquareFacesFile)
	two := mustMesh(t, aSquare(height), faces, nil, false)
	over := vectorsIn(t, walkOverSquareFile)[nought]
	if got := two.Distance(over); got != picked(t, "walk-over-distance") {
		t.Errorf("two faces, 2 away: %v", got)
	}
	walkStragglers(t)
}

// walkTriple is the three parts of a point in the columns x, y and z of a
// csv row.
func walkTriple(test testing.TB, row map[string]string) [axes]float64 {
	test.Helper()
	var triple [axes]float64
	triple[alongX] = float(test, row, "x")
	triple[alongY] = float(test, row, "y")
	triple[alongZ] = float(test, row, "z")
	return triple
}

// walkBox is a node whose box has the two ends in the box bounds file.
func walkBox(test testing.TB) *node {
	test.Helper()
	at := &node{}
	for _, row := range read(test, walkBoxBoundsFile) {
		at.bounds[whole(test, row, "end")] = walkTriple(test, row)
	}
	return at
}

// the distance to a box is zero inside it and otherwise how far past the near
// side on each axis, by the square: on each axis the farther of the two gaps,
// or zero, and the squares added.
func TestAPointIsAwayFromABoxByTheGapOnEachAxis(t *testing.T) {
	at := walkBox(t)
	for _, each := range read(t, walkBoxPointsFile) {
		point, want := walkTriple(t, each), float(t, each, "want")
		if got := awayFrom(at, point); got != want {
			t.Errorf("%v is %v away, squared, not %v", point, got, want)
		}
	}
}

// walkBounded is that the most work for a point, and the work for all of
// them, are within the bounds.
func walkBounded(t *testing.T, most, tried, total, all int) {
	if most > pickedCount(t, "walk-most-nodes") ||
		tried > pickedCount(t, "walk-most-faces") {
		t.Errorf("a point cost %d nodes and %d faces", most, tried)
	}
	if total > pickedCount(t, "walk-total-nodes") ||
		all > pickedCount(t, "walk-total-faces") {
		t.Errorf("a thousand points cost %d nodes and %d faces", total,
			all)
	}
}

// the walk for the nearest face enters few nodes and tries few faces:
// a thousand points on and round a ball of 1280 faces and 1469 nodes,
// asking every face would try 1280 each. measured, the walk enters at
// most 102 nodes and tries at most 57 faces for a point, and 35955
// nodes and 16405 faces for the thousand. the bounds are those and a
// twentieth.
func TestTheNearestWalkEntersFewNodesAndTriesFewFaces(t *testing.T) {
	subdivisions := pickedCount(t, "walk-ball-subdivisions")
	made, _, _ := ball(t, subdivisions)
	var most, tried, total, all int
	one := maths.V(unity, nought, nought)
	nodes, faces := work(walkNearest{made, one}.ask)
	if nodes == nought || faces == nought {
		t.Errorf("a point entered %d nodes and tried %d faces", nodes, faces)
	}
	count := pickedCount(t, "walk-points")
	radii := walkColumn(t, walkCostRadiiFile, "radius")
	for _, point := range shell(t, count, radii...) {
		nodes, faces = work(walkNearest{made, point}.ask)
		most, tried = max(most, nodes), max(tried, faces)
		total += nodes
		all += faces
	}
	walkBounded(t, most, tried, total, all)
}
