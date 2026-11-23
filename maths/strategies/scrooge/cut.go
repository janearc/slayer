package scrooge

import (
	"container/heap"
	"errors"
)

// A cut is the slice across a set's tree that a picture is drawn from:
// every node on it is either opened, and stood for by its children, or
// taken as its aggregate.
//
// It starts as the set's top nodes and is refined
// by splitting the node that matters most first, until no node on it
// should be opened or its budget is spent. This is the cut of lightcuts
// (Walter and others, 2005) and the split queue of ROAM (Duchaineau and
// others, 1997).
//
// When the nodes it holds pass its budget, the cut forgets: the opened
// node that matters least is closed, its children let go, and it stands
// as its aggregate again. A node's children are made from its seed, so a
// forgotten node opened later gives the same children. This is the
// forgetting of simplified memory-bounded A* (Russell, 1992).

// Asker is who a cut is refined for: where each node's footprint comes
// from, and how much the node matters to the picture.
type Asker interface {
	// Footprint is the cone a node is judged by.
	Footprint(node Node) Footprint
	// Worth is how much a node could change the picture: nought for one
	// the picture cannot show, such as one behind closed pixels. A node
	// worth nought is never opened.
	Worth(node Node) float64
}

// Budget is the most a cut may hold.
type Budget struct {
	// Nodes is the most nodes the cut holds at once, opened and not;
	// nought is no limit.
	Nodes int
	// Opens is the most nodes it opens in one refining; nought is no
	// limit.
	Opens int
}

// Spent is what a refining did.
type Spent struct {
	Opened, Forgotten, Held int
	// Short is whether the budget ran out with nodes still worth
	// opening.
	Short bool
}

// held is a node on the cut, with the node it was opened from and how
// many of its children are held.
type held struct {
	node     Node
	parent   *held
	worth    float64
	children []*held
	opened   bool
	place    int // its place in the queue it is in, or -1
}

// Cut is a slice across a tree of nodes.
type Cut struct {
	asker  Asker
	budget Budget
	open   queue // leaves worth opening, most first
	shut   queue // opened nodes whose children are all leaves, least first
	leaves map[*held]struct{}
	count  int
}

// ErrNoAsker is the refusal of a cut made for nobody.
var ErrNoAsker = errors.New("scrooge: a cut is refined for an asker")

// NewCut is the cut of a set's top nodes for an asker, under a budget.
func NewCut(top []Node, asker Asker, budget Budget) (*Cut, error) {
	if asker == nil {
		return nil, ErrNoAsker
	}
	cut := &Cut{asker: asker, budget: budget,
		open: queue{most: true}, shut: queue{},
		leaves: map[*held]struct{}{}}
	for _, node := range top {
		cut.add(node, nil)
	}
	return cut, nil
}

// add puts a node on the cut as a leaf under a parent.
func (cut *Cut) add(node Node, parent *held) *held {
	one := &held{node: node, parent: parent, place: -1,
		worth: cut.asker.Worth(node)}
	cut.leaves[one] = struct{}{}
	cut.count++
	if one.worth > 0 && Resolve(node, cut.asker.Footprint(node)) == Open {
		heap.Push(&cut.open, one)
	}
	return one
}

// Refine opens the node that matters most, again and again, until none
// on the cut should be opened or the budget is spent, forgetting the
// opened node that matters least whenever the cut holds too many.
func (cut *Cut) Refine() Spent {
	var spent Spent
	for cut.open.Len() > 0 {
		if cut.budget.Opens > 0 && spent.Opened >= cut.budget.Opens {
			spent.Short = true
			break
		}
		one := heap.Pop(&cut.open).(*held)
		cut.unfold(one)
		spent.Opened++
		for cut.over() {
			// a node is forgotten only for one that matters more: where
			// nothing held matters less than the node just opened, that
			// node is closed again and the refining is over
			lesser := cut.least(one)
			if lesser == nil || lesser.worth >= one.worth {
				if one.opened {
					cut.shutUp(one)
					spent.Forgotten++
				}
				spent.Short = true
				break
			}
			cut.shutUp(lesser)
			spent.Forgotten++
		}
		if spent.Short {
			break
		}
	}
	spent.Held = cut.count
	return spent
}

// over is whether the cut holds more nodes than its budget.
func (cut *Cut) over() bool {
	return cut.budget.Nodes > 0 && cut.count > cut.budget.Nodes
}

// least is the opened node that matters least and may be forgotten,
// other than one to keep, or nil where there is none.
func (cut *Cut) least(keep *held) *held {
	var found *held
	// the least of a heap is its first, or, with the first kept, the
	// lesser of the first's two children
	for place := 0; place < len(cut.shut.items) && place < 3; place++ {
		one := cut.shut.items[place]
		if one != keep && (found == nil || one.worth < found.worth) {
			found = one
		}
		if place == 0 && one != keep {
			break
		}
	}
	return found
}

// unfold opens a leaf: its children join the cut and it stands for them.
func (cut *Cut) unfold(one *held) {
	delete(cut.leaves, one)
	one.opened = true
	for child := range one.node.Detail(cut.asker.Footprint(one.node)) {
		one.children = append(one.children, cut.add(child, one))
	}
	if len(one.children) == 0 {
		// it gave nothing finer: it stands as its aggregate after all
		one.opened = false
		cut.leaves[one] = struct{}{}
		return
	}
	if one.parent != nil && one.parent.place >= 0 {
		heap.Remove(&cut.shut, one.parent.place)
	}
	heap.Push(&cut.shut, one)
}

// shutUp forgets an opened node whose children are all leaves: they are
// let go, and it stands as its aggregate again.
func (cut *Cut) shutUp(one *held) {
	if one.place >= 0 {
		heap.Remove(&cut.shut, one.place)
	}
	for _, child := range one.children {
		delete(cut.leaves, child)
		if child.place >= 0 {
			heap.Remove(&cut.open, child.place)
		}
		cut.count--
	}
	one.children, one.opened = nil, false
	cut.leaves[one] = struct{}{}
	if parent := one.parent; parent != nil && parent.settled() {
		heap.Push(&cut.shut, parent)
	}
}

// settled is whether an opened node's children are all leaves, so that
// it may be forgotten.
func (one *held) settled() bool {
	if !one.opened || one.place >= 0 {
		return false
	}
	for _, child := range one.children {
		if child.opened {
			return false
		}
	}
	return true
}

// Leaves is the cut itself: every node a picture is drawn from, each as
// its aggregate.
func (cut *Cut) Leaves() []Node {
	nodes := make([]Node, 0, len(cut.leaves))
	for one := range cut.leaves {
		nodes = append(nodes, one.node)
	}
	return nodes
}

// queue is nodes ordered by worth: most first where most is set, least
// first where not. It keeps each node's place in it.
type queue struct {
	items []*held
	most  bool
}

// Len is how many items are waiting.
func (q *queue) Len() int { return len(q.items) }

// Less orders the queue by worth: the greatest first when the queue is for
// the most, the least first when it is not.
func (q *queue) Less(i, j int) bool {
	if q.most {
		return q.items[i].worth > q.items[j].worth
	}
	return q.items[i].worth < q.items[j].worth
}

// Swap exchanges two items and keeps each one's recorded place.
func (q *queue) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
	q.items[i].place, q.items[j].place = i, j
}

// Push adds an item at the end and records its place.
func (q *queue) Push(item any) {
	one := item.(*held)
	one.place = len(q.items)
	q.items = append(q.items, one)
}

// Pop removes the last item and returns it, with its place cleared.
func (q *queue) Pop() any {
	last := len(q.items) - 1
	one := q.items[last]
	q.items = q.items[:last]
	one.place = -1
	return one
}
