package cities

import (
	"errors"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// A part of a design that can be drawn coarse or fine is a node. A pass
// on the CPU, before anything is drawn, asks each node for its children
// only when it is in view and its level calls for them; otherwise the
// node draws itself whole. What the pass leaves is plain parts, each with
// its level, so nothing deferred reaches a renderer.

// node is a part of a design drawn as finely as a pass asks. Bounds is
// the box it fills in its own space; a node whose box is empty takes the
// level of what holds it. Draw is the node whole at a level. Opens says
// whether a level calls for its children, and Unfurl makes them.
type node struct {
	Bounds maths.Box
	Draw   func(level Detail) ([]placedElement, error)
	// Opens is asked with the node's level and the level the pass gives
	// a box in the node's own space. Nil never opens.
	Opens  func(level Detail, levelOf func(maths.Box) Detail) bool
	Unfurl func(level Detail) ([]placedElement, error)
}

// deferredNode is a node standing in a design as one of its elements. It
// keeps every frame it has been taken out through, in order, so that a
// pass stands its parts exactly where the frames would have stood them
// had they been drawn when it was made.
type deferredNode struct {
	Node   *node
	frames []maths.Subspace
}

// errDeferred is the refusal of a deferred node asked to draw alone.
var errDeferred = errors.New("cities: a deferred node is drawn by a " +
	"pass, not as one element")

// Rendered refuses: a node is many parts, and a pass draws it.
func (deferredNode) Rendered(Detail) (Rendered, error) {
	return Rendered{}, errDeferred
}

// deferred is a node placed at the origin of what holds it.
func deferred(node *node) placedElement {
	return placedElement{Element: deferredNode{Node: node},
		Place: standingAt(maths.Vec{})}
}

// liftedInto is a part as it stands in a parent frame: its place taken
// out through the parent's, and a node's frames kept.
func liftedInto(part placedElement, parent maths.Subspace) placedElement {
	moved := placedElement{Element: part.Element, Place: part.Place.Within(parent)}
	if node, ok := part.Element.(deferredNode); ok {
		node.frames = append(slices.Clone(node.frames), parent)
		moved.Element = node
	}
	return moved
}

// bounded is whether a box has any size: an empty box is no bounds.
func bounded(box maths.Box) bool { return box.Min != box.Max }

// boxOut is the box that holds a box of a space, seen from its parent.
func boxOut(box maths.Box, space maths.Subspace) maths.Box {
	return corners(box, space.Out)
}

// corners is the box that holds the eight corners of a box, each moved.
func corners(box maths.Box, move func(maths.Vec) maths.Vec) maths.Box {
	first := move(box.Min)
	held := maths.Box{Min: first, Max: first}
	for _, x := range []float64{box.Min.X, box.Max.X} {
		for _, y := range []float64{box.Min.Y, box.Max.Y} {
			for _, z := range []float64{box.Min.Z, box.Max.Z} {
				point := move(maths.V(x, y, z))
				held.Min = maths.V(math.Min(held.Min.X, point.X),
					math.Min(held.Min.Y, point.Y),
					math.Min(held.Min.Z, point.Z))
				held.Max = maths.V(math.Max(held.Max.X, point.X),
					math.Max(held.Max.Y, point.Y),
					math.Max(held.Max.Z, point.Z))
			}
		}
	}
	return held
}

// levelled is one plain part as a pass leaves it, and its level.
type levelled struct {
	placedElement
	Level Detail
}

// Pass is how a design's nodes are levelled: the leveller, what the eye
// sees, nil for everything, and the finest level, which the design is
// built at and its parts with no bounds of their own above any node are
// drawn at. Eye is the camera itself, for a part that judges its own
// size in pixels; nil when the pass has no camera to give.
type Pass struct {
	Leveller Leveller
	Sees     func(maths.Box) bool
	Finest   Detail
	Eye      *strategies.Eye
}

// errPass is the refusal of a pass with no leveller.
var errPass = errors.New("cities: a pass needs a leveller")

// levelled is the parts with every node in them drawn or unfurled, each
// part at its level. Outer is where the parts' frame stands in the world
// the leveller sees, and level the level of what holds them.
func (pass Pass) levelled(parts []placedElement, outer maths.Subspace,
	level Detail) ([]levelled, error) {
	var flat []levelled
	for _, part := range parts {
		node, ok := part.Element.(deferredNode)
		if !ok {
			flat = append(flat, levelled{placedElement: part, Level: level})
			continue
		}
		inner, err := pass.node(*node.Node, part.Place.Within(outer), level)
		if err != nil {
			return nil, err
		}
		for _, parent := range node.frames {
			raise(inner, parent)
		}
		flat = append(flat, inner...)
	}
	return flat, nil
}

// raise takes every part out through a parent frame, in place.
func raise(flat []levelled, parent maths.Subspace) {
	for index := range flat {
		flat[index].Place = flat[index].Place.Within(parent)
	}
}

// node is one node drawn whole, or unfurled where it is seen and its
// level calls for its children, in the node's own frame.
func (pass Pass) node(node node, world maths.Subspace,
	level Detail) ([]levelled, error) {
	seen := true
	if bounded(node.Bounds) {
		box := boxOut(node.Bounds, world)
		seen = pass.Sees == nil || pass.Sees(box)
		level = pass.level(box)
	}
	levelOf := func(box maths.Box) Detail {
		return pass.level(boxOut(box, world))
	}
	if seen && node.Opens != nil && node.Opens(level, levelOf) {
		children, err := node.Unfurl(level)
		if err != nil {
			return nil, err
		}
		return pass.levelled(children, world, level)
	}
	drawn, err := node.Draw(level)
	if err != nil {
		return nil, err
	}
	return pass.levelled(drawn, world, level)
}

// level is the level the leveller gives a box in the world.
func (pass Pass) level(box maths.Box) Detail {
	middle := box.Min.Add(box.Max).Scale(0.5)
	return pass.Leveller(standingAt(middle), box.Max.Sub(box.Min).Len())
}
