package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// tier is a run of storeys of a building that stand one over another on
// one footprint: how many, and how deep the tier is from its back to its
// front. A building that steps back is a stack of tiers, each shallower
// than the one under it.
type tier struct {
	storeys int
	depth   float64
}

// stack is what a building's tiers come to, in its frame: every part
// drawn, the roof the highest tier has, how far the highest tier's front
// stands back from the lowest tier's, and the telescopes it keeps.
type stack struct {
	parts  []placedElement
	top    roofArea
	reach  float64
	mounts []Mount
	// low is the height of the lowest tier's wall, as high as a board on
	// the building's side wall may reach.
	low float64
}

// roofTelescope is the telescope on a building's highest roof, if it
// keeps one, which takes the left end of the roof from what else stands
// there. At the quick level there is none: the building is not a home.
func (drawing *drawer) roofTelescope(at place, built *stack) []Mount {
	if !drawing.homes {
		return nil
	}
	mounts, taken := drawing.telescope(place{Bay: at.Bay}, built.top)
	built.top.x0 += taken
	return mounts
}

// tiers are the tiers of a building of a shape. At the demo level a
// building steps back by the culture's share of them: every few storeys
// its front stands back by the culture's step, and the roof it leaves
// bare is a deck. It stops stepping back when a tier would be shallower
// than the culture's least, and the storeys left go on the last tier.
func (drawing *drawer) tiers(at place, shape massShape) []tier {
	whole := []tier{{shape.storeys, shape.depth}}
	if !drawing.homes || !drawing.chance("setback-share", at) {
		return whole
	}
	every := drawing.count("setback-every", at)
	step := drawing.number("setback-depth", at)
	least := drawing.fixed("setback-least-depth")
	var tiers []tier
	for left := shape.storeys; left > 0; {
		depth := shape.depth - float64(len(tiers))*step
		if depth < least {
			tiers[len(tiers)-1].storeys += left
			break
		}
		take := every
		if left-take < 1 {
			take = left
		}
		tiers = append(tiers, tier{take, depth})
		left -= take
	}
	return tiers
}

// stacked is a building drawn tier by tier, each standing on the roof of
// the one below, with a deck on every roof that is left bare. The upper
// tiers have no ground storey, plinth or downpipe, and no balcony on the
// storey that stands on a deck.
func (shared court) stacked(drawing *drawer, at place, shape massShape,
	wall *thing.Material) stack {
	var built stack
	tiers := drawing.tiers(at, shape)
	floor, done := 0.0, 0
	for index, each := range tiers {
		house := drawing.tierHouse(at, shape, wall, each, done, index)
		last := index == len(tiers)-1
		if !last {
			house.Roof.Parapet = 0
		}
		parts, err := house.parts()
		drawing.fail(err)
		built.parts = append(built.parts, within(parts, standingAt(
			maths.V(0, floor, 0)))...)
		top := floor + house.Facade.Height + house.Roof.Slab
		if index == 0 {
			built.low = house.Facade.Height
		}
		if !last {
			deck := drawing.deckArea(shape.width, each, tiers[index+1], top)
			mounts, taken := drawing.telescope(place{Bay: at.Bay,
				Row: index + 1}, deck)
			built.mounts = append(built.mounts, mounts...)
			built.parts = append(built.parts, drawing.deck(at, index+1, deck,
				taken)...)
		}
		built.top = drawing.roofOf(shape.width, each, top)
		floor, done = top, done+each.storeys
	}
	built.mounts = append(built.mounts, drawing.roofTelescope(at, &built)...)
	built.reach = shape.depth - tiers[len(tiers)-1].depth
	return built
}

// tierHouse is the house of one tier: its storeys drawn as the storeys
// that follow those below, in the shape of the whole.
func (drawing *drawer) tierHouse(at place, shape massShape,
	wall *thing.Material, each tier, done, index int) house {
	sized := shape
	sized.storeys, sized.depth = each.storeys, each.depth
	drawing.storeyBase = done
	if index > 0 {
		if drawing.onDecks == nil {
			drawing.onDecks = map[place]bool{}
		}
		drawing.onDecks[place{Bay: at.Bay, Storey: done}] = true
	}
	house := drawing.house(at, sized, wall)
	drawing.storeyBase = 0
	if index > 0 {
		front := &house.Facade
		front.Height -= front.Ground
		front.Ground, front.Upper = 0, true
		front.Fixtures = fixtures{Stone: front.Fixtures.Stone}
	}
	return house
}

// roofOf is the flat top of a tier whose floor is at a height: inside
// its parapet.
func (drawing *drawer) roofOf(width float64, each tier, y float64) roofArea {
	wall := drawing.fixed("parapet-wall")
	return roofArea{x0: wall, z0: wall, x1: width - wall,
		z1: each.depth - wall, y: y}
}

// deckArea is the bare roof of a tier that the tier above leaves: the
// strip along its front, from a margin off the face of the tier above to
// the tier's own edge, at the height of its roof.
func (drawing *drawer) deckArea(width float64, under, over tier,
	y float64) roofArea {
	wall := drawing.fixed("parapet-wall")
	return roofArea{x0: wall, x1: width - wall, y: y,
		z0: over.depth + drawing.fixed("deck-margin"), z1: under.depth}
}
