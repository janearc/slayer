package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// What hangs from a cable: lanterns, banners and tarpaulins.

// hangMiddle is the middle of a share.
const hangMiddle = 0.5

// cordInset is how far in from the end of a panel its cords hang, as a
// share of half its width: near the end, not at it.
const cordInset = 0.8

// paperShare is how much of a lantern's colour its unlit paper reflects,
// as a share: the glow does the lighting, the paper is only dim.
const paperShare = 0.3

// boardShare is how much light a sign's board reflects, as a grey: a dark
// board, with its colour carried by its glow.
const boardShare = 0.05

// flat is the direction a cable runs along the ground, as a unit vector.
func (hung hung) flat() maths.Vec {
	return maths.V(hung.to.X-hung.from.X, 0, hung.to.Z-hung.from.Z).Unit()
}

// lanterns hangs a string of lanterns along a cable, evenly, closer
// together the denser the street.
func (job *stringing) lanterns(at place, hung hung) {
	drawing := job.drawing
	spacing := numberBetween(drawing.fixed("spacing-most"),
		drawing.fixed("spacing-least"), job.density)
	if !(spacing > 0) {
		return
	}
	count := int(hung.run / spacing)
	for index := range count {
		share := (float64(index) + hangMiddle) / float64(count)
		job.lantern(place{Row: at.Row, Bay: at.Bay, Storey: index},
			hung.at(share), index+at.Bay)
	}
}

// lantern hangs one lantern by a cord from a point of a cable: a glowing
// ball of paper in a colour of the culture's, one of three sizes, and
// every so many of them also a light. The number is what the culture's
// count of lanterns between lights is taken of, so that the lights of
// two cables do not stand in a row.
func (job *stringing) lantern(at place, top maths.Vec, number int) {
	drawing := job.drawing
	size := drawing.count("lantern-size", at)
	radius := drawing.fixed("lantern-radius") +
		float64(size)*drawing.fixed("lantern-step")
	stretch, drop := drawing.fixed("lantern-stretch"),
		drawing.fixed("lantern-drop")
	colour := drawing.pick("lantern-colour",
		drawing.options("lantern-colour"), at)
	var lamp *Lamp
	if number%max(int(drawing.fixed("lamp-every")), 1) == 0 {
		lamp = &Lamp{Heat: drawing.fixed("lantern-heat"),
			Luminance: drawing.fixed("lantern-luminance")}
	}
	paper := &thing.Material{Name: "cities lantern " + colour.Name,
		Colour: colourOf(colour).Scale(paperShare),
		Glow:   colourOf(colour).Scale(drawing.fixed("lantern-glow"))}
	middle := top.Sub(maths.V(0, drop+radius*stretch, 0))
	job.out.parts = append(job.out.parts, placedElement{Element: solid{
		Name: "lantern", Form: lashingLantern{radius, stretch}, Material: paper,
		Lamp: lamp}, Place: standingAt(middle)}, job.cord(top, drop))
	job.out.Lanterns++
}

// cord is a cord hanging from a point down a number of metres.
func (job *stringing) cord(from maths.Vec, length float64) placedElement {
	thick := job.drawing.fixed("cord-radius") * 2
	return placedElement{Element: solid{Name: "cord", Material: job.steel(),
		Form: Box{thick, length, thick}},
		Place: standingAt(from.Sub(maths.V(thick/2, length, thick/2)))}
}

// banner hangs a banner across the middle of a cable: a plain panel that
// glows in a colour of the culture's, and is a light, and a slot for a
// sign to be stood in. The panel is drawn only if no kind is asked to
// fill the slot.
func (job *stringing) banner(at place, hung hung) {
	drawing := job.drawing
	colour := drawing.pick("banner-colour", drawing.options("banner-colour"),
		at)
	board := &thing.Material{Name: "cities banner " + colour.Name,
		Colour: maths.Grey(boardShare),
		Glow:   colourOf(colour).Scale(drawing.fixed("banner-glow"))}
	lamp := &Lamp{Heat: drawing.fixed("banner-heat"),
		Luminance: drawing.fixed("banner-luminance")}
	panel, slot, cords := job.panel(at, hung, "banner",
		drawing.number("banner-place", at), board, lamp)
	job.out.Slots = append(job.out.Slots, slot)
	job.out.panels = append(job.out.panels, panel)
	job.out.parts = append(job.out.parts, cords...)
}

// tarpaulin hangs a tarpaulin over the middle of a cable: a dull panel
// that glows not at all, in a colour of the culture's. It is no slot.
func (job *stringing) tarpaulin(at place, hung hung) {
	drawing := job.drawing
	cloth := paint("tarpaulin", drawing.pick("tarp-colour",
		drawing.options("tarp-colour"), at))
	panel, _, cords := job.panel(at, hung, "tarp", hangMiddle, cloth, nil)
	job.out.parts = append(job.out.parts, panel)
	job.out.parts = append(job.out.parts, cords...)
}

// panel is a flat panel hung from a cable by two cords, across the cable
// and so facing along the street, a share of the way along it. Its size
// is the culture's, under the name given, and its top hangs a drop
// under the lowest of the cable where its cords are tied. It is given
// with the slot it fills and its cords.
func (job *stringing) panel(at place, hung hung, name string, share float64,
	material *thing.Material, lamp *Lamp) (placedElement, Slot, []placedElement) {
	drawing := job.drawing
	width, height := drawing.number(name+"-width", at),
		drawing.number(name+"-height", at)
	reach := width * cordInset / 2 / hung.run
	ties := []maths.Vec{hung.at(share - reach), hung.at(share),
		hung.at(share + reach)}
	top := math.Min(ties[0].Y, math.Min(ties[1].Y, ties[2].Y)) -
		drawing.fixed("banner-drop")
	depth, along := drawing.fixed(name+"-depth"), hung.flat()
	side := along.Cross(maths.YAxis)
	middle := maths.V(ties[1].X, top, ties[1].Z)
	origin := middle.Sub(along.Scale(width / 2)).Sub(
		maths.V(0, height, 0)).Sub(side.Scale(depth / 2))
	panel := placedElement{Element: solid{Name: name, Form: Box{width, height,
		depth}, Material: material, Lamp: lamp}, Place: maths.NewSubspace(
		origin, maths.Orientation{X: along, Y: maths.YAxis, Z: side}, 0)}
	slot := Slot{Middle: middle.Sub(maths.V(0, height/2, 0)), Along: along,
		Width: width, Height: height}
	return panel, slot, []placedElement{
		job.cord(ties[0], ties[0].Y-top), job.cord(ties[2], ties[2].Y-top)}
}
