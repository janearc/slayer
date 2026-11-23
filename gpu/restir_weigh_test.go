package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths/strategies"
)

// TestTheKernelsWeighingIsTheSharedWeighingWithItsFloorAndFacing says the
// weighing the restir kernel is mirrored by is one of the shared
// weighings: no half diagonal inside a box, this file's floor, and the
// normal given. A change to either is then a change to both.
func TestTheKernelsWeighingIsTheSharedWeighingWithItsFloorAndFacing(
	t *testing.T) {
	places := [][3]float32{{0, 3, 0}, {4, 3, 0}, {4, 3, 4}, {9, 8, 1}}
	powers := []float32{1, 2, 3, 4}
	tree, err := buildLightTree(places, powers)
	if err != nil {
		t.Fatal(err)
	}
	points := [][3]float32{{1, 0, 1}, {4, 3, 0}, {-5, 20, 3}, {9, 8, 1}}
	normals := [][3]float32{{0, 1, 0}, {0, -1, 0}, {1, 0, 0}}
	for _, node := range tree.nodes {
		shared := strategies.LightNode[float32]{Low: node.Low,
			High: node.High, Power: node.Power, Left: node.Left,
			Right: node.Right, Leaf: node.Lamp}
		for _, point := range points {
			for _, normal := range normals {
				by := strategies.Weighing[float32]{Floor: restirFloor,
					Facing: &normal}
				if got, want := shared.Worth(point, by),
					node.weigh(point, normal); got != want {
					t.Errorf("point %v normal %v: shared %v, kernel %v",
						point, normal, got, want)
				}
			}
		}
	}
}

// TestTheTreeIsBuiltInIndexOrderForLampsOfOnePlace says that lamps at one
// place keep their index order in the leaves, the gpu's rule.
func TestTheTreeIsBuiltInIndexOrderForLampsOfOnePlace(t *testing.T) {
	places := [][3]float32{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	tree, err := buildLightTree(places, []float32{1, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	var order []int32
	var down func(at int32)
	down = func(at int32) {
		node := tree.nodes[at]
		if node.leaf() {
			order = append(order, node.Lamp)
			return
		}
		down(node.Left)
		down(node.Right)
	}
	down(0)
	if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Errorf("leaves in order %v, want 0 1 2", order)
	}
}
