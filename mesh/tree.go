package mesh

import (
	"cmp"
	"math"
	"slices"
	"sync"

	"github.com/janearc/slayer/maths"
)

// the tree's shape: a leaf of few faces is never cut, one of more than
// most always is, and between, the surface area heuristic decides,
// choosing among bins on each axis. deepest is as deep as it goes.
const (
	// defined now by the trees built so far: a run of this many faces
	// or fewer is a leaf.
	leafFew = 2
	// defined now by the trees built so far: a run of more than this
	// many faces is always cut.
	leafMost = 16
	// defined now by the trees built so far: the bins a run's middles
	// are put in along an axis.
	bins = 16
	// defined now by the trees built so far: the depth no cut goes past.
	deepest = 64
	// defined now by the trees built so far: a side of a cut with this many
	// faces or more is worth a goroutine of its own; a smaller one is cut
	// by the goroutine that cut its parent.
	parallelFaces = 2048
)

// node is a box of the tree, its bounds low then high, and either a run of the
// order, when count is greater than zero, or two children, at first and the one
// after it.
type node struct {
	bounds       [boundEnds][axes]float64
	first, count int
}

// box is the node's box as a maths.Box.
func (at node) box() maths.Box {
	return maths.NewBox(vectorOf(at.bounds[low]),
		vectorOf(at.bounds[high]))
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

// growing is a tree being cut, with each face's box and middle. spare is
// the places left to cut a side of a cut on another goroutine, each held
// by a value sent into it while the side is cut; nil when the tree is cut
// by one goroutine alone. The spans and middles are only read, and each
// goroutine writes only its own run of the order and its own nodes.
type growing struct {
	tree    *tree
	spans   []span
	middles [][axes]float64
	spare   chan struct{}
}

// grown is the tree over a mesh's faces, cut by the surface area
// heuristic over binned middles: wald, "on fast construction of
// sah-based bounding volume hierarchies" (2007).
func grown(points []maths.Vec, faces []face) tree {
	return grownBy(points, faces, nil)
}

// grownBy is grown with large sides of its cuts handed to other
// goroutines while spare has room. The tree is the same one grown makes:
// the same boxes in the same places, and the same order of the faces.
func grownBy(points []maths.Vec, faces []face, spare chan struct{}) tree {
	made := tree{order: make([]int, len(faces)),
		nodes: make([]node, unity, pair*len(faces))}
	grow := growing{tree: &made, spans: make([]span, len(faces)),
		middles: make([][axes]float64, len(faces)), spare: spare}
	for i, at := range faces {
		grow.spans[i] = spanOf(points, at)
		grow.middles[i] = grow.spans[i].middle()
		made.order[i] = i
	}
	grow.cut(nought, nought, len(faces), nought)
	// Construction reserves the binary tree's worst case. Most meshes use
	// far fewer nodes; do not retain that unused capacity for the whole film.
	// Copying preserves every node, its order, and every intersection result.
	if len(made.nodes) < cap(made.nodes) {
		made.nodes = slices.Clone(made.nodes)
	}
	return made
}

// cut makes node at the box over a run of the order, and a leaf of it
// or two children, cut where the heuristic says.
func (grow *growing) cut(at, from, to, depth int) {
	around := grow.around(from, to)
	grow.tree.nodes[at].bounds[low] = around.low
	grow.tree.nodes[at].bounds[high] = around.high
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
	if grow.lends(to - half) {
		grow.cutApart(left, from, half, to, depth+unity)
		return
	}
	grow.cut(left, from, half, depth+unity)
	grow.cut(left+unity, half, to, depth+unity)
}

// lends is whether a side of a cut of this many faces is to be cut on
// another goroutine, and takes a place of spare for it if so. A tree cut
// with no spare is never lent: a send on a nil channel is never ready.
func (grow *growing) lends(faces int) bool {
	if faces < parallelFaces {
		return false
	}
	select {
	case grow.spare <- struct{}{}:
		return true
	default:
		return false
	}
}

// cutApart cuts the two sides of a cut, nodes left and the one after it,
// the right side on another goroutine and the left on this one, and puts
// the right's nodes
// where cutting it here would have put them: after every node of the
// left side's. lends has taken the place the right goes by.
func (grow *growing) cutApart(left, from, half, to, depth int) {
	var right []node
	var done sync.WaitGroup
	done.Add(unity)
	go func() {
		defer done.Done()
		defer func() { <-grow.spare }()
		right = grow.apart(half, to, depth)
	}()
	grow.cut(left, from, half, depth)
	done.Wait()
	grow.splice(left+unity, right)
}

// apart cuts a run of the order into nodes of its own, the run's root
// first and each branch naming its children by their place among them.
func (grow *growing) apart(from, to, depth int) []node {
	own := tree{order: grow.tree.order, nodes: make([]node, unity)}
	sub := growing{tree: &own, spans: grow.spans, middles: grow.middles,
		spare: grow.spare}
	sub.cut(nought, from, to, depth)
	return own.nodes
}

// splice puts nodes cut apart into the tree: the first as node at, and the
// rest after every node there. A node cut apart numbers its children from
// its own first node, which stands where the tree's next node does less
// one, as the first of them is the one already in the tree.
func (grow *growing) splice(at int, part []node) {
	shift := len(grow.tree.nodes) - unity
	for i := range part {
		if part[i].count == nought {
			part[i].first += shift
		}
	}
	grow.tree.nodes[at] = part[nought]
	grow.tree.nodes = append(grow.tree.nodes, part[unity:]...)
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
	axis, split, cost := nought, nought, math.Inf(plusInfinity)
	whole := around.area()
	for along := nought; along < axes; along++ {
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
	right, many := emptySpan(), nought
	for i := bins - unity; i > nought; i-- {
		right, many = right.joined(boxes[i]), many+counts[i]
		rightAreas[i], rightCounts[i] = right.area(), many
	}
	split, cost := nought, math.Inf(plusInfinity)
	left, few := emptySpan(), nought
	for i := nought; i < bins-unity; i++ {
		left, few = left.joined(boxes[i]), few+counts[i]
		if few == nought || rightCounts[i+unity] == nought {
			continue
		}
		costs := unity + (left.area()*float64(few)+
			rightAreas[i+unity]*float64(rightCounts[i+unity]))/whole
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
	if !(wide > nought) {
		return nought
	}
	share := (middle - middles.low[axis]) / wide
	placed := int(share * bins)
	return min(bins-unity, max(nought, placed))
}

// parting is a run being put in two, by the bins of its faces' middles
// along an axis: the growing tree, the box over the middles, the axis,
// and the last bin of the left side.
type parting struct {
	grow    *growing
	middles span
	axis    int
	split   int
}

// leftward is whether face i falls in the bins of the left side.
func (cutting parting) leftward(i int) bool {
	return binOf(cutting.grow.middles[i][cutting.axis], cutting.middles,
		cutting.axis) <= cutting.split
}

// compare orders two faces by their middles along the axis.
func (cutting parting) compare(one, other int) int {
	return cmp.Compare(cutting.grow.middles[one][cutting.axis],
		cutting.grow.middles[other][cutting.axis])
}

// parted puts a run's faces in bins up to split before the rest,
// keeping each side's order, and is where the right side starts. if
// one side would be empty, the run is put in order of its middles
// along the axis and cut in half.
func (grow *growing) parted(from, to, axis, split int) int {
	cutting := parting{grow: grow, middles: grow.middling(from, to),
		axis: axis, split: split}
	run := grow.tree.order[from:to]
	sides := slices.Clone(run)
	at := nought
	for _, i := range sides {
		if cutting.leftward(i) {
			run[at] = i
			at++
		}
	}
	half := from + at
	for _, i := range sides {
		if !cutting.leftward(i) {
			run[at] = i
			at++
		}
	}
	if half == from || half == to {
		slices.SortStableFunc(run, cutting.compare)
		half = from + len(run)/pair
	}
	return half
}
