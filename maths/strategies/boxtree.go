package strategies

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"sort"
)

// The box tree: a binary tree over boxes, one box to a leaf, that a ray
// walks from the root, skipping every branch whose box it misses. It is
// built by the surface area heuristic over bins of the boxes' middles, as
// Wald's "On fast construction of SAH-based bounding volume hierarchies"
// (2007) has it: of every cut along every axis, the one that costs least,
// where a side costs its box's surface area times its count of boxes.
//
// The tree is laid out for a walker with a fixed stack. A branch's two
// children are side by side, so a branch holds the place of its first and
// the second is the next.
//
// A walk that holds a node waiting for each level
// it has descended, and one more for the sibling it has just put there,
// fits in a stack of the depth plus two; the build keeps the depth within
// the stack it is given, by cutting at the median where the heuristic
// would go deeper.

const (
	// boxBins is how many bins the middles of a run of boxes fall in
	// along one axis. The cuts tried are the edges between them.
	boxBins = 16
	// walkSpare is the nodes a walker's stack holds beyond the depth of the
	// tree: the sibling of the node it takes, and the node itself.
	walkSpare = 2
	// boxAxes is the sides of a box, x, y and z.
	boxAxes = 3
)

// TreeBox is a box that a leaf holds, by its two corners.
type TreeBox struct {
	Low, High [3]float64
}

// BoxTreeNode is a branch or a leaf of the tree: the box it holds, and
// either its first child (the second is the next node), with a Count of
// nought, or, for a leaf, the index of its one box, with a Count of one.
type BoxTreeNode struct {
	Low, High    [3]float64
	First, Count int32
}

// IsLeaf is whether the node holds a box and not two children.
func (node BoxTreeNode) IsLeaf() bool { return node.Count != 0 }

// BoxTree is the nodes of the tree, the root first.
type BoxTree struct {
	Nodes []BoxTreeNode
}

// What a tree is refused with: no boxes, boxes that are not boxes, and a
// stack too small to walk any tree over so many boxes.
var (
	ErrNoBoxes    = errors.New("strategies: a box tree needs at least one box")
	ErrBoxShape   = errors.New("strategies: a box needs finite corners, the low no higher than the high")
	ErrBoxStack   = errors.New("strategies: the stack is too small to walk a tree over so many boxes")
	errBoxCounted = errors.New("strategies: a box tree is built over fewer boxes than a word holds")
)

// BuildBoxTree is the tree over boxes, which a walker of a stack of this
// many nodes can walk. A leaf's box is its index in boxes. It refuses no
// boxes, a box that is not finite or is inside out, and a stack too small
// for the boxes, whose tree is at least log2 of their number deep.
func BuildBoxTree(boxes []TreeBox, stack int) (BoxTree, error) {
	if len(boxes) == 0 {
		return BoxTree{}, ErrNoBoxes
	}
	if len(boxes) > math.MaxInt32/2 {
		return BoxTree{}, errBoxCounted
	}
	levels := bits.Len(uint(len(boxes)))
	if stack < levels+walkSpare {
		return BoxTree{}, fmt.Errorf("%w: %d boxes in a stack of %d",
			ErrBoxStack, len(boxes), stack)
	}
	builder := boxBuilder{boxes: boxes, deepest: stack - walkSpare - levels,
		middles: make([][boxAxes]float64, len(boxes)),
		nodes:   make([]BoxTreeNode, 2*len(boxes)-1), used: 1}
	items := make([]int, len(boxes))
	for index, box := range boxes {
		for axis := range boxAxes {
			if !finite(box.Low[axis]) || !finite(box.High[axis]) ||
				box.Low[axis] > box.High[axis] {
				return BoxTree{}, fmt.Errorf("%w: box %d", ErrBoxShape,
					index)
			}
			builder.middles[index][axis] = (box.Low[axis] + box.High[axis]) / 2
		}
		items[index] = index
	}
	builder.grow(items, 0, 0)
	return BoxTree{Nodes: builder.nodes}, nil
}

// finite is whether a number is neither an infinity nor a nan.
func finite(number float64) bool {
	return !math.IsInf(number, 0) && !math.IsNaN(number)
}

// boxBuilder is a tree being grown: the boxes and their middles, the nodes
// so far and how many are used, and how deep the heuristic may cut before
// the median takes over.
type boxBuilder struct {
	boxes   []TreeBox
	middles [][boxAxes]float64
	nodes   []BoxTreeNode
	used    int
	deepest int
}

// grow puts the node of a run of boxes at a slot, and below it the nodes of
// the two halves it is cut in, at the next two free slots.
func (builder *boxBuilder) grow(items []int, slot, depth int) {
	node := builder.around(items)
	if len(items) == 1 {
		node.First, node.Count = int32(items[0]), 1
		builder.nodes[slot] = node
		return
	}
	left := builder.cut(items, depth)
	first := builder.used
	builder.used += 2
	node.First = int32(first)
	builder.nodes[slot] = node
	builder.grow(items[:left], first, depth+1)
	builder.grow(items[left:], first+1, depth+1)
}

// around is the node whose box holds every box of a run.
func (builder *boxBuilder) around(items []int) BoxTreeNode {
	first := builder.boxes[items[0]]
	node := BoxTreeNode{Low: first.Low, High: first.High}
	for _, item := range items[1:] {
		box := builder.boxes[item]
		for axis := range boxAxes {
			node.Low[axis] = min(node.Low[axis], box.Low[axis])
			node.High[axis] = max(node.High[axis], box.High[axis])
		}
	}
	return node
}

// cut puts a run of more than one box in two, in place, and says how many
// are on the left: by the heuristic while the tree is shallow enough, and
// where it finds no cut that parts the run, or the tree is deep, at the
// median of the longest side of the middles.
func (builder *boxBuilder) cut(items []int, depth int) int {
	low, high := builder.spanOf(items)
	if depth < builder.deepest {
		if left := builder.binned(items, low, high); left > 0 &&
			left < len(items) {
			return left
		}
	}
	return builder.median(items, longest(low, high))
}

// spanOf is the box of the middles of a run of boxes.
func (builder *boxBuilder) spanOf(items []int) (low, high [boxAxes]float64) {
	low, high = builder.middles[items[0]], builder.middles[items[0]]
	for _, item := range items[1:] {
		for axis := range boxAxes {
			low[axis] = min(low[axis], builder.middles[item][axis])
			high[axis] = max(high[axis], builder.middles[item][axis])
		}
	}
	return low, high
}

// longest is the axis along which a box is longest, the lowest where two
// are as long.
func longest(low, high [boxAxes]float64) int {
	found := 0
	for axis := 1; axis < boxAxes; axis++ {
		if high[axis]-low[axis] > high[found]-low[found] {
			found = axis
		}
	}
	return found
}

// median puts the run in two halves by the middles along an axis, the lower
// first, boxes with one middle by their index so the halves are the same on
// every run, and says how many are on the left.
func (builder *boxBuilder) median(items []int, axis int) int {
	sort.SliceStable(items, func(one, other int) bool {
		a, b := builder.middles[items[one]][axis],
			builder.middles[items[other]][axis]
		if a != b {
			return a < b
		}
		return items[one] < items[other]
	})
	return len(items) / 2
}

// binCut is the cheapest cut found so far: along which axis, after which
// bin, and what it costs. A cost of infinity is none found.
type binCut struct {
	axis, after int
	cost        float64
}

// binned is how many boxes of a run are left of its cheapest cut among the
// bins of the middles on every axis, the run put in two by it, in place; or
// nought when every middle is the same and no cut parts them.
func (builder *boxBuilder) binned(items []int, low, high [boxAxes]float64) int {
	best := binCut{cost: math.Inf(1)}
	for axis := range boxAxes {
		if high[axis] > low[axis] {
			best = builder.cheaper(best, items, axis, low[axis], high[axis])
		}
	}
	if math.IsInf(best.cost, 1) {
		return 0
	}
	left := 0
	for index := range items {
		if binOf(builder.middles[items[index]][best.axis], low[best.axis],
			high[best.axis]) <= best.after {
			items[left], items[index] = items[index], items[left]
			left++
		}
	}
	return left
}

// binOf is the bin a middle falls in along an axis of the middles' box.
func binOf(middle, low, high float64) int {
	placed := int((middle - low) / (high - low) * boxBins)
	return min(boxBins-1, max(0, placed))
}

// cheaper is the cheaper of a cut so far and the cheapest along one axis:
// the surface area of each side's box times its boxes, summed.
func (builder *boxBuilder) cheaper(best binCut, items []int, axis int,
	low, high float64) binCut {
	var counts [boxBins]int
	var around [boxBins]BoxTreeNode
	for _, item := range items {
		bin := binOf(builder.middles[item][axis], low, high)
		box := builder.boxes[item]
		if counts[bin] == 0 {
			around[bin] = BoxTreeNode{Low: box.Low, High: box.High}
		} else {
			around[bin] = merged(around[bin], box)
		}
		counts[bin]++
	}
	var rightCost [boxBins]float64
	var right BoxTreeNode
	rightCount := 0
	for bin := boxBins - 1; bin > 0; bin-- {
		if counts[bin] > 0 {
			right = enclosing(right, around[bin], rightCount)
			rightCount += counts[bin]
		}
		rightCost[bin] = float64(rightCount) * area(right, rightCount)
	}
	var left BoxTreeNode
	leftCount := 0
	for bin := 0; bin < boxBins-1; bin++ {
		if counts[bin] > 0 {
			left = enclosing(left, around[bin], leftCount)
			leftCount += counts[bin]
		}
		if leftCount == 0 || leftCount == len(items) {
			continue
		}
		cost := float64(leftCount)*area(left, leftCount) + rightCost[bin+1]
		if cost < best.cost {
			best = binCut{axis: axis, after: bin, cost: cost}
		}
	}
	return best
}

// merged is a node's box grown to hold a box.
func merged(node BoxTreeNode, box TreeBox) BoxTreeNode {
	for axis := range boxAxes {
		node.Low[axis] = min(node.Low[axis], box.Low[axis])
		node.High[axis] = max(node.High[axis], box.High[axis])
	}
	return node
}

// enclosing is the box of a run of boxes held so far, counted, grown to
// hold another node's box; the first takes that box as it is.
func enclosing(held, next BoxTreeNode, counted int) BoxTreeNode {
	if counted == 0 {
		return next
	}
	return merged(held, TreeBox{Low: next.Low, High: next.High})
}

// area is the surface area of a node's box, or nought for no boxes held.
func area(node BoxTreeNode, counted int) float64 {
	if counted == 0 {
		return 0
	}
	x := node.High[0] - node.Low[0]
	y := node.High[1] - node.Low[1]
	z := node.High[2] - node.Low[2]
	return 2 * (x*y + y*z + z*x)
}
