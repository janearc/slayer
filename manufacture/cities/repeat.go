package cities

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
)

// unit is one thing a run repeats: given which one it is, counting from
// nought, the parts it is made of, standing in the unit's own frame with
// its own origin at the start of its space.
type unit func(index int) ([]placedElement, error)

// repeat is a unit placed Count times along an axis, one Spacing after
// the last. The same primitive makes a row of houses along a street, with
// the axis along the street, and the storeys up a building, with the
// axis up: a run is a run.
//
// Handed mirrors every other unit about the middle of its own space, so
// that a row of doors or chimneys comes in pairs, as a semi-detached
// pair is built. The ends of a run may differ: First, when given, is the
// unit at index nought, and Last the unit at the end, in place of Unit.
// When the run is one unit long that unit is First.
type repeat struct {
	Count   int
	Spacing float64
	Axis    maths.Vec
	Handed  bool
	Unit    unit
	First   unit
	Last    unit
}

// errUnit is the refusal of a run that has no unit to repeat.
var errUnit = errors.New("cities: a run needs a unit to repeat")

// parts is every unit of the run, each moved along the axis to its own
// place, the handed ones mirrored. It refuses a run of no units, one
// with no spacing, and one with no unit.
func (repeat repeat) parts() ([]placedElement, error) {
	if repeat.Count < 1 || !(repeat.Spacing > 0) {
		return nil, fmt.Errorf("cities: a run needs at least one unit and "+
			"a positive spacing, not %d at %v", repeat.Count, repeat.Spacing)
	}
	if repeat.Unit == nil {
		return nil, errUnit
	}
	var parts []placedElement
	for index := range repeat.Count {
		unit, err := repeat.unitAt(index)(index)
		if err != nil {
			return nil, fmt.Errorf("cities: unit %d of a run: %w", index, err)
		}
		if repeat.Handed && index%2 == 1 {
			unit = repeat.mirrored(unit)
		}
		offset := repeat.Axis.Scale(float64(index) * repeat.Spacing)
		parts = append(parts, within(unit, standingAt(offset))...)
	}
	return parts, nil
}

// unitAt is the unit that stands at an index: the first or the last
// where the run has one for that end, else the repeated unit.
func (repeat repeat) unitAt(index int) unit {
	if index == 0 && repeat.First != nil {
		return repeat.First
	}
	if index == repeat.Count-1 && index > 0 && repeat.Last != nil {
		return repeat.Last
	}
	return repeat.Unit
}

// mirrored is the parts of a unit reflected about the middle of its
// space, across the axis. A node is mirrored as a node, each part it
// draws or unfurls reflected as it comes.
func (repeat repeat) mirrored(parts []placedElement) []placedElement {
	reflected := make([]placedElement, len(parts))
	for index, part := range parts {
		mirror := reflection{inner: part, axis: repeat.Axis,
			spacing: repeat.Spacing}
		if node, ok := part.Element.(deferredNode); ok {
			reflected[index] = deferred(mirror.node(node, repeat))
			continue
		}
		reflected[index] = placedElement{Place: standingAt(maths.Vec{}),
			Element: mirror}
	}
	return reflected
}

// node is a deferred node mirrored: its box, and each part it draws or
// unfurls, taken out through its frames and reflected.
func (mirror reflection) node(inner deferredNode, repeat repeat) *node {
	out := func(parts []placedElement, err error) ([]placedElement, error) {
		for _, parent := range inner.frames {
			parts = within(parts, parent)
		}
		return repeat.mirrored(parts), err
	}
	box := func(each maths.Box) maths.Box { return mirror.box(inner, each) }
	node := &node{Bounds: box(inner.Node.Bounds),
		Draw: func(level Detail) ([]placedElement, error) {
			return out(inner.Node.Draw(level))
		},
		Unfurl: func(level Detail) ([]placedElement, error) {
			return out(inner.Node.Unfurl(level))
		}}
	if opens := inner.Node.Opens; opens != nil {
		node.Opens = func(level Detail, levelOf func(maths.Box) Detail) bool {
			return opens(level, func(each maths.Box) Detail {
				return levelOf(box(each))
			})
		}
	}
	return node
}

// box is a box of a mirrored node's space, taken out through its frames
// and reflected. An empty box stays empty.
func (mirror reflection) box(inner deferredNode, box maths.Box) maths.Box {
	if !bounded(box) {
		return box
	}
	for _, parent := range inner.frames {
		box = boxOut(box, parent)
	}
	return corners(box, mirror.reflect)
}

// reflection is a placed element drawn as its mirror image. The place
// is baked into its mesh, so it stands at the origin of its holder.
type reflection struct {
	inner   placedElement
	axis    maths.Vec
	spacing float64
}

// Rendered is the inner element drawn, taken to its place, reflected
// about the plane across the axis at half the spacing, and wound again
// so that its faces still look outward.
func (mirror reflection) Rendered(level Detail) (Rendered, error) {
	drawn, err := mirror.inner.Element.Rendered(level)
	if err != nil {
		return Rendered{}, err
	}
	flipped := &Mesh{Name: drawn.Mesh.Name}
	for _, triangle := range drawn.Mesh.Triangles {
		var corners Triangle
		for index, corner := range triangle {
			placed := mirror.inner.Place.Out(drawn.Place.Out(corner))
			corners[index] = mirror.reflect(placed)
		}
		corners[1], corners[2] = corners[2], corners[1]
		flipped.Triangles = append(flipped.Triangles, corners)
	}
	return Rendered{Mesh: flipped, Place: standingAt(maths.Vec{}),
		Material: drawn.Material, Lamp: drawn.Lamp}, nil
}

// reflect is a point's mirror image in the plane across the axis, half a
// spacing along it.
func (mirror reflection) reflect(point maths.Vec) maths.Vec {
	over := point.Dot(mirror.axis) - mirror.spacing/2
	return point.Sub(mirror.axis.Scale(2 * over))
}
