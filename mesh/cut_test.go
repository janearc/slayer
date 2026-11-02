package mesh

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// this file's name sorts before any that builds a mesh. game mutants
// runs the suite failfast, in file order, so a change that makes a
// cut, a root or a walk never end is killed here, by name, before a
// mesh's build hangs. renamed to sort later, those changes time out.

// strips is faces side by side along an axis, each a right triangle wide and
// one high, its lower left corner at its place along the axis and the rest of
// its place at zero, one place for each face. a face across is flat; its box is
// wide by one.
func strips(axis int, wide float64, places ...float64) ([]maths.Vec,
	[][3]int) {
	var corners []maths.Vec
	var faces [][3]int
	for _, at := range places {
		low := [3]float64{}
		low[axis] = at
		one, two, three := low, low, low
		two[axis] += wide
		three[(axis+1)%3]++
		faces = append(faces, [3]int{len(corners), len(corners) + 1,
			len(corners) + 2})
		corners = append(corners, vectorOf(one), vectorOf(two),
			vectorOf(three))
	}
	return corners, faces
}

// shape is a node of the tree written out: a leaf as its faces in
// square brackets, a node as its two sides in round ones.
func shape(made tree, at int) string {
	node := &made.nodes[at]
	if node.count > 0 {
		return fmt.Sprint(made.run(node))
	}
	return "(" + shape(made, node.first) + " " +
		shape(made, node.first+1) + ")"
}

// shapeOf is the tree over faces, written out.
func shapeOf(corners []maths.Vec, faces [][3]int) string {
	made := grown(corners, spansFaces(faces))
	return shape(made, 0)
}

// spansFaces is faces as the mesh holds them.
func spansFaces(faces [][3]int) []face {
	made := make([]face, len(faces))
	for i, each := range faces {
		made[i] = face(each)
	}
	return made
}

// growth is a tree about to be cut over faces: each face's box and
// middle, none cut.
func growth(corners []maths.Vec, faces [][3]int) (*growing, span) {
	made := tree{order: make([]int, len(faces))}
	grow := &growing{tree: &made, spans: make([]span, len(faces)),
		middles: make([][3]float64, len(faces))}
	whole := emptySpan()
	for i, at := range spansFaces(faces) {
		grow.spans[i] = spanOf(corners, at)
		grow.middles[i] = grow.spans[i].middle()
		whole = whole.joined(grow.spans[i])
		made.order[i] = i
	}
	return grow, whole
}

// a run is cut by its own faces and not its place in the order: faces
// 3 to 5 of a row of six, cut from sixty-two down, are one face and two,
// the two kept as a leaf, and not cut again below sixty-three. this is
// the first test here so that a tree that never stops cutting is
// caught before any other grows one.
func TestARunIsCutByItsFacesAndNotItsPlaceInTheOrder(t *testing.T) {
	corners, faces := strips(0, 0.5, 0, 1, 2, 3, 4, 5)
	grow, _ := growth(corners, faces)
	grow.tree.nodes = make([]node, 1, 8)
	grow.cut(0, 3, 6, 62)
	if got := shape(*grow.tree, 0); got != "([3] [4 5])" {
		t.Errorf("faces 3 to 5 are cut %s", got)
	}
}

// the surface area heuristic weighs a cut by the share of the box each
// side's box is, times its faces: three triangles two wide on a row,
// one apart, cut best after the first, at a cost of one for the node,
// and the first's two and the others' three over the whole's four.
func TestTheCheapestCutWeighsEachSideByItsBoxAndFaces(t *testing.T) {
	corners, faces := strips(0, 2, 0, 1, 2)
	grow, whole := growth(corners, faces)
	axis, split, cost := grow.best(0, 3, whole)
	if axis != 0 || split != 0 || cost != 3 {
		t.Errorf("cut axis %d after bin %d at %v, not 0, 0, 3", axis,
			split, cost)
	}
	corners, faces = strips(0, 0.5, 0, 1, 2)
	grow, whole = growth(corners, faces)
	_, _, cost = grow.best(0, 3, whole)
	if want := 1 + (0.25*1+0.75*2)/1.25; math.Abs(cost-want) > 1e-12 {
		t.Errorf("apart, the cost is %v, not %v", cost, want)
	}
}

// the cut is along the axis the faces' middles spread on, and of two
// axes as dear, the first.
func TestTheCutIsAlongTheAxisTheMiddlesSpreadOn(t *testing.T) {
	for axis := range 3 {
		corners, faces := strips(axis, 2, 0, 1, 2)
		grow, whole := growth(corners, faces)
		got, split, _ := grow.best(0, 3, whole)
		if got != axis || split != 0 {
			t.Errorf("faces along %d cut along %d after %d", axis, got,
				split)
		}
	}
	corners, faces := diagonal(3)
	grow, whole := growth(corners, faces)
	if axis, _, _ := grow.best(0, 3, whole); axis != 0 {
		t.Errorf("faces as dear on x and y were cut along %d", axis)
	}
}

// diagonal is so many right triangles, two wide, each a step on from
// the last along both x and y.
func diagonal(faces int) ([]maths.Vec, [][3]int) {
	var corners []maths.Vec
	var made [][3]int
	for k := range faces {
		x, y := float64(k), float64(k)
		made = append(made, [3]int{len(corners), len(corners) + 1,
			len(corners) + 2})
		corners = append(corners, maths.V(x, y, 0), maths.V(x+2, y, 0),
			maths.V(x, y+2, 0))
	}
	return corners, made
}

// the tree's shape for a few rows of faces, each worked out from the
// boxes: two or fewer is a leaf; three apart are cut after the first,
// where the heuristic is under three; three overlapping so much that
// cutting costs exactly three are kept; faces given out of order keep
// their order on each side.
func TestTheTreeIsCutWhereTheHeuristicSays(t *testing.T) {
	for name, each := range map[string]struct {
		wide   float64
		places []float64
		want   string
	}{
		"one":          {2, []float64{0}, "[0]"},
		"two":          {2, []float64{0, 1}, "[0 1]"},
		"three apart":  {0.5, []float64{0, 1, 2}, "([0] [1 2])"},
		"three tied":   {2, []float64{0, 1, 2}, "[0 1 2]"},
		"three nearly": {1, []float64{0, 0.01, 0.02}, "[0 1 2]"},
		"out of order": {0.5, []float64{2, 0, 1}, "([1] [0 2])"},
		"a straggler": {2, []float64{0, 1, 2, 3, 100},
			"(([0 1] [2 3]) [4])"},
	} {
		corners, faces := strips(0, each.wide, each.places...)
		if got := shapeOf(corners, faces); got != each.want {
			t.Errorf("%s: %s, not %s", name, got, each.want)
		}
	}
}

// faces whose boxes are one box have no middle to cut at: sixteen are
// kept in one leaf, and seventeen are cut in halves, the first eight
// and the nine after, in the order they came.
func TestFacesInOneBoxAreCutInHalfOnlyPastSixteen(t *testing.T) {
	for faces, want := range map[int]string{
		16: "[0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15]",
		17: "([0 1 2 3 4 5 6 7] [8 9 10 11 12 13 14 15 16])",
	} {
		corners, each := sharedBox(faces)
		if got := shapeOf(corners, each); got != want {
			t.Errorf("%d faces: %s, not %s", faces, got, want)
		}
	}
}

// sharedBox is so many triangles in the unit square, each with a
// corner at its low corner, one at its far edge and one at its top, so
// all of them have its box and none shares an edge with another.
func sharedBox(faces int) ([]maths.Vec, [][3]int) {
	var corners []maths.Vec
	var made [][3]int
	for k := range faces {
		share := float64(k+1) / float64(faces+2)
		made = append(made, [3]int{len(corners), len(corners) + 1,
			len(corners) + 2})
		corners = append(corners, maths.V(0, 0, 0), maths.V(1, share, 0),
			maths.V(share, 1, 0))
	}
	return corners, made
}

// a mesh allocates its nodes once, at twice its faces, as a binary tree
// of leaves cannot have more, and the root is the only node when the
// faces make one leaf.
func TestTheNodesAreAllocatedOnceAtTwiceTheFaces(t *testing.T) {
	made, _, faces := ball(t, 2)
	if cap(made.tree.nodes) != 2*len(faces) {
		t.Errorf("%d faces have room for %d nodes", len(faces),
			cap(made.tree.nodes))
	}
	corners, few := strips(0, 2, 0, 1)
	if got := grown(corners, spansFaces(few)); len(got.nodes) != 1 {
		t.Errorf("a leaf of two faces made %d nodes", len(got.nodes))
	}
}

// every face is in the order once, and every node's box holds its
// children and its faces.
func TestTheTreeHoldsEveryFaceOnceInsideItsBoxes(t *testing.T) {
	made, corners, faces := ball(t, 3)
	order := slices.Clone(made.tree.order)
	slices.Sort(order)
	for i := range faces {
		if order[i] != i {
			t.Fatalf("face %d is not in the order once", i)
		}
	}
	var walk func(at int, outer span)
	walk = func(at int, outer span) {
		node := &made.tree.nodes[at]
		box := span{low: node.bounds[0], high: node.bounds[1]}
		if box.joined(outer) != outer {
			t.Fatalf("node %d is outside its parent", at)
		}
		if node.count == 0 {
			walk(node.first, box)
			walk(node.first+1, box)
			return
		}
		for _, i := range made.tree.run(node) {
			if box.joined(spanOf(corners, face(faces[i]))) != box {
				t.Fatalf("face %d is outside leaf %d", i, at)
			}
		}
	}
	root := &made.tree.nodes[0]
	walk(0, span{low: root.bounds[0], high: root.bounds[1]})
}

// faces on a row each twice as far along as the last are cut one side
// to the far one, the nearer ones all in one bin, so the tree grows
// deep and is stopped at sixty-four down, with the faces there kept in
// a leaf of more than two. so it is with the row the other way, which
// grows down the other side.
func TestADeepTreeIsStoppedAtSixtyFourDown(t *testing.T) {
	for _, sign := range []float64{1, -1} {
		var places []float64
		for k := range 300 {
			places = append(places, sign*math.Ldexp(1, k))
		}
		corners, faces := strips(0, 0.5*sign, places...)
		made := grown(corners, spansFaces(faces))
		deepest, crowded := 0, false
		var walk func(at, depth int)
		walk = func(at, depth int) {
			node := &made.nodes[at]
			deepest = max(deepest, depth)
			if node.count > 0 {
				crowded = crowded || (depth == 64 && node.count > 2)
				return
			}
			walk(node.first, depth+1)
			walk(node.first+1, depth+1)
		}
		walk(0, 0)
		if deepest != 64 || !crowded {
			t.Errorf("row of %v: %d deep, crowded at the bottom: %v",
				sign, deepest, crowded)
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
	corners := []maths.Vec{
		maths.V(-10, 0, 0), maths.V(10, 0, 0), maths.V(-10, 1, 0),
		maths.V(4, 0, 0), maths.V(24, 0, 0), maths.V(4, 1, 0),
		maths.V(15, 0, 0), maths.V(15.1, 0, 0), maths.V(15, 100, 0)}
	faces := [][3]int{{0, 1, 2}, {3, 4, 5}, {6, 7, 8}}
	grow, whole := growth(corners, faces)
	axis, split, cost := grow.best(0, 3, whole)
	if want := 1 + 78.0/3400; axis != 0 || split != 14 ||
		math.Abs(cost-want) > 1e-12 {
		t.Errorf("cut axis %d after bin %d at %v, not 0, 14, %v", axis,
			split, cost, want)
	}
}

// where the middles are one place there is no cut: the cost is endless
// and the axis and bin are the first, and the same for binning on an
// axis they do not spread on.
func TestNoCutIsFoundWhereTheMiddlesAreOnePlace(t *testing.T) {
	corners, faces := sharedBox(4)
	grow, whole := growth(corners, faces)
	axis, split, cost := grow.best(0, 4, whole)
	if axis != 0 || split != 0 || !math.IsInf(cost, 1) {
		t.Errorf("cut axis %d after %d at %v", axis, split, cost)
	}
	for along := range 3 {
		middles := grow.middling(0, 4)
		split, cost = grow.binned(0, 4, along, middles, whole.area())
		if split != 0 || !math.IsInf(cost, 1) {
			t.Errorf("binned on %d: after %d at %v", along, split, cost)
		}
	}
}
