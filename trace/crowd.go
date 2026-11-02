package trace

import (
	"cmp"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// crowd is a picture's things with a tree of boxes over them, so that a
// ray asks only the things whose boxes it passes through. It is exact:
// a ray finds what asking every thing in order finds, and an earlier
// thing wins a tie.
//
// The tree trusts each shape's Bounds: a shape met outside its own box
// is lost here, where asking every thing would find it.
type crowd struct {
	things []placedThing
	// boxes is each thing's box in the world
	boxes []maths.Box
	// nodes is the tree, its root first, and order the things' places in
	// the order its leaves hold them
	nodes []branch
	order []int
	// everywhere is the things with no edge, asked by every ray
	everywhere []int
}

// branch is a node of the tree: a box around everything under it, and either
// two children or a run of the order. The root is never a child, so a left of
// zero is a leaf.
//
// shadows is whether anything under it casts a shadow; a shadow ray passes over
// a branch of only clear things without asking its box.
type branch struct {
	box         maths.Box
	left, right int
	from, to    int
	shadows     bool
}

// leafSize is the most things a leaf holds before it is split.
const leafSize = 4

// deepest is the most branches a walk of the tree holds at once; a tree
// split at the middle is far shallower for any count of things.
const deepest = 128

// gathered is a picture's things with the tree built over them.
func gathered(things []placedThing) crowd {
	made := crowd{things: things, boxes: make([]maths.Box, len(things))}
	for i := range things {
		made.boxes[i] = worldBox(&things[i])
		if edged(made.boxes[i]) {
			made.order = append(made.order, i)
		} else {
			made.everywhere = append(made.everywhere, i)
		}
	}
	if len(made.order) > 0 {
		made.build(0, len(made.order))
	}
	return made
}

// build makes the branch over a run of the order, split at the middle
// of the things along the way they spread most, and is its place.
func (made *crowd) build(from, to int) int {
	at := len(made.nodes)
	made.nodes = append(made.nodes, branch{box: made.around(from, to),
		from: from, to: to, shadows: made.shadowing(from, to)})
	if to-from <= leafSize {
		return at
	}
	axis := widest(made.middles(from, to))
	slices.SortStableFunc(made.order[from:to], func(one, other int) int {
		return cmp.Compare(middle(made.boxes[one], axis),
			middle(made.boxes[other], axis))
	})
	half := from + (to-from)/2
	left := made.build(from, half)
	right := made.build(half, to)
	made.nodes[at].left, made.nodes[at].right = left, right
	return at
}

// around is the box around the things of a run of the order.
func (made *crowd) around(from, to int) maths.Box {
	box := emptyBox()
	for _, index := range made.order[from:to] {
		box = joined(box, made.boxes[index])
	}
	return box
}

// shadowing is whether any thing of a run of the order casts a shadow.
func (made *crowd) shadowing(from, to int) bool {
	for _, index := range made.order[from:to] {
		if !made.things[index].thing.Clear() {
			return true
		}
	}
	return false
}

// middles is the box around the middles of the things of a run.
func (made *crowd) middles(from, to int) maths.Box {
	box := emptyBox()
	for _, index := range made.order[from:to] {
		around := made.boxes[index]
		centre := around.Min.Add(around.Max).Scale(0.5)
		box = joined(box, maths.Box{Min: centre, Max: centre})
	}
	return box
}

// nearest is the best a search has found: a thing's place and how far along the
// ray it is met, or a place less than zero for nothing yet, and where on a mesh
// of slayer's it landed.
type nearest struct {
	index int
	dist  float64
	on    landing
}

// first is the nearest thing the ray meets past near, and whether it
// meets any.
func (seen crowd) first(ray maths.Ray, near float64) (hit, bool) {
	best := nearest{index: -1, dist: nowhere}
	for _, index := range seen.everywhere {
		best = seen.closer(best, index, ray, near)
	}
	var stack [deepest]int
	top := seen.rooted(&stack)
	for top > 0 {
		top--
		node := &seen.nodes[stack[top]]
		if _, _, ok := node.box.Hit(ray, near, best.dist); !ok {
			continue
		}
		if node.left == 0 {
			for _, index := range seen.order[node.from:node.to] {
				best = seen.closer(best, index, ray, near)
			}
			continue
		}
		stack[top], stack[top+1] = node.right, node.left
		top += 2
	}
	if best.index < 0 {
		return hit{dist: nowhere}, false
	}
	return hit{thing: seen.things[best.index], dist: best.dist,
		on: best.on}, true
}

// rooted puts the root on a walk's stack, if there is a tree, and is
// how many branches the stack holds.
func (seen crowd) rooted(stack *[deepest]int) int {
	if len(seen.nodes) == 0 {
		return 0
	}
	stack[0] = 0
	return 1
}

// closer is the nearer of the best so far and one thing, as asking
// every thing in order decides it: a nearer thing wins, and at the same
// distance the earlier one. The thing is asked up to just past the best,
// so one met at the best's own distance is seen.
func (seen crowd) closer(best nearest, index int, ray maths.Ray,
	near float64) nearest {
	far := best.dist
	if far < nowhere {
		far = math.Nextafter(far, math.Inf(1))
	}
	along, on, ok := seen.things[index].reach(ray, near, far)
	if !ok {
		return best
	}
	if along < best.dist || along == best.dist && index < best.index {
		return nearest{index: index, dist: along, on: on}
	}
	return best
}

// blocked is whether anything that casts a shadow stands between a
// point and a lamp far along the ray; far is +Inf for a lamp at
// infinity.
func (seen crowd) blocked(ray maths.Ray, far float64) bool {
	far = math.Min(far, nowhere)
	for _, index := range seen.everywhere {
		if seen.casts(index, ray, far) {
			return true
		}
	}
	var stack [deepest]int
	top := seen.rooted(&stack)
	for top > 0 {
		top--
		node := &seen.nodes[stack[top]]
		if !node.shadows {
			continue
		}
		if _, _, ok := node.box.Hit(ray, 0, far); !ok {
			continue
		}
		if node.left == 0 {
			if seen.castsAny(node, ray, far) {
				return true
			}
			continue
		}
		stack[top], stack[top+1] = node.right, node.left
		top += 2
	}
	return false
}

// castsAny is whether any thing of a leaf stands in the ray's way.
func (seen crowd) castsAny(leaf *branch, ray maths.Ray, far float64) bool {
	for _, index := range seen.order[leaf.from:leaf.to] {
		if seen.casts(index, ray, far) {
			return true
		}
	}
	return false
}

// casts is whether one thing stands in the ray's way: a clear thing
// never does.
func (seen crowd) casts(index int, ray maths.Ray, far float64) bool {
	placed := &seen.things[index]
	if placed.thing.Clear() {
		return false
	}
	_, ok := placed.meet(ray, 0, far)
	return ok
}
