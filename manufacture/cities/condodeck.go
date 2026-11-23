package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// deckKit is the materials of a deck and what stands on it, drawn once
// for the deck.
type deckKit struct{ boards, rail, planter, leaf, furniture *thing.Material }

// deckKitOf draws the materials of a deck.
func (drawing *drawer) deckKitOf(at place) deckKit {
	return deckKit{
		boards: paint("decking", drawing.pick("deck-colour",
			drawing.options("deck-colour"), at)),
		rail: paint("deck rail", drawing.pick("iron-colour",
			drawing.options("iron-colour"), at)),
		planter: paint("planter", drawing.pick("planter-colour",
			drawing.options("planter-colour"), at)),
		leaf: paint("foliage", foliage),
		furniture: paint("deck furniture", drawing.pick("chair-colour",
			drawing.options("chair-colour"), at))}
}

// deck is the roof a tier above leaves bare, made a place to be: boards
// over its slab, a railing round its three open sides, a row of planters
// inside the front rail, a table and chairs, and sometimes a pergola over
// them.
//
// The level is how many tiers are under it, so each deck is drawn
// anew. In the building's frame, everything inside the area. The boards
// and the railing run its whole length; what stands on it keeps clear of
// the first part of it, as far as the taken gives, where a telescope is.
func (drawing *drawer) deck(at place, level int, area roofArea,
	taken float64) []placedElement {
	at.Row = level
	kit := drawing.deckKitOf(at)
	thick := drawing.fixed("deck-thickness")
	parts := []placedElement{put("decking", Box{area.x1 - area.x0, thick,
		area.z1 - area.z0}, kit.boards, maths.V(area.x0, area.y, area.z0))}
	area.y += thick
	parts = append(parts, drawing.railing(kit, area)...)
	area.x0 += taken
	parts = append(parts, drawing.planters(at, kit, area)...)
	parts = append(parts, drawing.seating(at, kit, area)...)
	if drawing.chance("pergola-share", at) {
		parts = append(parts, drawing.pergola(kit, area)...)
	}
	return parts
}

// railing is the rail round a deck's front and two sides: a top bar on
// each, and a post a pitch apart along the front.
func (drawing *drawer) railing(kit deckKit, area roofArea) []placedElement {
	high, bar := drawing.fixed("deck-rail-height"), drawing.fixed("rail-bar")
	post, pitch := drawing.fixed("deck-rail-post"),
		drawing.fixed("deck-rail-pitch")
	top := area.y + high - bar
	parts := []placedElement{
		put("deck rail", Box{area.x1 - area.x0, bar, bar}, kit.rail,
			maths.V(area.x0, top, area.z1-bar)),
		put("deck rail", Box{bar, bar, area.z1 - area.z0}, kit.rail,
			maths.V(area.x0, top, area.z0)),
		put("deck rail", Box{bar, bar, area.z1 - area.z0}, kit.rail,
			maths.V(area.x1-bar, top, area.z0)),
	}
	for left := area.x0; left+post <= area.x1; left += pitch {
		parts = append(parts, put("deck post", Box{post, high - bar, post},
			kit.rail, maths.V(left, area.y, area.z1-post)))
	}
	return parts
}

// planters are a row of troughs with leaves in them along the inside of a
// deck's front rail, as many as the culture draws, spread along it.
func (drawing *drawer) planters(at place, kit deckKit,
	area roofArea) []placedElement {
	count := drawing.count("deck-planters", at)
	wide, high := drawing.fixed("deck-planter-width"),
		drawing.fixed("deck-planter-height")
	deep := drawing.fixed("deck-planter-depth")
	lefts, _ := spread(area.x0, area.x1, repeated(wide, count))
	front := area.z1 - drawing.fixed("deck-rail-post") - deep
	var parts []placedElement
	for _, left := range lefts {
		parts = append(parts,
			put("deck planter", Box{wide, high, deep}, kit.planter,
				maths.V(left, area.y, front)),
			put("deck foliage", Box{wide, drawing.fixed(
				"deck-foliage-height"), deep}, kit.leaf,
				maths.V(left, area.y+high, front)))
	}
	return parts
}

// bandOf is how deep the strip along a deck's front is that the planters
// and the rail take, which the seating and the pergola keep out of.
func (drawing *drawer) bandOf() float64 {
	return drawing.fixed("deck-planter-depth") + drawing.fixed(
		"deck-rail-post") + drawing.fixed("deck-gap")
}

// seating is a table at the middle of the deck's free depth, and the
// chairs the culture draws round it, half behind it and half before it,
// or all behind it where the deck is too shallow for both.
func (drawing *drawer) seating(at place, kit deckKit, area roofArea) []placedElement {
	wide, high := drawing.fixed("deck-table-width"),
		drawing.fixed("deck-table-height")
	deep, gap := drawing.fixed("deck-table-depth"), drawing.fixed("deck-gap")
	size, seat := drawing.fixed("chair-width"), drawing.fixed("chair-seat")
	free := area.z1 - area.z0 - drawing.bandOf()
	both := free >= deep+2*(gap+size)
	middle := maths.V((area.x0+area.x1)/half, area.y,
		(area.z0+area.z1-drawing.bandOf())/half)
	parts := []placedElement{put("deck table", Box{wide, high, deep}, kit.furniture,
		middle.Sub(maths.V(wide/half, 0, deep/half)))}
	for index := range drawing.count("deck-chairs", at) {
		along := float64(index/half)*(size+gap) - wide/half
		behind := middle.Z - deep/half - gap - size
		if both && index%half == 1 {
			behind = middle.Z + deep/half + gap
		}
		foot := maths.V(middle.X+along, area.y, behind)
		parts = append(parts,
			put("deck chair", Box{size, seat, size}, kit.furniture, foot),
			put("deck chair back", Box{size, drawing.fixed("chair-back"),
				drawing.fixed("chair-leg")}, kit.furniture,
				foot.Add(maths.V(0, seat, 0))))
	}
	return parts
}

// pergola is four posts with a beam along each side and slats across,
// over the table at the middle of the deck.
func (drawing *drawer) pergola(kit deckKit, area roofArea) []placedElement {
	wide, high := drawing.fixed("pergola-width"), drawing.fixed(
		"pergola-height")
	post, pitch := drawing.fixed("pergola-post"), drawing.fixed(
		"pergola-pitch")
	deep := area.z1 - area.z0 - drawing.bandOf()
	left := (area.x0+area.x1)/half - wide/half
	var parts []placedElement
	for _, across := range []float64{0, wide - post} {
		for _, back := range []float64{0, deep - post} {
			parts = append(parts, put("pergola post", Box{post, high, post},
				kit.rail, maths.V(left+across, area.y, area.z0+back)))
		}
	}
	for _, back := range []float64{0, deep - post} {
		parts = append(parts, put("pergola beam", Box{wide, post, post},
			kit.rail, maths.V(left, area.y+high, area.z0+back)))
	}
	for along := 0.0; along+post <= wide; along += pitch {
		parts = append(parts, put("pergola slat", Box{post / half, post / half,
			deep}, kit.rail, maths.V(left+along, area.y+high+post, area.z0)))
	}
	return parts
}
