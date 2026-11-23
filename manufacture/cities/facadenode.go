package cities

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// A facade levelled is a node: whole, it is its slab at the hull and its
// parts at any other level, as Parts draws them. Unfurled, its wall is
// drawn and each storey's openings are a node of their own, and each
// opening of a storey is a node again, so a near window is drawn to its
// lights and its sill while a far one on the same face is a pane.

// at is the facade drawn at a level, whatever level it was made at.
func (facade facade) at(level Detail) facade {
	facade.Level = level
	return facade
}

// node is the facade as a node, in the facade's own frame.
func (facade facade) node() *node {
	return &node{Bounds: facade.box(0, facade.Width, 0, facade.Height),
		Draw:   facade.whole,
		Opens:  aboveHull,
		Unfurl: facade.unfurled}
}

// aboveHull opens a node at every level but the hull.
func aboveHull(level Detail, _ func(maths.Box) Detail) bool {
	return level != hullLevel
}

// unfurled is the facade's parts as Parts draws them at a level, with
// each storey's openings a node, standing where Parts stands them.
func (facade facade) unfurled(level Detail) ([]placedElement, error) {
	if err := facade.check(); err != nil {
		return nil, err
	}
	drawn := facade.at(level)
	var wall, filled []placedElement
	for index, openings := range facade.Storeys {
		storey := facade.heightOf(index)
		stand := standingAt(maths.V(0, facade.floorAt(index), 0))
		wall = append(wall, within(drawn.skin(openings, storey), stand)...)
		filled = append(filled, within([]placedElement{deferred(
			facade.storeyNode(openings, storey))}, stand)...)
	}
	if level == demoLevel {
		filled = append(filled, drawn.cornice()...)
		filled = append(filled, drawn.fixtures()...)
		filled = append(filled, drawn.plinth()...)
	}
	return append(wall, filled...), nil
}

// storeyNode is the openings of one storey as a node: whole, each drawn
// at the storey's level, or a pane each at the hull; unfurled, a node
// for each opening.
func (facade facade) storeyNode(openings []facadeOpening, storey float64) *node {
	return &node{Bounds: facade.box(0, facade.Width, 0, storey),
		Draw: func(level Detail) ([]placedElement, error) {
			if level == hullLevel {
				return facade.panes(openings), nil
			}
			return facade.at(level).fill(openings, storey), nil
		},
		Opens: aboveHull,
		Unfurl: func(Detail) ([]placedElement, error) {
			nodes := make([]placedElement, len(openings))
			for index, opening := range openings {
				nodes[index] = deferred(facade.openingNode(opening, storey))
			}
			return nodes, nil
		}}
}

// openingNode is one opening as a node: a pane at the hull, and at any
// other level the opening as a storey fills it.
func (facade facade) openingNode(opening facadeOpening, storey float64) *node {
	reach := opening.ShutterWidth
	return &node{Bounds: facade.box(opening.Left-reach,
		opening.Left+opening.Width+reach, opening.Sill,
		opening.Sill+opening.Height),
		Draw: func(level Detail) ([]placedElement, error) {
			if level == hullLevel {
				return facade.panes([]facadeOpening{opening}), nil
			}
			return facade.at(level).fill([]facadeOpening{opening}, storey), nil
		}}
}

// panes are openings at the hull. On a face made at the hull each is a
// pane as big as its opening at the back of the cut. On a face made finer
// and drawn coarse, as a city's distant lit level is, a lit window keeps
// its light, a pane that glows with its lamp in it, and unlit glass is
// left out, so the cut shows the dark wall behind it.
func (facade facade) panes(openings []facadeOpening) []placedElement {
	var parts []placedElement
	for _, opening := range openings {
		at := maths.V(opening.Left, opening.Sill, -facade.Recess)
		if !facade.coarse(hullLevel) {
			parts = append(parts, put("pane", Box{opening.Width,
				opening.Height, facade.GlassThickness}, opening.Pane, at))
			continue
		}
		parts = append(parts, facade.litPane(opening, at)...)
	}
	return parts
}

// litPane is a lit window drawn coarse, at a place: a pane as big as the
// opening, glowing as the window did, and the light of the lamp behind
// it, if it has one. An unlit window has none.
func (facade facade) litPane(opening facadeOpening, at maths.Vec) []placedElement {
	look := opening.Lamp
	if look == nil {
		look = opening.Pane
	}
	if look == nil || look.Glow == (maths.Light{}) {
		return nil
	}
	pane := solid{Name: "lit pane", Material: look,
		Form: Box{opening.Width, opening.Height, facade.GlassThickness}}
	if opening.Lamp != nil {
		pane.Lamp = &Lamp{Heat: opening.Heat,
			Luminance: facade.Finish.LampStrength}
	}
	return []placedElement{{Element: pane, Place: standingAt(at)}}
}

// whole is the facade drawn whole at a level: its parts as Parts draws
// them, and, drawn at the hull when made finer, its slab with each lit
// window's light on the slab's face.
func (facade facade) whole(level Detail) ([]placedElement, error) {
	parts, err := facade.at(level).parts()
	if err != nil || level != hullLevel || !facade.coarse(hullLevel) {
		return parts, err
	}
	for index, openings := range facade.Storeys {
		floor := facade.floorAt(index)
		for _, opening := range openings {
			parts = append(parts, facade.litPane(opening,
				maths.V(opening.Left, floor+opening.Sill, 0))...)
		}
	}
	return parts, nil
}

// facadeLevels are the levels a face is made and drawn at, coarsest
// first.
var facadeLevels = []Detail{hullLevel, quickLevel, demoLevel}

// quickLevel is the level a face is drawn at when none is given.
const quickLevel Detail = "quick"

// coarse is whether a level is coarser than the one the facade was made
// at, which is the quick level when it was given none.
func (facade facade) coarse(level Detail) bool {
	rank := func(of Detail) int {
		if index := slices.Index(facadeLevels, of); index >= 0 {
			return index
		}
		return slices.Index(facadeLevels, quickLevel)
	}
	return rank(level) < rank(facade.Level)
}

// box is the box of a stretch of the face, from left to right and from
// foot to head, from the back of its wall to the front of what stands
// proudest of it.
func (facade facade) box(left, right, foot, head float64) maths.Box {
	front := facade.ShutterThickness
	for _, openings := range facade.Storeys {
		for _, opening := range openings {
			front = math.Max(front, opening.Proud)
		}
	}
	return maths.Box{Min: maths.V(left, foot, -facade.shell()),
		Max: maths.V(right, head, front+facade.Recess)}
}

// deferred is the house as Parts draws it, with its face a node that a
// pass draws as finely as it is seen. It refuses what Parts refuses.
func (house house) deferred() ([]placedElement, error) {
	facade := house.Facade
	if facade.Level == demoLevel {
		facade.Cavity = facade.cavityFor(house.Depth)
	}
	if !(house.Depth > facade.Recess) {
		return nil, fmt.Errorf("cities: a house must be deeper than the "+
			"cuts in its front, not %v against %v", house.Depth,
			facade.Recess)
	}
	if err := facade.check(); err != nil {
		return nil, err
	}
	roof, err := house.Roof.parts()
	if err != nil {
		return nil, err
	}
	body := put("wall", Box{facade.Width, facade.Height,
		house.Depth - facade.shell()}, facade.Wall, maths.Vec{})
	front := within([]placedElement{deferred(facade.node())},
		standingAt(maths.V(0, 0, house.Depth)))
	parts := append([]placedElement{body}, front...)
	return append(parts, within(roof, standingAt(
		maths.V(0, facade.Height, 0)))...), nil
}
