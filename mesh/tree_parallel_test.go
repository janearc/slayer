package mesh

import (
	"math"
	"slices"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the fixture's size: cells along a side of the grid, each two faces, so
// that a cut near the root leaves sides of more than parallelFaces.
const gridCells = 100

// the fixture's relief, a height that folds the grid so its cuts differ
// from a plane's: the wave lengths along each side, and its height.
const (
	reliefLengthX = 0.7
	reliefLengthY = 1.3
	reliefHeight  = 3.0
)

// gridPoints is the corners of a grid of cells by cells squares, each a
// column of the grid, raised by a fixed relief.
func gridPoints(cells int) []maths.Vec {
	var points []maths.Vec
	for row := 0; row <= cells; row++ {
		for column := 0; column <= cells; column++ {
			across, along := float64(column), float64(row)
			height := reliefHeight * math.Sin(across*reliefLengthX) *
				math.Cos(along*reliefLengthY)
			points = append(points, maths.V(across, along, height))
		}
	}
	return points
}

// gridFaces is the faces of that grid, two to a square, all wound alike.
func gridFaces(cells int) [][triangleCorners]int {
	var faces [][triangleCorners]int
	stride := cells + 1
	for row := 0; row < cells; row++ {
		for column := 0; column < cells; column++ {
			low := row*stride + column
			faces = append(faces,
				[triangleCorners]int{low, low + 1, low + stride},
				[triangleCorners]int{low + 1, low + stride + 1, low + stride})
		}
	}
	return faces
}

// sameTrees fails the test unless two trees have the same nodes, bit for
// bit, in the same places, and the same order of faces.
func sameTrees(t *testing.T, name string, want, got tree) {
	t.Helper()
	if !slices.Equal(want.nodes, got.nodes) {
		t.Fatalf("%s: the nodes differ: %d against %d", name,
			len(want.nodes), len(got.nodes))
	}
	if !slices.Equal(want.order, got.order) {
		t.Fatalf("%s: the order of the faces differs", name)
	}
}

func TestATreeGrownByManyIsTheTreeGrownByOne(t *testing.T) {
	points := gridPoints(gridCells)
	faces := spansFaces(gridFaces(gridCells))
	serial := grown(points, faces)
	for _, places := range []int{1, 2, 3, 8, 64} {
		spare := make(chan struct{}, places)
		sameTrees(t, "places", serial, grownBy(points, faces, spare))
		if len(spare) != 0 {
			t.Fatalf("%d places left held after the tree was grown",
				len(spare))
		}
	}
}

func TestEveryBranchOfAParallelTreeNamesItsChildrenAfterItself(t *testing.T) {
	points := gridPoints(gridCells)
	faces := spansFaces(gridFaces(gridCells))
	made := grownBy(points, faces, make(chan struct{}, 8))
	reached := make([]int, len(made.nodes))
	for at := range made.nodes {
		each := &made.nodes[at]
		if each.count == nought {
			if each.first <= at || each.first+unity >= len(made.nodes) {
				t.Fatalf("node %d names children at %d", at, each.first)
			}
			reached[each.first]++
			reached[each.first+unity]++
		}
	}
	for at := unity; at < len(reached); at++ {
		if reached[at] != unity {
			t.Fatalf("node %d is the child of %d nodes", at, reached[at])
		}
	}
}

func TestNoSideIsLentWithoutAPlaceOrBelowTheSize(t *testing.T) {
	alone := growing{}
	if alone.lends(parallelFaces * pair) {
		t.Fatal("a tree with no spare lent a side")
	}
	full := growing{spare: make(chan struct{}, unity)}
	full.spare <- struct{}{}
	if full.lends(parallelFaces * pair) {
		t.Fatal("a side was lent with no place free")
	}
	roomy := growing{spare: make(chan struct{}, unity)}
	if roomy.lends(parallelFaces - unity) {
		t.Fatal("a side below the size was lent")
	}
	if len(roomy.spare) != 0 {
		t.Fatal("a refused side took a place")
	}
	if !roomy.lends(parallelFaces) || len(roomy.spare) != unity {
		t.Fatal("a side of the size was not lent a place")
	}
}

func TestGrowersGrowEachMeshOnceToTheTreeOneAskWould(t *testing.T) {
	var meshes []*Mesh
	for range 3 {
		made, err := New(gridPoints(gridCells), gridFaces(gridCells), nil,
			false)
		if err != nil {
			t.Fatal(err)
		}
		meshes = append(meshes, made)
	}
	want := grown(meshes[nought].points, meshes[nought].faces)
	growers := NewGrowers(4)
	var asked sync.WaitGroup
	for _, made := range meshes {
		for range 2 {
			asked.Add(1)
			go func() {
				defer asked.Done()
				growers.Grow(made)
			}()
		}
	}
	asked.Wait()
	for _, made := range meshes {
		sameTrees(t, "growers", want, made.tree)
		if made.TreeSize() != len(want.nodes) {
			t.Fatalf("TreeSize is %d, not %d", made.TreeSize(),
				len(want.nodes))
		}
	}
	if len(growers.places) != 0 {
		t.Fatalf("%d places left held", len(growers.places))
	}
}

func TestGrowersOfNoWorkersStillGrowWithOne(t *testing.T) {
	made, err := New(gridPoints(unity), gridFaces(unity), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	NewGrowers(nought).Grow(made)
	if made.TreeSize() < unity {
		t.Fatal("the tree was not grown")
	}
}
