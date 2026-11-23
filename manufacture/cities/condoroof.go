package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The things a flat roof carries.
const (
	roofPlantRoom = "plant-room"
	roofTank      = "tank"
	roofAntenna   = "antenna"
	roofGantry    = "gantry"
)

// roofArea is the flat top a thing may stand on, in a building's frame:
// its corners on the ground plane and the height of its floor. A roof
// and a deck are each one, and nothing placed on one reaches past it.
type roofArea struct{ x0, z0, x1, z1, y float64 }

// roofKit is the materials of what a roof carries, drawn once for a
// building.
type roofKit struct{ steel, housing, tank *thing.Material }

// roofKitOf draws the materials of a building's roof furniture.
func (drawing *drawer) roofKitOf(at place) roofKit {
	return roofKit{
		steel: paint("roof steel", drawing.pick("roof-steel-colour",
			drawing.options("roof-steel-colour"), at)),
		housing: paint("roof housing", drawing.pick("roof-housing-colour",
			drawing.options("roof-housing-colour"), at)),
		tank: paint("roof tank", drawing.pick("roof-tank-colour",
			drawing.options("roof-tank-colour"), at))}
}

// roofWidth is how much of a roof's length a thing of a kind takes.
func (drawing *drawer) roofWidth(kind string) float64 {
	return drawing.fixed(map[string]string{roofPlantRoom: "roof-plant-width",
		roofTank: "tank-width", roofAntenna: "antenna-span",
		roofGantry: "gantry-span"}[kind])
}

// crown is what a building carries on its roof: the plant room, tanks,
// antennas and a gantry the culture draws for it, spread along the roof
// with the same gap between each and at the ends, on its middle line.
func (drawing *drawer) crown(at place, area roofArea) []placedElement {
	kit := drawing.roofKitOf(at)
	var kinds []string
	for index := range drawing.count("roof-items", at) {
		kinds = append(kinds, drawing.choose("roof-item", scopeBay,
			string(rune('a'+index)), drawing.options("roof-item"), at,
			nil).Name)
	}
	widths := make([]float64, len(kinds))
	for index, kind := range kinds {
		widths[index] = drawing.roofWidth(kind)
	}
	lefts, _ := spread(area.x0, area.x1, widths)
	middle := (area.z0 + area.z1) / half
	var parts []placedElement
	for index, kind := range kinds {
		base := maths.V(lefts[index], area.y, middle)
		parts = append(parts, drawing.roofThing(kind, kit, base)...)
	}
	return parts
}

// roofThing is one thing a roof carries, standing with its left foot at
// a point of the roof and its depth centred on it.
func (drawing *drawer) roofThing(kind string, kit roofKit,
	base maths.Vec) []placedElement {
	switch kind {
	case roofTank:
		return drawing.roofTanks(kit, base)
	case roofAntenna:
		return drawing.roofAntennas(kit, base)
	case roofGantry:
		return drawing.roofGantries(kit, base)
	}
	return drawing.roofPlantRooms(kit, base)
}

// roofPlantRooms is the lift motor room and its vent stack: a box with a
// narrower stack on its top.
func (drawing *drawer) roofPlantRooms(kit roofKit, base maths.Vec) []placedElement {
	wide, high := drawing.fixed("roof-plant-width"),
		drawing.fixed("roof-plant-height")
	deep, stack := drawing.fixed("roof-plant-depth"),
		drawing.fixed("vent-stack")
	return []placedElement{
		put("plant room", Box{wide, high, deep}, kit.housing,
			base.Sub(maths.V(0, 0, deep/half))),
		put("vent stack", Box{stack, stack, stack}, kit.steel,
			base.Add(maths.V(wide/half-stack/half, high, -stack/half))),
	}
}

// roofTanks is a water tank on four legs: a drum on a stand.
func (drawing *drawer) roofTanks(kit roofKit, base maths.Vec) []placedElement {
	radius, stand := drawing.fixed("tank-width")/half,
		drawing.fixed("tank-stand")
	leg := drawing.fixed("tank-leg")
	parts := []placedElement{{Place: standingAt(base.Add(maths.V(radius, stand, 0))),
		Element: solid{Name: "water tank", Material: kit.tank,
			Form: prism{Bottom: outlineCircle{radius}, Top: outlineCircle{radius},
				Height: drawing.fixed("tank-height")}}}}
	for _, across := range []float64{0, 2*radius - leg} {
		for _, back := range []float64{-radius, radius - leg} {
			parts = append(parts, put("tank leg", Box{leg, stand, leg},
				kit.steel, base.Add(maths.V(across, 0, back))))
		}
	}
	return parts
}

// roofAntennas is a mast with two cross bars near its top and a dish
// standing off its side.
func (drawing *drawer) roofAntennas(kit roofKit, base maths.Vec) []placedElement {
	span, mast := drawing.fixed("antenna-span"), drawing.fixed("mast-width")
	high, bar := drawing.fixed("mast-height"), drawing.fixed("antenna-bar")
	middle := base.Add(maths.V(span/half-mast/half, 0, -mast/half))
	return []placedElement{
		put("mast", Box{mast, high, mast}, kit.steel, middle),
		put("antenna bar", Box{span, bar, bar}, kit.steel,
			middle.Add(maths.V(-span/half+mast/half, high*antennaUpper, 0))),
		put("antenna bar", Box{span / half, bar, bar}, kit.steel,
			middle.Add(maths.V(-span/half+mast/half+span/half/half,
				high*antennaLower, 0))),
		put("dish", Box{span / half, span / half, bar}, kit.housing,
			middle.Add(maths.V(mast, high*antennaLower-span/half/half,
				-bar/half))),
	}
}

// antennaUpper and antennaLower are how far up a mast its two cross bars
// are, as shares of its height. Choices, by eye.
const (
	antennaUpper = 0.9
	antennaLower = 0.7
)

// roofGantries is the gantry a building cleans its face from: two posts
// and a beam between them, with a cradle hung from it.
func (drawing *drawer) roofGantries(kit roofKit, base maths.Vec) []placedElement {
	span, post := drawing.fixed("gantry-span"), drawing.fixed("gantry-post")
	high := drawing.fixed("gantry-height")
	return []placedElement{
		put("gantry post", Box{post, high, post}, kit.steel,
			base.Sub(maths.V(0, 0, post/half))),
		put("gantry post", Box{post, high, post}, kit.steel,
			base.Add(maths.V(span-post, 0, -post/half))),
		put("gantry beam", Box{span, post, post}, kit.steel,
			base.Add(maths.V(0, high, -post/half))),
		put("cradle", Box{span / half, drawing.fixed("cradle-height"),
			post}, kit.housing, base.Add(maths.V(span/half/half,
			high-drawing.fixed("cradle-height"), -post/half))),
	}
}
