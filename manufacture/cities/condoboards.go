package cities

import (
	"github.com/janearc/slayer/maths"
)

// telescopeKind is the name of the manufactory's kind that a telescope on
// a roof or a deck is: the catalogue's domed optical observatory, which
// this package knows by its name alone.
const telescopeKind = "optical-observatory"

// telescope is a mount for a telescope on a flat area, if the building
// keeps one there and the area has room for its dome: the culture's share
// of roofs and decks have one, at the culture's scale of the catalogue's
// observatory. It stands at the left end of the area, with a gap to the
// wall, at the middle of the area's depth. It also gives how much of the
// area's length it takes from the left, which nothing else may stand in.
func (drawing *drawer) telescope(at place, area roofArea) (
	mounts []Mount, taken float64) {
	if !drawing.chance("telescope-share", at) {
		return nil, 0
	}
	scale := drawing.fixed("telescope-scale")
	radius := drawing.fixed("telescope-diameter") * scale / half
	gap := drawing.fixed("deck-gap")
	if 2*radius+gap > area.z1-area.z0 {
		return nil, 0
	}
	centre := maths.V(area.x0+gap+radius, area.y, (area.z0+area.z1)/half)
	place := maths.NewSubspace(centre, maths.Identity, scale)
	return []Mount{{Kind: telescopeKind, Place: place}}, 2*radius + 2*gap
}

// board is a sign of a colour the culture draws, that glows, a billboard
// of the many a town has: drawn at a place of the board-colour element.
func (drawing *drawer) board(name string, at place, wide, high,
	thick float64, foot maths.Vec) placedElement {
	look := drawing.choose("board-colour", scopeBay, name,
		drawing.options("board-colour"), at, nil)
	return put(name, Box{wide, high, thick}, drawing.glowing(name, look),
		foot)
}

// roofBoard is a billboard on a roof: a board on two posts at the back
// edge of the roof, at its right end, facing the front, if the building
// has one.
func (drawing *drawer) roofBoard(at place, area roofArea) []placedElement {
	if !drawing.chance("roof-board-share", at) {
		return nil
	}
	wide, high := drawing.fixed("board-width"), drawing.fixed("board-height")
	post, lift := drawing.fixed("board-post"), drawing.fixed("board-lift")
	thick := drawing.fixed("board-thickness")
	if wide > area.x1-area.x0 || thick > area.z1-area.z0 {
		return nil
	}
	left := area.x1 - wide
	steel := paint("board post", drawing.pick("roof-steel-colour",
		drawing.options("roof-steel-colour"), at))
	return []placedElement{
		put("board post", Box{post, lift, post}, steel,
			maths.V(left+post, area.y, area.z0)),
		put("board post", Box{post, lift, post}, steel,
			maths.V(left+wide-2*post, area.y, area.z0)),
		drawing.board("roof board", at, wide, high, thick,
			maths.V(left, area.y+lift, area.z0)),
	}
}

// wallBoard is a billboard on a blank side wall: on the end of the
// building, over the height the culture gives it, spanning the middle of
// the wall's depth, if the building has one. The low tier's height is as
// high as the board may reach.
func (drawing *drawer) wallBoard(at place, shape massShape, low float64,
	left bool) []placedElement {
	if !drawing.chance("wall-board-share", at) {
		return nil
	}
	rise, high := drawing.fixed("wall-board-rise"),
		drawing.fixed("wall-board-height")
	thick, inset := drawing.fixed("board-thickness"),
		drawing.fixed("wall-board-inset")
	if rise+high > low || 2*inset >= shape.depth {
		return nil
	}
	x := -thick
	if !left {
		x = shape.width
	}
	return []placedElement{drawing.board("wall board", at, thick, high,
		shape.depth-2*inset, maths.V(x, rise, inset))}
}
