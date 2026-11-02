package mesh

import (
	"cmp"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// the tree's shape: a leaf of few faces is never cut, one of more than
// most always is, and between, the surface area heuristic decides,
// choosing among bins on each axis. deepest is as deep as it goes.
const (
	leafFew  = 2
	leafMost = 16
	bins     = 16
	deepest  = 64
)

// node is a box of the tree, its bounds low then high, and either a run of the
// order, when count is greater than zero, or two children, at first and the one
// after it.
type node struct {
	bounds       [2][3]float64
	first, count int
}

// box is the node's box as a maths.Box.
func (at node) box() maths.Box {
	return maths.Box{Min: vectorOf(at.bounds[0]),
		Max: vectorOf(at.bounds[1])}
}

// tree is the nodes, root first, and the faces in the order the leaves
// hold them.
type tree struct {
	nodes []node
	order []int
}

// run is the faces of a leaf.
func (grown tree) run(at *node) []int {
	return grown.order[at.first : at.first+at.count]
}

// growing is a tree being cut, with each face's box and middle.
type growing struct {
	tree    *tree
	spans   []span
	middles [][3]float64
}

// grown is the tree over a mesh's faces, cut by the surface area
// heuristic over binned middles: wald, "on fast construction of
// sah-based bounding volume hierarchies" (2007).
func grown(points []maths.Vec, faces []face) tree {
	made := tree{order: make([]int, len(faces)),
		nodes: make([]node, 1, 2*len(faces))}
	grow := growing{tree: &made, spans: make([]span, len(faces)),
		middles: make([][3]float64, len(faces))}
	for i, at := range faces {
		grow.spans[i] = spanOf(points, at)
		grow.middles[i] = grow.spans[i].middle()
		made.order[i] = i
	}
	grow.cut(0, 0, len(faces), 0)
	return made
}

// cut makes node at the box over a run of the order, and a leaf of it
// or two children, cut where the heuristic says.
func (grow *growing) cut(at, from, to, depth int) {
	around := grow.around(from, to)
	grow.tree.nodes[at].bounds[0] = around.low
	grow.tree.nodes[at].bounds[1] = around.high
	count := to - from
	if count <= leafFew || depth >= deepest {
		grow.leaf(at, from, to)
		return
	}
	axis, split, cost := grow.best(from, to, around)
	if !(cost < float64(count)) && count <= leafMost {
		grow.leaf(at, from, to)
		return
	}
	half := grow.parted(from, to, axis, split)
	left := len(grow.tree.nodes)
	grow.tree.nodes = append(grow.tree.nodes, node{}, node{})
	grow.tree.nodes[at].first = left
	grow.cut(left, from, half, depth+1)
	grow.cut(left+1, half, to, depth+1)
}

// leaf makes node at a leaf of a run of the order.
func (grow *growing) leaf(at, from, to int) {
	grow.tree.nodes[at].first, grow.tree.nodes[at].count = from, to-from
}

// around is the box over a run's faces.
func (grow *growing) around(from, to int) span {
	made := emptySpan()
	for _, i := range grow.tree.order[from:to] {
		made = made.joined(grow.spans[i])
	}
	return made
}

// middling is the box over the middles of a run's faces.
func (grow *growing) middling(from, to int) span {
	made := emptySpan()
	for _, i := range grow.tree.order[from:to] {
		made = made.with(grow.middles[i])
	}
	return made
}

// best is the cheapest cut of a run: its axis, the last bin of its
// left side, and its cost, one for the node and each side's faces
// weighed by its share of the box's area. with no axis to cut along,
// the cost is endless.
func (grow *growing) best(from, to int, around span) (int, int, float64) {
	middles := grow.middling(from, to)
	axis, split, cost := 0, 0, math.Inf(1)
	whole := around.area()
	for along := 0; along < 3; along++ {
		if !(middles.high[along] > middles.low[along]) {
			continue
		}
		at, costs := grow.binned(from, to, along, middles, whole)
		if costs < cost {
			axis, split, cost = along, at, costs
		}
	}
	return axis, split, cost
}

// binned is the cheapest cut along one axis, among the bins: the last
// bin of its left side, and its cost.
func (grow *growing) binned(from, to, axis int, middles span,
	whole float64) (int, float64) {
	var counts [bins]int
	var boxes [bins]span
	for i := range boxes {
		boxes[i] = emptySpan()
	}
	for _, i := range grow.tree.order[from:to] {
		at := binOf(grow.middles[i][axis], middles, axis)
		counts[at]++
		boxes[at] = boxes[at].joined(grow.spans[i])
	}
	var rightAreas [bins]float64
	var rightCounts [bins]int
	right, many := emptySpan(), 0
	for i := bins - 1; i > 0; i-- {
		right, many = right.joined(boxes[i]), many+counts[i]
		rightAreas[i], rightCounts[i] = right.area(), many
	}
	split, cost := 0, math.Inf(1)
	left, few := emptySpan(), 0
	for i := 0; i < bins-1; i++ {
		left, few = left.joined(boxes[i]), few+counts[i]
		if few == 0 || rightCounts[i+1] == 0 {
			continue
		}
		costs := 1 + (left.area()*float64(few)+
			rightAreas[i+1]*float64(rightCounts[i+1]))/whole
		if costs < cost {
			split, cost = i, costs
		}
	}
	return split, cost
}

// binOf is the bin a middle falls in along an axis of the middles' box:
// the first, when the box has no width along it.
func binOf(middle float64, middles span, axis int) int {
	wide := middles.high[axis] - middles.low[axis]
	if !(wide > 0) {
		return 0
	}
	share := (middle - middles.low[axis]) / wide
	return min(bins-1, max(0, int(share*bins)))
}

// parted puts a run's faces in bins up to split before the rest,
// keeping each side's order, and is where the right side starts. if
// one side would be empty, the run is put in order of its middles
// along the axis and cut in half.
func (grow *growing) parted(from, to, axis, split int) int {
	middles := grow.middling(from, to)
	run := grow.tree.order[from:to]
	leftward := func(i int) bool {
		return binOf(grow.middles[i][axis], middles, axis) <= split
	}
	sides := slices.Clone(run)
	at := 0
	for _, i := range sides {
		if leftward(i) {
			run[at] = i
			at++
		}
	}
	half := from + at
	for _, i := range sides {
		if !leftward(i) {
			run[at] = i
			at++
		}
	}
	if half == from || half == to {
		slices.SortStableFunc(run, func(one, other int) int {
			return cmp.Compare(grow.middles[one][axis],
				grow.middles[other][axis])
		})
		half = from + len(run)/2
	}
	return half
}
