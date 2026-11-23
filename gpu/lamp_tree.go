package gpu

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The lamp tree of a world with many lamps. Above lampsEvery lamps the
// kernel does not price every lamp at every point it shades: it walks
// down a tree of the lamps that stand somewhere, as the tracer does
// (trace/lamptree.go), taking lampsAsked picks at a surface, each counting
// one over the chance of its walk.
//
// At a step through a medium it splits the
// tree into at most lampSplitMost nodes where the point is inside a node or
// near it for its size, and takes one lamp from each (strategies.Cut).
//
// The tree is the shared one, strategies.BuildLightTree,
// built here over the world's lamps and read by the kernel in the layout
// restir's tree already has (restir_tree.go): nodeFloats floats a node.
//
// The tree and the lamps outside it are kept in the lamps' buffer, after
// the rows of the lamps themselves: the nodes, root first, as floats, and
// then each outside lamp's index as a float. The header's lamp_tree says
// how many of each. A world of lampsEvery lamps or fewer has none, and the
// kernel lights it lamp by lamp as it always has.
//
// This file has no Metal in it. The kernel's walk is lamp_tree_walk in
// trace.metal. It differs from restir's walk, which is weighed by a normal
// the point faces and has none for a point in a medium: this one is the
// tracer's weighing (a millimetre squared at least from a node, and half
// the box's diagonal for a point inside it), in float32.

// lampTreeLeast is the fewest standing lamps a world must have for a tree
// to be built: trace's own, so a world is given a tree when the tracer
// gives its scene one.
const lampTreeLeast = lampsEvery

// standingLamp is a lamp at a place whose light falls off by the inverse
// square, with an upper bound on the light it sends: trace's standingLamp.
type standingLamp interface {
	Stands() maths.Vec
	AtMost(distance float64) float64
}

// lampCrowd is the tree the kernel walks and the lamps it does not hold.
// The tree's leaves number the lamps by their index among the world's
// lamps. A crowd with no nodes is no tree.
type lampCrowd struct {
	nodes []lightNode
	// others is the index of every lamp that is not in the tree: the
	// lamps that stand nowhere (a sun, the beams, the polar lamps), and
	// a standing lamp whose light is not a positive float32.
	others []int32
}

// lampCrowdOf is the crowd of a world's lamps, chosen as trace's treeOf
// chooses: a lamp that stands somewhere is in the tree by its brightest
// part a unit away, one that sends no light is left out of both, and
// every other lamp is outside. Too few standing lamps are no tree, and
// then no lamp is outside either, for the kernel prices them all.
func lampCrowdOf(lamps []thing.Lamp) lampCrowd {
	var indices, others []int32
	var places [][3]float32
	var powers []float32
	for index, lamp := range lamps {
		placed, stands := lamp.(standingLamp)
		if !stands {
			others = append(others, int32(index))
			continue
		}
		power := placed.AtMost(1)
		if power == 0 {
			continue
		}
		narrow := float32(power)
		if !(narrow > 0) || math.IsInf(float64(narrow), 0) {
			others = append(others, int32(index))
			continue
		}
		at := placed.Stands()
		indices = append(indices, int32(index))
		places = append(places, [3]float32{float32(at.X), float32(at.Y),
			float32(at.Z)})
		powers = append(powers, narrow)
	}
	if len(indices) <= lampTreeLeast {
		return lampCrowd{}
	}
	// the build refuses only no lamps, or places and powers that are not
	// as many, and these are as many and more than lampTreeLeast
	tree, _ := buildLightTree(places, powers)
	for at, node := range tree.nodes {
		if node.leaf() {
			tree.nodes[at].Lamp = indices[node.Lamp]
		}
	}
	return lampCrowd{nodes: tree.nodes, others: others}
}

// errTooManyDistant is a world whose distant lamps, each asked at every
// step through a medium, leave the kernel's asks too little room for the
// lamps a step picks besides.
type errTooManyDistant struct{ distant int }

// Error says how many distant lamps there are and how many a step has
// room for.
func (found errTooManyDistant) Error() string {
	return fmt.Sprintf("gpu: a step through a medium asks every distant "+
		"lamp and this world has %d, more than the %d the kernel's %d "+
		"asks leave room for beside the %d picked outside the lamp tree "+
		"and the %d taken from it; draw it with fewer distant lamps",
		found.distant, lampsDistantMost, lampsEvery, lampsMarched,
		lampSplitMost)
}

// distantLamps is how many of the lamps outside the tree are distant,
// each a lamp of lamps, the world's: those a step asks every time.
func (crowd lampCrowd) distantLamps(lamps []thing.Lamp) int {
	var distant int
	for _, index := range crowd.others {
		if _, _, _, is := distantOf(lamps[index]); is {
			distant++
		}
	}
	return distant
}

// refuse says why the kernel cannot ask the lamps of a world with this
// crowd, or nothing when it can. A world with no crowd has no lamp
// outside one, and the kernel asks it lamp by lamp, which holds.
func (crowd lampCrowd) refuse(lamps []thing.Lamp) error {
	if distant := crowd.distantLamps(lamps); distant > lampsDistantMost {
		return errTooManyDistant{distant}
	}
	return nil
}

// floats is the crowd as the kernel reads it: the tree's nodes, then the
// outside lamps' indices, each as a float.
func (crowd lampCrowd) floats() []float32 {
	out := lightTree{nodes: crowd.nodes}.floats()
	for _, index := range crowd.others {
		out = append(out, float32(index))
	}
	return out
}

// bytes is the crowd's floats as bytes, little-endian as every buffer.
func (crowd lampCrowd) bytes() []byte {
	var packed packer
	for _, one := range crowd.floats() {
		packed.word(math.Float32bits(one))
	}
	return packed.bytes
}

// lampsRead is the lamps' buffer as the kernel reads it: the lamps' rows,
// after them the crowd where the world has one, and after that the lists
// of lamps its loops ask of (lamp_lists.go) where it has any.
func (encoded *scene) lampsRead() []byte {
	if len(encoded.crowd.nodes) == 0 && encoded.lists.empty() {
		return encoded.lamps.bytes
	}
	return slices.Concat(encoded.lamps.bytes, encoded.crowd.bytes(),
		encoded.lists.bytes())
}
