package mesh

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// work counts the nodes entered and faces tried by whatever f does.
func work(f func()) (nodes, faces int) {
	counted = func(kind int) {
		if kind == entered {
			nodes++
		} else {
			faces++
		}
	}
	defer func() { counted = nil }()
	f()
	return nodes, faces
}

// brute is the nearest face by asking every face, the first of two as
// near: the face, its weights and how far, squared.
func brute(made *Mesh, point maths.Vec) (int, weights, float64) {
	best, at, squared := -1, weights{}, math.Inf(1)
	for i := range made.faces {
		on := made.closest(i, point)
		gap := made.pointAt(i, on).Sub(point)
		if got := gap.Dot(gap); got < squared {
			best, at, squared = i, on, got
		}
	}
	return best, at, squared
}

// shell is points on spheres round the origin, the radii given, from a
// fixed sequence.
func shell(n int, radii ...float64) []maths.Vec {
	random := rand.New(rand.NewPCG(7, 8))
	var made []maths.Vec
	for i := range n {
		direction := maths.V(random.NormFloat64(), random.NormFloat64(),
			random.NormFloat64()).Unit()
		made = append(made, direction.Scale(radii[i%len(radii)]))
	}
	return made
}

// the nearest face by the tree is the nearest by asking every face,
// for points inside the ball of faces, on it and outside, and for faces
// of every size in a row.
func TestTheTreeFindsTheNearestFaceAskingEveryFace(t *testing.T) {
	made, _, _ := ball(t, 3)
	for _, point := range shell(1000, 0.3, 0.9, 1, 1.05, 2, 3) {
		wantFace, wantAt, wantSquared := brute(made, point)
		face, at, squared := made.nearest(point)
		if face != wantFace || at != wantAt || squared != wantSquared {
			t.Fatalf("at %v the tree gives %d %v %v, asking all %d %v %v",
				point, face, at, squared, wantFace, wantAt, wantSquared)
		}
	}
	var places []float64
	for k := range 40 {
		places = append(places, math.Ldexp(1, k%20)*(1+0.5*float64(k/20)))
	}
	corners, faces := strips(0, 0.5, places...)
	row := mustMesh(t, corners, faces, nil, false)
	random := rand.New(rand.NewPCG(9, 10))
	for range 1000 {
		point := maths.V(random.Float64()*math.Ldexp(1, 21),
			random.NormFloat64()*5, random.NormFloat64()*5)
		wantFace, wantAt, wantSquared := brute(row, point)
		face, at, squared := row.nearest(point)
		if face != wantFace || at != wantAt || squared != wantSquared {
			t.Fatalf("at %v the tree gives %d, asking all %d", point,
				face, wantFace)
		}
	}
}

// of two faces as near, the first is the nearest, whichever the tree
// reaches first: faces 1 and 2 are in the tree's left side and face 0,
// the first, alone in its right, and a point halfway between 0 and 1
// is a unit from both.
func TestOfTwoAsNearTheTreeFindsTheFirstFace(t *testing.T) {
	corners := []maths.Vec{
		maths.V(1, 0, 0), maths.V(1, 1, 0), maths.V(1, 0, 1),
		maths.V(-1, 0, 0), maths.V(-1, 0, 1), maths.V(-1, 1, 0),
		maths.V(-1.5, 0, 0), maths.V(-1.5, 0, 1), maths.V(-1.5, 1, 0)}
	made := mustMesh(t, corners, [][3]int{{0, 1, 2}, {3, 4, 5}, {6, 7, 8}},
		nil, false)
	if got := shape(made.tree, 0); got != "([1 2] [0])" {
		t.Fatalf("the tree is %s", got)
	}
	face, at, squared := made.nearest(maths.V(0, 0.25, 0.25))
	if face != 0 || squared != 1 || at != (weights{0.5, 0.25, 0.25}) {
		t.Errorf("nearest %d %v %v, not 0 %v 1", face, at, squared,
			weights{0.5, 0.25, 0.25})
	}
}

// a tie goes to the earlier face, and a nearer face beats a later one:
// the same distance from a later face does not, from an earlier does.
func TestAFaceBeatsTheBestOnlyIfNearerOrAsNearAndEarlier(t *testing.T) {
	for _, each := range []struct {
		squared float64
		face    int
		want    bool
	}{{0.5, 9, true}, {1, 2, true}, {1, 3, false}, {1, 4, false},
		{2, 0, false}} {
		if got := (found{face: 3, squared: 1}).beaten(each.squared,
			each.face); got != each.want {
			t.Errorf("%v away, face %d against face 3 at 1: %v",
				each.squared, each.face, got)
		}
		if got := (Met{face: 3, along: 1}).beaten(each.squared,
			each.face); got != each.want {
			t.Errorf("%v along, face %d against face 3 at 1: %v",
				each.squared, each.face, got)
		}
	}
}

// a mesh of one leaf, or of a leaf of one face, is searched whole: the
// root is a leaf of two, and the straggler is alone in its leaf.
func TestLeavesOfOneAndTwoAreSearched(t *testing.T) {
	two := mustMesh(t, aSquare(2), [][3]int{{0, 1, 2}, {0, 2, 3}}, nil,
		false)
	if got := two.Distance(maths.V(0.25, 0.5, 4)); got != 2 {
		t.Errorf("two faces, 2 away: %v", got)
	}
	corners, faces := strips(0, 2, 0, 1, 2, 3, 100)
	row := mustMesh(t, corners, faces, nil, false)
	if got := shape(row.tree, 0); got != "(([0 1] [2 3]) [4])" {
		t.Fatalf("the tree is %s", got)
	}
	for _, each := range []struct {
		point maths.Vec
		face  int
	}{{maths.V(99, 0.25, 3), 4}, {maths.V(100.5, 0.5, 0.25), 4},
		{maths.V(2, 0.25, 1), 1}, {maths.V(4.5, 0.1, 1), 3},
		{maths.V(0.5, 0.25, -1), 0}} {
		if face, _, _ := row.nearest(each.point); face != each.face {
			t.Errorf("at %v the nearest is %d, not %d", each.point, face,
				each.face)
		}
	}
}

// the distance to a box is zero inside it and otherwise how far past the near
// side on each axis, by the square: on each axis the farther of the two gaps,
// or zero, and the squares added.
func TestAPointIsAwayFromABoxByTheGapOnEachAxis(t *testing.T) {
	at := &node{bounds: [2][3]float64{{1, 2, 3}, {4, 6, 9}}}
	for _, each := range []struct {
		point [3]float64
		want  float64
	}{
		{[3]float64{2, 3, 4}, 0},
		{[3]float64{1, 6, 9}, 0},
		{[3]float64{0, 3, 4}, 1},
		{[3]float64{6, 3, 4}, 4},
		{[3]float64{2, 0, 4}, 4},
		{[3]float64{2, 9, 4}, 9},
		{[3]float64{2, 3, 1}, 4},
		{[3]float64{2, 3, 12}, 9},
		{[3]float64{-2, 10, 0}, 9 + 16 + 9},
		{[3]float64{5, 1, 11}, 1 + 1 + 4},
	} {
		if got := awayFrom(at, each.point); got != each.want {
			t.Errorf("%v is %v away, squared, not %v", each.point, got,
				each.want)
		}
	}
}

// the walk for the nearest face enters few nodes and tries few faces:
// a thousand points on and round a ball of 1280 faces and 1469 nodes,
// asking every face would try 1280 each. measured, the walk enters at
// most 102 nodes and tries at most 57 faces for a point, and 35955
// nodes and 16405 faces for the thousand. the bounds are those and a
// twentieth.
func TestTheNearestWalkEntersFewNodesAndTriesFewFaces(t *testing.T) {
	made, _, _ := ball(t, 3)
	var most, tried, total, all int
	nodes, faces := work(func() { made.nearest(maths.V(1, 0, 0)) })
	if nodes == 0 || faces == 0 {
		t.Errorf("a point entered %d nodes and tried %d faces", nodes, faces)
	}
	for _, point := range shell(1000, 0.97, 1, 1.03, 1.5, 2, 3) {
		nodes, faces = work(func() { made.nearest(point) })
		most, tried = max(most, nodes), max(tried, faces)
		total += nodes
		all += faces
	}
	if most > 107 || tried > 60 {
		t.Errorf("a point cost %d nodes and %d faces", most, tried)
	}
	if total > 37753 || all > 17225 {
		t.Errorf("a thousand points cost %d nodes and %d faces", total,
			all)
	}
}
