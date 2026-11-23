package strategies

import (
	"math"
	"testing"
)

// street is lamps in a row along x, two at each place of the row so that
// lamps share a coordinate, each with the power its index plus one.
func street(count int) ([][3]float64, []float64) {
	var places [][3]float64
	var powers []float64
	for index := range count {
		places = append(places, [3]float64{float64(index / 2 * 4), 3, 0})
		powers = append(powers, float64(index+1))
	}
	return places, powers
}

// leaves is every lamp in the leaves of a tree, counted.
func leaves[T Number](tree LightTree[T]) map[int32]int {
	found := map[int32]int{}
	for _, node := range tree.Nodes {
		if node.IsLeaf() {
			found[node.Leaf]++
		}
	}
	return found
}

func TestEveryLampIsInExactlyOneLeaf(t *testing.T) {
	places, powers := street(11)
	for _, order := range []Order{ByCoordinate, ByCoordinateThenIndex} {
		tree, err := BuildLightTree(places, powers, order)
		if err != nil {
			t.Fatal(err)
		}
		found := leaves(tree)
		if len(found) != len(places) {
			t.Errorf("order %d: %d lamps in leaves, want %d", order,
				len(found), len(places))
		}
		for lamp, count := range found {
			if count != 1 {
				t.Errorf("order %d: lamp %d in %d leaves", order, lamp, count)
			}
		}
		if len(tree.Nodes) != 2*len(places)-1 {
			t.Errorf("%d nodes for %d lamps", len(tree.Nodes), len(places))
		}
	}
}

func TestARootHoldsAllThePowerAndTheBoxOfAllTheLamps(t *testing.T) {
	places, powers := street(6)
	tree, err := BuildLightTree(places, powers, ByCoordinate)
	if err != nil {
		t.Fatal(err)
	}
	root := tree.Nodes[0]
	if root.Power != 21 || root.Low != [3]float64{0, 3, 0} ||
		root.High != [3]float64{8, 3, 0} {
		t.Errorf("a root of %+v", root)
	}
}

func TestOrderingByIndexPutsTheLowerIndexOfAPairFirst(t *testing.T) {
	places := [][3]float64{{1, 0, 0}, {1, 0, 0}, {1, 0, 0}, {1, 0, 0}}
	powers := []float64{1, 1, 1, 1}
	tree, err := BuildLightTree(places, powers, ByCoordinateThenIndex)
	if err != nil {
		t.Fatal(err)
	}
	left := tree.Nodes[tree.Nodes[0].Left]
	right := tree.Nodes[tree.Nodes[0].Right]
	if left.IsLeaf() || right.IsLeaf() {
		t.Fatal("four lamps in two leaves")
	}
	if tree.Nodes[left.Left].Leaf != 0 || tree.Nodes[left.Right].Leaf != 1 ||
		tree.Nodes[right.Left].Leaf != 2 || tree.Nodes[right.Right].Leaf != 3 {
		t.Errorf("lamps of one place are not in index order")
	}
}

func TestNoLampsAndMismatchedPowersAreRefused(t *testing.T) {
	if _, err := BuildLightTree[float64](nil, nil, ByCoordinate); err == nil {
		t.Error("a tree over no lamps")
	}
	if _, err := BuildLightTree([][3]float64{{}}, nil, ByCoordinate); err == nil {
		t.Error("a tree over a place with no power")
	}
}

func TestAWalkTakesALampWithTheChanceItsWorthsGive(t *testing.T) {
	places, powers := street(8)
	tree, _ := BuildLightTree(places, powers, ByCoordinate)
	by := Weighing[float64]{Floor: 1e-6, InsideHalfDiagonal: true}
	point := [3]float64{5, 0, 2}
	const shares = 2048
	total := 0.0
	for each := range shares {
		lamp, chance := Walk(tree, point, (float64(each)+0.5)/shares, by)
		if lamp == NoNode || !(chance > 0) || chance > 1 {
			t.Fatalf("a walk to lamp %d with chance %v", lamp, chance)
		}
		total += 1 / chance / shares
	}
	count := float64(len(places))
	if math.Abs(total-count) > 0.05*count {
		t.Errorf("the walks' weights sum to %v, want about %d", total,
			len(places))
	}
}

func TestAPointInsideABoxIsWorthMoreOrLessByTheRuleChosen(t *testing.T) {
	node := LightNode[float64]{Low: [3]float64{0, 0, 0},
		High: [3]float64{2, 0, 0}, Power: 1, Leaf: NoNode}
	point := [3]float64{1, 0, 0}
	half := node.Worth(point, Weighing[float64]{Floor: 1e-4,
		InsideHalfDiagonal: true})
	floor := node.Worth(point, Weighing[float64]{Floor: 1e-4})
	if math.Abs(half-1) > 1e-12 || math.Abs(floor-1e4) > 1e-6 {
		t.Errorf("inside: %v with the half diagonal, %v with the floor",
			half, floor)
	}
}

func TestABoxBehindTheSurfaceIsWorthNothingWhenFacingIsGiven(t *testing.T) {
	node := LightNode[float64]{Low: [3]float64{0, -3, 0},
		High: [3]float64{1, -2, 1}, Power: 1, Leaf: NoNode}
	point := [3]float64{0, 0, 0}
	up := [3]float64{0, 1, 0}
	if got := node.Worth(point, Weighing[float64]{Floor: 1e-4,
		Facing: &up}); got != 0 {
		t.Errorf("a box behind the plane is worth %v", got)
	}
	if got := node.Worth(point, Weighing[float64]{Floor: 1e-4}); !(got > 0) {
		t.Errorf("a box with no facing is worth %v", got)
	}
}
