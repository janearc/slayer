package gpu

import (
	"fmt"

	"github.com/janearc/slayer/maths/strategies"
)

// The light tree of restir (doc/restir.md): a binary tree over the lamps
// that light from a place, which the restir kernels walk to pick a
// candidate lamp for a surface point. The CPU builds it; the kernel reads
// it as floats, nodeFloats a node. This file has no Metal in it.

// nodeFloats is the floats one node takes on the device: the box's low
// corner and the node's power, the box's high corner and the left child,
// then the right child, the lamp (minus one for a node that is not a
// leaf), and two spare.
const nodeFloats = 12

// A lightNode is a box of lamps, their total luminance, and either two
// children or one lamp.
type lightNode struct {
	Low, High   [3]float32
	Power       float32
	Left, Right int32
	// Lamp is the lamp's index among the world's lamps for a leaf, and
	// minus one for a node with children.
	Lamp int32
}

// leaf is whether a node holds a lamp.
func (node lightNode) leaf() bool { return node.Lamp >= 0 }

// A lightTree is the nodes, root first.
type lightTree struct {
	nodes []lightNode
}

// errNoCrowd is a tree asked of no lamps.
var errNoCrowd = fmt.Errorf("gpu: restir needs at least one lamp that " +
	"lights from a place")

// crowdLamp is whether a lamp row is one the tree holds: it has a
// place, and is not a flood, a polar lamp, a beam or a sun. row is the
// lamp's lampStruct row as floats: at, strength, toward, extent, second.
func crowdLamp(row []float32) bool {
	kind := row[19]
	beam := row[13] > 0
	sun := row[3] < 0
	return kind == 0 && !beam && !sun
}

// lampFloats is a lamp row's floats.
const lampFloats = 20

// crowdOf reads the lamps' rows and says which lamps are in the crowd,
// with their places and luminances. It is refused with the first lamp
// that has no place, so a world is drawn whole or not at all.
func crowdOf(rows []float32) (places [][3]float32, powers []float32,
	err error) {
	count := len(rows) / lampFloats
	for index := range count {
		row := rows[index*lampFloats : (index+1)*lampFloats]
		if !crowdLamp(row) {
			return nil, nil, unsupported{fmt.Sprintf(
				"restir with lamp %d, which lights from no one place "+
					"(a flood, a sun, a beam or a polar lamp)", index)}
		}
		places = append(places, [3]float32{row[0], row[1], row[2]})
		powers = append(powers, 0.2126*row[4]+0.7152*row[5]+0.0722*row[6])
	}
	if count == 0 {
		return nil, nil, errNoCrowd
	}
	return places, powers, nil
}

// buildLightTree is the tree over lamps at places with the luminances
// powers, split at the median of the longest axis of the lamps' places.
// The lamp's number in the tree's leaf is its index in places.
func buildLightTree(places [][3]float32, powers []float32) (lightTree, error) {
	if len(places) == 0 {
		return lightTree{}, errNoCrowd
	}
	if len(places) != len(powers) {
		return lightTree{}, fmt.Errorf("gpu: %d places and %d powers",
			len(places), len(powers))
	}
	// the build refuses only what is refused above
	shared, _ := strategies.BuildLightTree(places, powers,
		strategies.ByCoordinateThenIndex)
	tree := lightTree{nodes: make([]lightNode, len(shared.Nodes))}
	for index, node := range shared.Nodes {
		tree.nodes[index] = lightNode{Low: node.Low, High: node.High,
			Power: node.Power, Left: node.Left, Right: node.Right,
			Lamp: node.Leaf}
	}
	return tree, nil
}

// floats is the tree as the kernel reads it.
func (tree lightTree) floats() []float32 {
	out := make([]float32, len(tree.nodes)*nodeFloats)
	for index, node := range tree.nodes {
		at := out[index*nodeFloats:]
		copy(at[0:3], node.Low[:])
		at[3] = node.Power
		copy(at[4:7], node.High[:])
		at[7] = float32(node.Left)
		at[8] = float32(node.Right)
		at[9] = float32(node.Lamp)
	}
	return out
}

// restirFloor is the least squared distance a node is weighed at, so a
// point on top of a lamp does not weigh it without end.
const restirFloor = 1e-4

// weigh is how much a node is worth as a proposal for a point with a
// normal: its power over the squared distance to its box, and nought for
// a box wholly behind the point's plane. It mirrors tree_weight in the
// kernel.
func (node lightNode) weigh(point, normal [3]float32) float32 {
	behind := true
	for corner := range 8 {
		var along float32
		for axis := range 3 {
			edge := node.Low[axis]
			if corner>>axis&1 == 1 {
				edge = node.High[axis]
			}
			along += normal[axis] * (edge - point[axis])
		}
		if along > 0 {
			behind = false
			break
		}
	}
	if behind {
		return 0
	}
	var apart float32
	for axis := range 3 {
		gap := max(node.Low[axis]-point[axis], 0, point[axis]-node.High[axis])
		apart += gap * gap
	}
	return node.Power / max(apart, restirFloor)
}

// sample takes a lamp for a point and a normal by the number u in the
// half-open range from nought to one, spent as the walk goes down, and says
// the lamp and the chance of that lamp. It mirrors the kernel's walk. A
// point nothing lights has no lamp, and says minus one.
func (tree lightTree) sample(point, normal [3]float32, u float32) (int32, float32) {
	at, chance := int32(0), float32(1)
	for {
		node := tree.nodes[at]
		if node.leaf() {
			return node.Lamp, chance
		}
		left := tree.nodes[node.Left].weigh(point, normal)
		right := tree.nodes[node.Right].weigh(point, normal)
		total := left + right
		if !(total > 0) {
			return -1, 0
		}
		share := left / total
		if u < share {
			at, chance, u = node.Left, chance*share, u/share
		} else {
			at, chance, u = node.Right, chance*(1-share), (u-share)/(1-share)
		}
	}
}

// chanceOf is the chance sample has of taking lamp for a point and a
// normal, which is what the resampling weight divides by.
func (tree lightTree) chanceOf(point, normal [3]float32, lamp int32) float32 {
	at, chance := int32(0), float32(1)
	for {
		node := tree.nodes[at]
		if node.leaf() {
			if node.Lamp == lamp {
				return chance
			}
			return 0
		}
		left, right := tree.nodes[node.Left], tree.nodes[node.Right]
		weights := [2]float32{left.weigh(point, normal),
			right.weigh(point, normal)}
		total := weights[0] + weights[1]
		if !(total > 0) {
			return 0
		}
		inLeft := contains(tree.nodes, node.Left, lamp)
		if inLeft {
			at, chance = node.Left, chance*weights[0]/total
		} else {
			at, chance = node.Right, chance*weights[1]/total
		}
	}
}

// contains is whether the subtree at node holds lamp.
func contains(nodes []lightNode, at int32, lamp int32) bool {
	node := nodes[at]
	if node.leaf() {
		return node.Lamp == lamp
	}
	return contains(nodes, node.Left, lamp) || contains(nodes, node.Right, lamp)
}
