package mesh

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// this file's name sorts before any that builds a mesh. game mutants
// runs the suite failfast, in file order, so a change that makes a
// cut, a root or a walk never end is killed here, by name, before a
// mesh's build hangs. renamed to sort later, those changes time out.

// The csv files of this file's fixtures, each row named in its first
// column.
const (
	cutShapesFile  = "testdata/cut-shapes.csv"
	cutBoxedFile   = "testdata/cut-boxed.csv"
	cutSignsFile   = "testdata/cut-signs.csv"
	cutLastBinFile = "testdata/cut-last-bin.csv"
)

// cutRow is so many places along a row, one apart from the first at
// zero.
func cutRow(count int) []float64 {
	var places []float64
	for place := range count {
		places = append(places, float64(place))
	}
	return places
}

// cutPlacesIn is the places of a row, written in the places column of
// a csv row with a space between them.
func cutPlacesIn(test testing.TB, row map[string]string) []float64 {
	test.Helper()
	var places []float64
	for _, word := range strings.Fields(row["places"]) {
		place, err := strconv.ParseFloat(word, floatBits)
		if err != nil {
			test.Fatalf("places: %v", err)
		}
		places = append(places, place)
	}
	return places
}

// cutNextFace is the face over the next three corners to be added to a
// list of corners.
func cutNextFace(corners []maths.Vec) [triangleCorners]int {
	var made [triangleCorners]int
	made[first] = len(corners)
	made[second] = len(corners) + unity
	made[third] = len(corners) + pair
	return made
}

// cutFacesOver is a face over each three corners of a list, in order.
func cutFacesOver(corners []maths.Vec) [][triangleCorners]int {
	var made [][triangleCorners]int
	for start := nought; start < len(corners); start += triangleCorners {
		made = append(made, [triangleCorners]int{start, start + unity,
			start + pair})
	}
	return made
}

// cutBounds is the box of a node as a span.
func cutBounds(at *node) span {
	return span{low: at.bounds[low], high: at.bounds[high]}
}

// strips is faces side by side along an axis, each a right triangle wide and
// one high, its lower left corner at its place along the axis and the rest of
// its place at zero, one place for each face. a face across is flat; its box is
// wide by one.
func strips(axis int, wide float64, places ...float64) ([]maths.Vec,
	[][triangleCorners]int) {
	var corners []maths.Vec
	var faces [][triangleCorners]int
	for _, at := range places {
		origin := [axes]float64{}
		origin[axis] = at
		one, two, three := origin, origin, origin
		two[axis] += wide
		three[(axis+unity)%axes]++
		faces = append(faces, cutNextFace(corners))
		corners = append(corners, vectorOf(one), vectorOf(two),
			vectorOf(three))
	}
	return corners, faces
}

// shape is a node of the tree written out: a leaf as its faces in
// square brackets, a node as its two sides in round ones.
func shape(made tree, at int) string {
	node := &made.nodes[at]
	if node.count > nought {
		return fmt.Sprint(made.run(node))
	}
	return "(" + shape(made, node.first) + " " +
		shape(made, node.first+unity) + ")"
}

// shapeOf is the tree over faces, written out.
func shapeOf(corners []maths.Vec, faces [][triangleCorners]int) string {
	made := grown(corners, spansFaces(faces))
	return shape(made, nought)
}

// spansFaces is faces as the mesh holds them.
func spansFaces(faces [][triangleCorners]int) []face {
	made := make([]face, len(faces))
	for i, each := range faces {
		made[i] = faceOf(each)
	}
	return made
}

// faceOf is a face as given, as the mesh holds it.
func faceOf(each [triangleCorners]int) face {
	return face{int32(each[first]), int32(each[second]),
		int32(each[third])}
}

// growth is a tree about to be cut over faces: each face's box and
// middle, none cut.
func growth(corners []maths.Vec, faces [][triangleCorners]int) (*growing,
	span) {
	made := tree{order: make([]int, len(faces))}
	grow := &growing{tree: &made, spans: make([]span, len(faces)),
		middles: make([][axes]float64, len(faces))}
	covering := emptySpan()
	for i, at := range spansFaces(faces) {
		grow.spans[i] = spanOf(corners, at)
		grow.middles[i] = grow.spans[i].middle()
		covering = covering.joined(grow.spans[i])
		made.order[i] = i
	}
	return grow, covering
}

// a run is cut by its own faces and not its place in the order: faces
// 3 to 5 of a row of six, cut from sixty-two down, are one face and two,
// the two kept as a leaf, and not cut again below sixty-three. this is
// the first test here so that a tree that never stops cutting is
// caught before any other grows one.
func TestARunIsCutByItsFacesAndNotItsPlaceInTheOrder(t *testing.T) {
	length := pickedCount(t, "cut-row-length")
	row := cutRow(length)
	corners, faces := strips(alongX, picked(t, "cut-narrow"), row...)
	grow, _ := growth(corners, faces)
	grow.tree.nodes = make([]node, unity, pickedCount(t, "cut-node-room"))
	from, depth := pickedCount(t, "cut-run-from"),
		pickedCount(t, "cut-run-depth")
	grow.cut(nought, from, length, depth)
	if got := shape(*grow.tree, nought); got != "([3] [4 5])" {
		t.Errorf("faces 3 to 5 are cut %s", got)
	}
}

// the surface area heuristic weighs a cut by the share of the box each
// side's box is, times its faces: three triangles two wide on a row,
// one apart, cut best after the first, at a cost of one for the node,
// and the first's two and the others' three over the whole's four.
func TestTheCheapestCutWeighsEachSideByItsBoxAndFaces(t *testing.T) {
	few := pickedCount(t, "cut-few-faces")
	row := cutRow(few)
	corners, faces := strips(alongX, picked(t, "cut-wide"), row...)
	grow, whole := growth(corners, faces)
	axis, split, cost := grow.best(nought, few, whole)
	if axis != nought || split != nought ||
		cost != picked(t, "cut-weighed-cost") {
		t.Errorf("cut axis %d after bin %d at %v, not 0, 0, 3", axis,
			split, cost)
	}
	corners, faces = strips(alongX, picked(t, "cut-narrow"), row...)
	grow, whole = growth(corners, faces)
	_, _, cost = grow.best(nought, few, whole)
	nearShare, farShare := picked(t, "cut-share-first"),
		picked(t, "cut-share-rest")
	want := unity + (nearShare*unity+farShare*pair)/
		picked(t, "cut-share-whole")
	if !(math.Abs(cost-want) <= picked(t, "cut-tolerance")) {
		t.Errorf("apart, the cost is %v, not %v", cost, want)
	}
}

// the cut is along the axis the faces' middles spread on, and of two
// axes as dear, the first.
func TestTheCutIsAlongTheAxisTheMiddlesSpreadOn(t *testing.T) {
	few := pickedCount(t, "cut-few-faces")
	row := cutRow(few)
	wide := picked(t, "cut-wide")
	for axis := range axes {
		corners, faces := strips(axis, wide, row...)
		grow, whole := growth(corners, faces)
		got, split, _ := grow.best(nought, few, whole)
		if got != axis || split != nought {
			t.Errorf("faces along %d cut along %d after %d", axis, got,
				split)
		}
	}
	corners, faces := diagonal(few)
	grow, whole := growth(corners, faces)
	if axis, _, _ := grow.best(nought, few, whole); axis != nought {
		t.Errorf("faces as dear on x and y were cut along %d", axis)
	}
}

// diagonal is so many right triangles, two wide, each a step on from
// the last along both x and y.
func diagonal(faces int) ([]maths.Vec, [][triangleCorners]int) {
	var corners []maths.Vec
	var made [][triangleCorners]int
	for k := range faces {
		x, y := float64(k), float64(k)
		made = append(made, cutNextFace(corners))
		corners = append(corners, maths.V(x, y, nought),
			maths.V(x+pair, y, nought), maths.V(x, y+pair, nought))
	}
	return corners, made
}

// the tree's shape for a few rows of faces, each worked out from the
// boxes: two or fewer is a leaf; three apart are cut after the first,
// where the heuristic is under three; three overlapping so much that
// cutting costs exactly three are kept; faces given out of order keep
// their order on each side.
func TestTheTreeIsCutWhereTheHeuristicSays(t *testing.T) {
	for _, row := range read(t, cutShapesFile) {
		places := cutPlacesIn(t, row)
		corners, faces := strips(alongX, float(t, row, "wide"), places...)
		if got := shapeOf(corners, faces); got != row["want"] {
			t.Errorf("%s: %s, not %s", row["name"], got, row["want"])
		}
	}
}

// faces whose boxes are one box have no middle to cut at: sixteen are
// kept in one leaf, and seventeen are cut in halves, the first eight
// and the nine after, in the order they came.
func TestFacesInOneBoxAreCutInHalfOnlyPastSixteen(t *testing.T) {
	for _, row := range read(t, cutBoxedFile) {
		count := whole(t, row, "faces")
		corners, each := sharedBox(count)
		if got := shapeOf(corners, each); got != row["want"] {
			t.Errorf("%d faces: %s, not %s", count, got, row["want"])
		}
	}
}

// sharedBox is so many triangles in the unit square, each with a
// corner at its low corner, one at its far edge and one at its top, so
// all of them have its box and none shares an edge with another.
func sharedBox(faces int) ([]maths.Vec, [][triangleCorners]int) {
	var corners []maths.Vec
	var made [][triangleCorners]int
	for k := range faces {
		share := float64(k+unity) / float64(faces+pair)
		made = append(made, cutNextFace(corners))
		corners = append(corners, maths.V(nought, nought, nought),
			maths.V(unity, share, nought), maths.V(share, unity, nought))
	}
	return corners, made
}

// cutVisiting is a walk over a finished tree that meets each node and
// each face once, and stops the test at one it meets twice.
type cutVisiting struct {
	test      *testing.T
	made      tree
	faces     int
	seenNodes []bool
	seenFaces []bool
}

// newCutVisiting is a walk over a tree of a mesh with so many faces,
// that has met nothing.
func newCutVisiting(test *testing.T, made tree, faces int) *cutVisiting {
	return &cutVisiting{test: test, made: made, faces: faces,
		seenNodes: make([]bool, len(made.nodes)),
		seenFaces: make([]bool, faces)}
}

// visit meets the node at an index, and everything below it.
func (visiting *cutVisiting) visit(index int) {
	made := visiting.made
	if index < nought || index >= len(made.nodes) ||
		visiting.seenNodes[index] {
		visiting.test.Fatalf("invalid or repeated node %d", index)
	}
	visiting.seenNodes[index] = true
	at := made.nodes[index]
	if at.count == nought {
		visiting.visit(at.first)
		visiting.visit(at.first + unity)
		return
	}
	visiting.visitRun(index, at)
}

// visitRun meets the faces of a leaf.
func (visiting *cutVisiting) visitRun(index int, at node) {
	made := visiting.made
	if at.count < nought || at.first < nought ||
		at.first > len(made.order)-at.count {
		visiting.test.Fatalf("invalid face run at node %d: %d + %d",
			index, at.first, at.count)
	}
	for _, member := range made.order[at.first : at.first+at.count] {
		if member < nought || member >= visiting.faces ||
			visiting.seenFaces[member] {
			visiting.test.Fatalf("invalid or repeated face %d", member)
		}
		visiting.seenFaces[member] = true
	}
}

// reportMissing says each node and each face that the walk did not meet.
func (visiting *cutVisiting) reportMissing() {
	for index, seen := range visiting.seenNodes {
		if !seen {
			visiting.test.Errorf("node %d is unreachable", index)
		}
	}
	for index, seen := range visiting.seenFaces {
		if !seen {
			visiting.test.Errorf("face %d is missing", index)
		}
	}
}

// A finished hierarchy releases its worst-case construction reservation,
// while retaining every live node and every face exactly once.
func TestTheCompletedTreeReleasesUnusedNodeCapacity(t *testing.T) {
	made, _, faces := ball(t, pickedCount(t, "cut-small-ball"))
	made.ensureTree()
	built := made.tree
	if cap(built.nodes) >= pair*len(faces) {
		t.Errorf("%d faces retain the worst-case reservation of %d nodes",
			len(faces), cap(built.nodes))
	}
	visiting := newCutVisiting(t, built, len(faces))
	visiting.visit(nought)
	visiting.reportMissing()

	wide := picked(t, "cut-wide")
	corners, few := strips(alongX, wide, nought, unity)
	if got := grown(corners, spansFaces(few)); len(got.nodes) != unity {
		t.Errorf("a leaf of two faces made %d nodes", len(got.nodes))
	}
}

// cutWalking is a walk over the boxes of a mesh's tree, that stops the
// test at a box that is not inside the box above it, or at a face that
// is not inside its leaf's box.
type cutWalking struct {
	test    *testing.T
	made    *Mesh
	corners []maths.Vec
	faces   [][triangleCorners]int
}

// walk checks the node at an index against the box above it, and all
// that is below it.
func (walking *cutWalking) walk(index int, outer span) {
	at := &walking.made.tree.nodes[index]
	box := cutBounds(at)
	if box.joined(outer) != outer {
		walking.test.Fatalf("node %d is outside its parent", index)
	}
	if at.count == nought {
		walking.walk(at.first, box)
		walking.walk(at.first+unity, box)
		return
	}
	for _, i := range walking.made.tree.run(at) {
		placed := faceOf(walking.faces[i])
		around := box.joined(spanOf(walking.corners, placed))
		if around != box {
			walking.test.Fatalf("face %d is outside leaf %d", i, index)
		}
	}
}

// every face is in the order once, and every node's box holds its
// children and its faces.
func TestTheTreeHoldsEveryFaceOnceInsideItsBoxes(t *testing.T) {
	made, corners, faces := ball(t, pickedCount(t, "cut-ball"))
	made.ensureTree()
	order := slices.Clone(made.tree.order)
	slices.Sort(order)
	for i := range faces {
		if order[i] != i {
			t.Fatalf("face %d is not in the order once", i)
		}
	}
	walking := &cutWalking{test: t, made: made, corners: corners,
		faces: faces}
	walking.walk(nought, cutBounds(&made.tree.nodes[nought]))
}

// cutDepthWalk is a walk over a tree that notes how deep it goes, and
// whether a leaf of more than a few faces is as deep as a cut goes.
type cutDepthWalk struct {
	made     tree
	furthest int
	crowded  bool
}

// walk goes down the node at an index, which is so deep.
func (walking *cutDepthWalk) walk(index, depth int) {
	at := &walking.made.nodes[index]
	walking.furthest = max(walking.furthest, depth)
	if at.count > nought {
		walking.crowded = walking.crowded ||
			(depth == deepest && at.count > leafFew)
		return
	}
	walking.walk(at.first, depth+unity)
	walking.walk(at.first+unity, depth+unity)
}

// faces on a row each twice as far along as the last are cut one side
// to the far one, the nearer ones all in one bin, so the tree grows
// deep and is stopped at sixty-four down, with the faces there kept in
// a leaf of more than two. so it is with the row the other way, which
// grows down the other side.
func TestADeepTreeIsStoppedAtSixtyFourDown(t *testing.T) {
	narrow := picked(t, "cut-narrow")
	count := pickedCount(t, "cut-row-count")
	for _, row := range read(t, cutSignsFile) {
		sign := float(t, row, "sign")
		var places []float64
		for k := range count {
			places = append(places, sign*math.Ldexp(unity, k))
		}
		corners, faces := strips(alongX, narrow*sign, places...)
		walking := &cutDepthWalk{made: grown(corners, spansFaces(faces))}
		walking.walk(nought, nought)
		if walking.furthest != deepest || !walking.crowded {
			t.Errorf("row of %v: %d deep, crowded at the bottom: %v",
				sign, walking.furthest, walking.crowded)
		}
	}
}

// the cheapest cut can be the last: after the bin before the far one,
// when the nearer two faces' boxes overlap and the far one is a thin
// strip reaching far up. the first two are twenty wide and one high,
// with middles at 0 and 14; the third is a tenth wide and a hundred
// high, at 15. cut there, the two cost 34 times two and the strip 10;
// cut after the first, the strip's box makes the others 2000 a face.
func TestTheCheapestCutCanBeTheLastBin(t *testing.T) {
	corners := vectorsIn(t, cutLastBinFile)
	faces := cutFacesOver(corners)
	grow, whole := growth(corners, faces)
	axis, split, cost := grow.best(nought, len(faces), whole)
	want := unity + picked(t, "cut-last-gain")/picked(t, "cut-last-whole")
	if axis != nought || split != pickedCount(t, "cut-last-bin") ||
		!(math.Abs(cost-want) <= picked(t, "cut-tolerance")) {
		t.Errorf("cut axis %d after bin %d at %v, not 0, 14, %v", axis,
			split, cost, want)
	}
}

// where the middles are one place there is no cut: the cost is endless
// and the axis and bin are the first, and the same for binning on an
// axis they do not spread on.
func TestNoCutIsFoundWhereTheMiddlesAreOnePlace(t *testing.T) {
	shared := pickedCount(t, "cut-shared-faces")
	corners, faces := sharedBox(shared)
	grow, whole := growth(corners, faces)
	axis, split, cost := grow.best(nought, shared, whole)
	if axis != nought || split != nought ||
		!math.IsInf(cost, plusInfinity) {
		t.Errorf("cut axis %d after %d at %v", axis, split, cost)
	}
	for along := range axes {
		middles := grow.middling(nought, shared)
		split, cost = grow.binned(nought, shared, along, middles,
			whole.area())
		if split != nought || !math.IsInf(cost, plusInfinity) {
			t.Errorf("binned on %d: after %d at %v", along, split, cost)
		}
	}
}
