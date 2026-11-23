package strategies

import (
	"errors"
	"math"
	"math/rand"
	"testing"
)

// testStack is the walker's stack the gpu gives, as its treeStack has it.
const testStack = 64

// unitBox is a box of side two around a place.
func unitBox(x, y, z float64) TreeBox {
	return TreeBox{Low: [3]float64{x - 1, y - 1, z - 1},
		High: [3]float64{x + 1, y + 1, z + 1}}
}

// scatter is count boxes in a cube, from a seed.
func scatter(count int, seed int64) []TreeBox {
	source := rand.New(rand.NewSource(seed))
	boxes := make([]TreeBox, count)
	for index := range boxes {
		boxes[index] = unitBox(source.Float64()*1000, source.Float64()*1000,
			source.Float64()*1000)
	}
	return boxes
}

// depthOf is the depth of the deepest leaf below a node.
func depthOf(tree BoxTree, at int) int {
	node := tree.Nodes[at]
	if node.IsLeaf() {
		return 0
	}
	first := int(node.First)
	return 1 + max(depthOf(tree, first),
		depthOf(tree, first+1))
}

// checkTree fails the test unless the tree holds every box once, in
// branches whose boxes hold their children's, with 2n-1 nodes.
func checkTree(t *testing.T, tree BoxTree, boxes []TreeBox) {
	t.Helper()
	if len(tree.Nodes) != 2*len(boxes)-1 {
		t.Fatalf("%d nodes for %d boxes", len(tree.Nodes), len(boxes))
	}
	seen := make([]int, len(boxes))
	var visit func(at int)
	visit = func(at int) {
		node := tree.Nodes[at]
		if node.IsLeaf() {
			seen[node.First]++
			box := boxes[node.First]
			if node.Low != box.Low || node.High != box.High {
				t.Fatalf("leaf %d is not its box", at)
			}
			return
		}
		for _, child := range []int{int(node.First), int(node.First) + 1} {
			kid := tree.Nodes[child]
			for axis := range 3 {
				if kid.Low[axis] < node.Low[axis] ||
					kid.High[axis] > node.High[axis] {
					t.Fatalf("node %d does not hold child %d", at, child)
				}
			}
			visit(child)
		}
	}
	visit(0)
	for index, count := range seen {
		if count != 1 {
			t.Fatalf("box %d is in %d leaves", index, count)
		}
	}
}

// TestBoxTreeHoldsEveryBoxOnce checks the tree over scattered boxes.
func TestBoxTreeHoldsEveryBoxOnce(t *testing.T) {
	for _, count := range []int{1, 2, 3, 17, 500, 4000} {
		boxes := scatter(count, int64(count))
		tree, err := BuildBoxTree(boxes, testStack)
		if err != nil {
			t.Fatal(err)
		}
		checkTree(t, tree, boxes)
	}
}

// TestBoxTreeIsTheSameEveryRun checks that boxes alike are cut by their
// index, so two builds give the same nodes.
func TestBoxTreeIsTheSameEveryRun(t *testing.T) {
	boxes := make([]TreeBox, 300)
	for index := range boxes {
		boxes[index] = unitBox(float64(index%3), 0, 0)
	}
	one, err := BuildBoxTree(boxes, testStack)
	if err != nil {
		t.Fatal(err)
	}
	two, _ := BuildBoxTree(boxes, testStack)
	checkTree(t, one, boxes)
	for index := range one.Nodes {
		if one.Nodes[index] != two.Nodes[index] {
			t.Fatalf("node %d differs between two builds", index)
		}
	}
}

// TestBoxTreeStaysInItsStack builds over boxes whose sizes grow by powers,
// the worst case for the heuristic, and checks that a walker of the stack
// given holds the tree: its depth plus the spare is within it.
func TestBoxTreeStaysInItsStack(t *testing.T) {
	boxes := make([]TreeBox, 3000)
	for index := range boxes {
		growth := math.Pow(1.01, float64(index))
		boxes[index] = unitBox(growth, 0, 0)
	}
	for _, stack := range []int{testStack, 32, 20} {
		tree, err := BuildBoxTree(boxes, stack)
		if err != nil {
			t.Fatal(err)
		}
		checkTree(t, tree, boxes)
		if depth := depthOf(tree, 0); depth+walkSpare > stack {
			t.Fatalf("depth %d does not fit a stack of %d", depth, stack)
		}
	}
}

// TestBoxTreeOfOneBoxIsALeaf checks the root of a single box.
func TestBoxTreeOfOneBoxIsALeaf(t *testing.T) {
	tree, err := BuildBoxTree([]TreeBox{unitBox(1, 2, 3)}, testStack)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Nodes) != 1 || !tree.Nodes[0].IsLeaf() {
		t.Fatalf("root of one box is %+v", tree.Nodes)
	}
}

// TestBoxTreeRefusals is the anti-test: what is not a tree is refused, and
// says why.
func TestBoxTreeRefusals(t *testing.T) {
	infinite := unitBox(0, 0, 0)
	infinite.High[1] = math.Inf(1)
	inside := unitBox(0, 0, 0)
	inside.Low[2] = 5
	not := unitBox(0, 0, 0)
	not.Low[0] = math.NaN()
	for _, test := range []struct {
		name  string
		boxes []TreeBox
		stack int
		want  error
	}{
		{"none", nil, testStack, ErrNoBoxes},
		{"infinite", []TreeBox{unitBox(1, 1, 1), infinite}, testStack,
			ErrBoxShape},
		{"inside out", []TreeBox{inside}, testStack, ErrBoxShape},
		{"nan", []TreeBox{not}, testStack, ErrBoxShape},
		{"stack", scatter(1000, 1), 5, ErrBoxStack},
	} {
		if _, err := BuildBoxTree(test.boxes, test.stack); !errors.Is(err,
			test.want) {
			t.Errorf("%s: got %v, want %v", test.name, err, test.want)
		}
	}
}

// TestBoxTreeCutsApartAWideGap checks that the heuristic parts two far
// clusters at the root, which a cut at the median of one cluster would not.
func TestBoxTreeCutsApartAWideGap(t *testing.T) {
	var boxes []TreeBox
	for index := range 10 {
		offset := float64(index) * 0.1
		boxes = append(boxes, unitBox(offset, 0, 0))
	}
	for index := range 2 {
		offset := 10000 + float64(index)
		boxes = append(boxes, unitBox(offset, 0, 0))
	}
	tree, err := BuildBoxTree(boxes, testStack)
	if err != nil {
		t.Fatal(err)
	}
	root := tree.Nodes[0]
	left, right := tree.Nodes[root.First], tree.Nodes[root.First+1]
	parted := (left.High[0] < 5000 && right.Low[0] > 5000) ||
		(right.High[0] < 5000 && left.Low[0] > 5000)
	if !parted {
		t.Fatalf("root does not part the clusters: %+v %+v", left, right)
	}
}
