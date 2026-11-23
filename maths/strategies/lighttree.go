package strategies

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// The light tree: a binary tree over lamps, each a place and a power, that
// a point walks down to pick one lamp, the brighter and nearer half more
// often. The tracer and the gpu both build it, over their own lamps, from
// plain numbers: a place, a power and the lamp's index. They differ in two
// things, and each is a choice here and not a fact:
//
//   - how lamps of one coordinate are ordered when a node is split: the
//     tracer's sort leaves them as the sort finds them (ByCoordinate), the
//     gpu's puts the lower index first (ByCoordinateThenIndex). A street of
//     windows has many lamps at one height, so the two trees differ.
//   - how a node is worth a point (see Weighing).
//
// The arithmetic is the number type's own, so a tree of float32 is the
// tree of float32 arithmetic and one of float64 is the tree of float64.

// Number is the type a tree's places and powers are given in.
type Number interface{ ~float32 | ~float64 }

// NoNode marks a node with no child, and a lamp that is none.
const NoNode = -1

// LightNode is a fork or a leaf of the tree: the box its lamps stand in, the
// light they send between them a unit away, and either its two children or,
// for a leaf, its one lamp, which is that lamp's index in the places the tree
// was built over.
type LightNode[T Number] struct {
	Low, High   [3]T
	Power       T
	Left, Right int32
	Leaf        int32
}

// IsLeaf is whether the node holds a lamp.
func (node LightNode[T]) IsLeaf() bool { return node.Leaf != NoNode }

// LightTree is the nodes of the tree, the root first.
type LightTree[T Number] struct {
	Nodes []LightNode[T]
}

// Order is how lamps of one coordinate are ordered when a node splits.
type Order int

const (
	// ByCoordinate sorts by the coordinate only, and leaves lamps of one
	// coordinate where the sort puts them.
	ByCoordinate Order = iota
	// ByCoordinateThenIndex sorts by the coordinate, and lamps of one
	// coordinate by their index, so the order is the same on every run.
	ByCoordinateThenIndex
)

// ErrNoLamps is a tree asked of no lamps.
var ErrNoLamps = errors.New("strategies: a light tree needs at least one lamp")

// BuildLightTree is the tree over lamps at places with powers, split at the
// median of the longest side of the box of the lamps of a node. A lamp's
// number in a leaf is its index in places. It refuses no lamps, and places
// and powers that are not as many.
func BuildLightTree[T Number](places [][3]T, powers []T,
	order Order) (LightTree[T], error) {
	if len(places) == 0 {
		return LightTree[T]{}, ErrNoLamps
	}
	if len(places) != len(powers) {
		return LightTree[T]{}, fmt.Errorf("strategies: %d places and %d "+
			"powers", len(places), len(powers))
	}
	lamps := make([]int, len(places))
	for index := range lamps {
		lamps[index] = index
	}
	tree := LightTree[T]{Nodes: make([]LightNode[T], 0, 2*len(places)-1)}
	tree.grow(lamps, places, powers, order)
	return tree, nil
}

// grow adds the node of the lamps given and, below it, theirs, and says
// where the node went.
func (tree *LightTree[T]) grow(lamps []int, places [][3]T, powers []T,
	order Order) int32 {
	at := int32(len(tree.Nodes))
	tree.Nodes = append(tree.Nodes, LightNode[T]{})
	node := LightNode[T]{Leaf: NoNode, Left: NoNode, Right: NoNode,
		Low: places[lamps[0]], High: places[lamps[0]]}
	for _, lamp := range lamps {
		node.Power += powers[lamp]
		for axis := range 3 {
			node.Low[axis] = min(node.Low[axis], places[lamp][axis])
			node.High[axis] = max(node.High[axis], places[lamp][axis])
		}
	}
	if len(lamps) == 1 {
		node.Leaf = int32(lamps[0])
		tree.Nodes[at] = node
		return at
	}
	longest := 0
	for axis := 1; axis < 3; axis++ {
		if node.High[axis]-node.Low[axis] >
			node.High[longest]-node.Low[longest] {
			longest = axis
		}
	}
	sort.Slice(lamps, func(one, other int) bool {
		a, b := places[lamps[one]][longest], places[lamps[other]][longest]
		if a != b || order == ByCoordinate {
			return a < b
		}
		return lamps[one] < lamps[other]
	})
	half := len(lamps) / 2
	node.Left = tree.grow(lamps[:half], places, powers, order)
	node.Right = tree.grow(lamps[half:], places, powers, order)
	tree.Nodes[at] = node
	return at
}

// Weighing is how a node is worth a point: its power over the squared
// distance from the point to its box, held up by a floor. Two rules, one for
// each of the tracer and the gpu, differ in what they add:
//
//   - InsideHalfDiagonal: for a point inside the box the distance is half
//     the box's diagonal, since a point among lamps is not as near all of
//     them as to the nearest. Without it the distance there is nothing, and
//     the floor decides. The tracer sets it, the gpu does not.
//   - Facing: a surface normal. A box wholly behind the plane through the
//     point that the normal stands on is worth nothing. The gpu sets it, the
//     tracer has none.
//
// Floor is the least squared distance a node is weighed at, so a point on a
// lamp does not weigh it without end: the tracer's is a millimetre squared,
// the gpu's a hundredth of a metre squared.
type Weighing[T Number] struct {
	Floor              T
	InsideHalfDiagonal bool
	Facing             *[3]T
}

// behind is whether all of a box is behind the plane through a point that a
// normal stands on, or on it.
func behind[T Number](node LightNode[T], point, normal [3]T) bool {
	for corner := range 8 {
		var along T
		for axis := range 3 {
			edge := node.Low[axis]
			if corner>>axis&1 == 1 {
				edge = node.High[axis]
			}
			along += normal[axis] * (edge - point[axis])
		}
		if along > 0 {
			return false
		}
	}
	return true
}

// Worth is how much a node's lamps may light a point, as a share to walk by.
func (node LightNode[T]) Worth(point [3]T, by Weighing[T]) T {
	if by.Facing != nil && behind(node, point, *by.Facing) {
		return 0
	}
	var square T
	for axis := range 3 {
		gap := max(node.Low[axis]-point[axis], 0, point[axis]-node.High[axis])
		square += gap * gap
	}
	if square == 0 && by.InsideHalfDiagonal {
		for axis := range 3 {
			half := (node.High[axis] - node.Low[axis]) * T(0.5)
			square += half * half
		}
	}
	return node.Power / max(square, by.Floor)
}

// Walk is the lamp a walk down a tree of float64 from the root takes, with a
// share from nought up to but not including one choosing the way at each
// fork, and the chance of taking it: the lamp's index and a chance, or no
// lamp and a chance of nought where nothing could light the point.
func Walk(tree LightTree[float64], point [3]float64, share float64,
	by Weighing[float64]) (int32, float64) {
	return WalkFrom(tree, 0, point, share, by)
}

// WalkFrom is Walk begun at a node of the tree and not at its root: the
// lamp among that node's own, and the chance of taking it among them.
func WalkFrom(tree LightTree[float64], from int32, point [3]float64,
	share float64, by Weighing[float64]) (int32, float64) {
	return walkBy(tree, from, share, func(node LightNode[float64]) float64 {
		return node.Worth(point, by)
	})
}

// WalkByPower is WalkFrom with no regard for where the point is: each fork
// is taken by the power of its two halves, so a lamp's chance among the
// node's lamps is its power over theirs, whatever the point. It is the
// pick for a node that Reliable says the point sees as one distance away.
func WalkByPower(tree LightTree[float64], from int32,
	share float64) (int32, float64) {
	return walkBy(tree, from, share, func(node LightNode[float64]) float64 {
		return node.Power
	})
}

// walkBy is the walk of a tree from a node, each fork taken by the worth
// that worth gives its two halves.
func walkBy(tree LightTree[float64], from int32, share float64,
	worth func(node LightNode[float64]) float64) (int32, float64) {
	node, chance := tree.Nodes[from], 1.0
	for !node.IsLeaf() {
		left, right := tree.Nodes[node.Left], tree.Nodes[node.Right]
		toLeft, toRight := worth(left), worth(right)
		both := toLeft + toRight
		if !(both > 0) || math.IsInf(both, 0) {
			return NoNode, 0
		}
		leftward := toLeft / both
		if share < leftward {
			share, chance, node = share/leftward, chance*leftward, left
		} else {
			share = (share - leftward) / (1 - leftward)
			chance, node = chance*(1-leftward), right
		}
		share = min(max(share, 0), math.Nextafter(1, 0))
	}
	return node.Leaf, chance
}

// Reliable is whether a node's worth can be trusted to share a walk by.
// The point lies outside its box, and the box is no wider across than
// spread times the distance from the point to it.
//
// No lamp of the node is then nearer than that distance, nor farther than
// one plus spread times it, so two of its isotropic lamps of one power
// light the point within the square of one plus spread of each other. A
// leaf is reliable, and so is a node whose lamps all stand at one place.
func (node LightNode[T]) Reliable(point [3]T, spread T) bool {
	var gapSquare, acrossSquare T
	for axis := range 3 {
		gap := max(node.Low[axis]-point[axis], 0, point[axis]-node.High[axis])
		across := node.High[axis] - node.Low[axis]
		gapSquare += gap * gap
		acrossSquare += across * across
	}
	return acrossSquare <= spread*spread*gapSquare
}

// doubt is nought less one for a node that is Reliable, and otherwise the
// most its lamps can light the point: its power over the squared distance
// to its box, held up by floor, which no lamp of the node can beat.
func doubt(node LightNode[float64], point [3]float64, spread,
	floor float64) float64 {
	if node.Reliable(point, spread) {
		return -1
	}
	var square float64
	for axis := range 3 {
		gap := max(node.Low[axis]-point[axis], 0, point[axis]-node.High[axis])
		square += gap * gap
	}
	return node.Power / max(square, floor)
}

// Cut is the nodes a tree is cut into for a point, after Conty Estevez and
// Kulla, "Importance sampling of many lights with adaptive tree splitting"
// (2018): a node whose worth cannot be trusted (not Reliable: the point
// inside it, or near it for its size) is not picked among but split, and
// both its halves taken in its place.
//
// The cut has at most most nodes. The node of most doubt is split first,
// the node round the point, its distance nought, so when the cut is full
// it is that node, and any of less doubt, that stay in doubt. The nodes'
// lamps are every lamp of the tree, each in exactly one node.
func Cut(tree LightTree[float64], point [3]float64, spread float64,
	most int, floor float64) []int32 {
	cut := []int32{0}
	doubts := []float64{doubt(tree.Nodes[0], point, spread, floor)}
	for len(cut) < most {
		worst := 0
		for at := 1; at < len(cut); at++ {
			if doubts[at] > doubts[worst] {
				worst = at
			}
		}
		if !(doubts[worst] > 0) {
			break
		}
		node := tree.Nodes[cut[worst]]
		left, right := tree.Nodes[node.Left], tree.Nodes[node.Right]
		cut[worst] = node.Left
		doubts[worst] = doubt(left, point, spread, floor)
		cut = append(cut, node.Right)
		doubts = append(doubts, doubt(right, point, spread, floor))
	}
	return cut
}
