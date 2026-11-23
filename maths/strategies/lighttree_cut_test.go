package strategies

import (
	"math"
	"testing"
)

// cutFloor is the floor the cut tests weigh a node by: a millimetre
// squared, as the tracer's.
const cutFloor = 1e-6

// below is every lamp in the leaves at or under a node, counted once for
// each leaf it is in.
func below(tree LightTree[float64], at int32) []int32 {
	node := tree.Nodes[at]
	if node.IsLeaf() {
		return []int32{node.Leaf}
	}
	return append(below(tree, node.Left), below(tree, node.Right)...)
}

// streetTree is the tree of a street of count lamps in a row.
func streetTree(t *testing.T, count int) (LightTree[float64], [][3]float64,
	[]float64) {
	t.Helper()
	places, powers := street(count)
	tree, err := BuildLightTree(places, powers, ByCoordinateThenIndex)
	if err != nil {
		t.Fatal(err)
	}
	return tree, places, powers
}

// TestACutHoldsEveryLampInExactlyOneNode says that whatever the point, the
// nodes of a cut between them hold every lamp of the tree once: a lamp in
// none would leave its light out, and one in two would count it twice.
func TestACutHoldsEveryLampInExactlyOneNode(t *testing.T) {
	tree, places, _ := streetTree(t, 40)
	for _, point := range [][3]float64{{30, 3, 0}, {30, 3.5, 0},
		{30, 40, 0}, {-500, 0, 0}} {
		seen := map[int32]int{}
		for _, node := range Cut(tree, point, 1, 8, cutFloor) {
			for _, lamp := range below(tree, node) {
				seen[lamp]++
			}
		}
		if len(seen) != len(places) {
			t.Errorf("from %v the cut holds %d lamps, want %d", point,
				len(seen), len(places))
		}
		for lamp, times := range seen {
			if times != 1 {
				t.Errorf("from %v lamp %d is in %d nodes", point, lamp, times)
			}
		}
	}
}

// TestACutSplitsAmongTheLampsAndNotFarFromThem says that a point inside
// the lamps' box has its root split, and one far off, where the box is
// small for its distance, takes the root whole.
func TestACutSplitsAmongTheLampsAndNotFarFromThem(t *testing.T) {
	tree, _, _ := streetTree(t, 40)
	if cut := Cut(tree, [3]float64{30, 3, 0}, 1, 8, cutFloor); len(cut) < 2 {
		t.Errorf("a point among the lamps took the root whole: %v", cut)
	}
	cut := Cut(tree, [3]float64{0, 5000, 0}, 1, 8, cutFloor)
	if len(cut) != 1 || cut[0] != 0 {
		t.Errorf("a point far off took %v, want the root alone", cut)
	}
}

// TestACutHoldsNoMoreNodesThanItIsGiven is the hard bound: however many
// nodes are in doubt, the cut stops at most.
func TestACutHoldsNoMoreNodesThanItIsGiven(t *testing.T) {
	tree, _, _ := streetTree(t, 200)
	for _, most := range []int{1, 2, 5, 8, 13} {
		got := Cut(tree, [3]float64{200, 3, 0}, 0.01, most, cutFloor)
		if len(got) != most {
			t.Errorf("most %d: a cut of %d nodes, want %d", most, len(got),
				most)
		}
	}
}

// TestTheNodeOfMostWorthIsSplitFirst says that with room for two nodes the
// cut is the root's halves, and that with room for three the half the
// point is inside is the one split again, and the far half is kept.
func TestTheNodeOfMostWorthIsSplitFirst(t *testing.T) {
	tree, _, _ := streetTree(t, 40)
	point := [3]float64{6, 3, 0}
	root := tree.Nodes[0]
	two := Cut(tree, point, 1, 2, cutFloor)
	if len(two) != 2 || two[0] != root.Left || two[1] != root.Right {
		t.Fatalf("two nodes %v, want the halves %d and %d", two, root.Left,
			root.Right)
	}
	three := Cut(tree, point, 1, 3, cutFloor)
	kept := map[int32]bool{}
	for _, node := range three {
		kept[node] = true
	}
	if len(three) != 3 || kept[root.Left] || !kept[root.Right] {
		t.Errorf("three nodes %v: want the near half %d split and the far "+
			"half %d kept", three, root.Left, root.Right)
	}
}

// TestAWalkByPowerTakesALampByItsShareOfPower says that a lamp's chance
// in a walk by power is its power over its node's, whatever the share:
// the weight of the pick is then the node's power over the lamp's.
func TestAWalkByPowerTakesALampByItsShareOfPower(t *testing.T) {
	tree, _, powers := streetTree(t, 11)
	const shares = 2048
	for each := range shares {
		lamp, chance := WalkByPower(tree, 0,
			(float64(each)+0.5)/shares)
		want := powers[lamp] / tree.Nodes[0].Power
		if lamp == NoNode || math.Abs(chance-want) > 1e-12 {
			t.Fatalf("lamp %d with chance %v, want %v", lamp, chance, want)
		}
	}
}

// TestAWalkFromANodeStaysAmongItsLamps says that a walk begun at a node
// takes one of that node's lamps and none else.
func TestAWalkFromANodeStaysAmongItsLamps(t *testing.T) {
	tree, _, _ := streetTree(t, 40)
	half := tree.Nodes[0].Right
	own := map[int32]bool{}
	for _, lamp := range below(tree, half) {
		own[lamp] = true
	}
	by := Weighing[float64]{Floor: cutFloor, InsideHalfDiagonal: true}
	point := [3]float64{6, 3, 0}
	for each := range 64 {
		share := (float64(each) + 0.5) / 64
		lamp, chance := WalkFrom(tree, half, point, share, by)
		power, _ := WalkByPower(tree, half, share)
		if !own[lamp] || !own[power] || !(chance > 0) {
			t.Fatalf("a walk from node %d took lamp %d and %d", half, lamp,
				power)
		}
	}
}

// TestANodeIsReliableOnlyAwayFromItsBoxForItsSize says that a node two
// across is reliable from two away at a spread of one and not from one
// away, nor from inside it, and that a leaf is reliable at its lamp.
func TestANodeIsReliableOnlyAwayFromItsBoxForItsSize(t *testing.T) {
	node := LightNode[float64]{Low: [3]float64{0, 0, 0},
		High: [3]float64{2, 0, 0}, Power: 1, Leaf: NoNode}
	for point, want := range map[[3]float64]bool{{1, 0, 0}: false,
		{3, 0, 0}: false, {4, 0, 0}: true, {40, 0, 0}: true} {
		if got := node.Reliable(point, 1); got != want {
			t.Errorf("node reliable from %v: %v, want %v", point, got, want)
		}
	}
	leaf := LightNode[float64]{Low: [3]float64{1, 1, 1},
		High: [3]float64{1, 1, 1}, Power: 1, Leaf: 3}
	if !leaf.Reliable([3]float64{1, 1, 1}, 1) {
		t.Error("a leaf at the point is not reliable")
	}
}

// TestAPickFromAReliableNodeWeighsAtMostFourTimesItsLight is the bound
// the cut is for: from a point, a pick of one lamp by power from each
// node of a cut at a spread of one, weighed by one over its chance, is no
// more than four times the light of that node's lamps, for every lamp.
func TestAPickFromAReliableNodeWeighsAtMostFourTimesItsLight(t *testing.T) {
	tree, places, powers := streetTree(t, 64)
	light := func(lamp int32, point [3]float64) float64 {
		var square float64
		for axis := range 3 {
			gap := places[lamp][axis] - point[axis]
			square += gap * gap
		}
		return powers[lamp] / math.Max(square, cutFloor)
	}
	for _, point := range [][3]float64{{30, 3, 0}, {30, 9, 0},
		{-40, 3, 0}, {200, 60, 0}} {
		for _, node := range Cut(tree, point, 1, 64, cutFloor) {
			if !tree.Nodes[node].Reliable(point, 1) {
				continue
			}
			sum := 0.0
			for _, lamp := range below(tree, node) {
				sum += light(lamp, point)
			}
			for _, lamp := range below(tree, node) {
				weight := light(lamp, point) *
					tree.Nodes[node].Power / powers[lamp]
				if weight > 4*sum*(1+1e-9) {
					t.Errorf("from %v lamp %d weighs %g, node light %g",
						point, lamp, weight, sum)
				}
			}
		}
	}
}
