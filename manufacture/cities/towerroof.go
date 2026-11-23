package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// What stands on a tower's roof at the demo level, said once. A slab
// closes the top of the shaft and a parapet runs along every edge of it.
// On whatever is highest, unless a spire is, stand a water tank on steel
// legs with steel hoops round it, and a billboard on steel posts with a
// catwalk behind it. These are the places a writer climbs to: the outer
// face of a parapet, a tank's staves, a billboard's back.

// roofLooks are the materials of what stands on a roof: the timber of a
// tank, the steel of legs, hoops, posts and catwalks, and a billboard's
// lit face.
type roofLooks struct {
	tank, steel, board *thing.Material
}

// roofLooksOf draws the materials of a roof from the culture.
func (tower *Tower) roofLooksOf() roofLooks {
	drawing := tower.drawing
	face := *paintOf(drawing, "billboard-colour", place{})
	face.Name += " lit"
	face.Glow = face.Colour.Scale(drawing.fixed("billboard-glow"))
	return roofLooks{tank: paintOf(drawing, "tank-colour", place{}),
		steel: paintOf(drawing, "roof-steel-colour", place{}),
		board: &face}
}

// roof is the slab on a shaft's top plan and the parapet along each edge
// of it, at the demo level, unless a deck is laid across the tops.
func (tower *Tower) roof(index int, looks towerLooks) []placedElement {
	drawing := tower.drawing
	if drawing.level != demoLevel || tower.Span == SpanDeck {
		return nil
	}
	shaft := tower.Shafts[index]
	_, ends, err := shaft.layer(len(shaft.Segments) - 1)
	drawing.fail(err)
	if err != nil {
		return nil
	}
	height, slab := tower.height(index), drawing.fixed("roof-slab")
	tall, thick := drawing.fixed("parapet-height"),
		drawing.fixed("parapet-thick")
	var parts []placedElement
	for _, part := range ends {
		parts = append(parts, put("roof", prism{Bottom: polygon{part},
			Top: polygon{part}, Height: slab}, looks.podium,
			maths.V(0, height, 0)))
		for each := range part {
			parts = append(parts, parapet(part, each, height+slab, tall,
				thick, looks.podium))
		}
	}
	return parts
}

// parapet is the low wall along one edge of a part of a roof's plan,
// from a corner to the next, standing at a height, flush with the face
// below and as thick as given inward of it.
func parapet(part []outlineCorner, each int, foot, tall, thick float64,
	look *thing.Material) placedElement {
	from, to := part[each], part[(each+1)%len(part)]
	run := maths.V(to.X-from.X, 0, to.Z-from.Z)
	along := run.Unit()
	across := along.Cross(maths.YAxis)
	middle := centroid(part)
	origin := maths.V(from.X, foot, from.Z)
	if across.Dot(maths.V(middle.X-from.X, 0, middle.Z-from.Z)) < 0 {
		origin = origin.Sub(across.Scale(thick))
	}
	return placedElement{Element: solid{Name: "parapet", Form: Box{run.Len(),
		tall, thick}, Material: look}, Place: maths.NewSubspace(origin,
		maths.Orientation{X: along, Y: maths.YAxis, Z: across}, 0)}
}

// roofworks are the water tank and the billboard on the highest roof of
// a shaft, its middle at a place and half as wide as span, each if it
// fits inside the roof's margin.
func (tower *Tower) roofworks(at maths.Vec, span float64) []placedElement {
	drawing := tower.drawing
	if drawing.level != demoLevel {
		return nil
	}
	looks := tower.roofLooksOf()
	margin := drawing.fixed("roof-margin")
	radius := drawing.fixed("tank-radius")
	var parts []placedElement
	if span-margin > 2*radius {
		middle := maths.V(at.X-span+margin+radius, at.Y,
			at.Z-span+margin+radius)
		parts = append(parts, tower.tank(middle, looks)...)
	}
	wide := drawing.fixed("billboard-width")
	if 2*(span-margin)-2*radius > wide {
		front := maths.V(at.X+span-margin-wide, at.Y, at.Z-span+margin)
		parts = append(parts, tower.billboard(front, looks)...)
	}
	return parts
}

// highest is where the roofworks of a shaft stand, given the middle of
// the top of what stands on it: there, or on its roof's slab when nothing
// stands on it.
func (tower *Tower) highest(shaft shaft, at maths.Vec) maths.Vec {
	if shaft.Deck || shaft.Penthouse || shaft.Crown {
		return at
	}
	return at.Add(maths.V(0, tower.drawing.fixed("roof-slab"), 0))
}

// roofSpan is half the width of the highest roof of a shaft, given half
// the width of its crown: a penthouse's, a crown's, or a deck's with its
// overhang when the deck is all that stands there.
func (tower *Tower) roofSpan(shaft shaft, half float64, index int) float64 {
	drawing := tower.drawing
	switch {
	case shaft.Crown:
		return half
	case shaft.Penthouse:
		return half * drawing.number("penthouse-fill", place{Bay: index})
	case shaft.Deck:
		return half + drawing.fixed("observation-overhang")
	}
	return half
}

// tank is a water tank of timber standing on four steel legs, its middle
// at a place: a drum of staves with steel hoops round it and a cone of a
// lid.
func (tower *Tower) tank(middle maths.Vec, looks roofLooks) []placedElement {
	drawing := tower.drawing
	radius, legs := drawing.fixed("tank-radius"), drawing.fixed("tank-legs")
	tall, lid := drawing.fixed("tank-height"), drawing.fixed("tank-lid")
	leg, sides := drawing.fixed("tank-leg"), int(drawing.fixed("tank-sides"))
	round := circle(radius, sides)
	parts := []placedElement{
		put("water tank", prism{Bottom: polygon{round}, Top: polygon{round},
			Height: tall}, looks.tank, middle.Add(maths.V(0, legs, 0))),
		put("tank lid", prism{Bottom: polygon{round}, Top: polygon{circle(
			radius*drawing.fixed("tank-lid-top"), sides)}, Height: lid},
			looks.tank, middle.Add(maths.V(0, legs+tall, 0)))}
	reach := radius * drawing.fixed("tank-leg-spread")
	for _, corner := range [][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		parts = append(parts, put("tank leg", Box{leg, legs, leg},
			looks.steel, middle.Add(maths.V(corner[0]*reach-leg/2, 0,
				corner[1]*reach-leg/2))))
	}
	hoop := circle(radius+drawing.fixed("tank-hoop-proud"), sides)
	hoops := int(drawing.fixed("tank-hoops"))
	for each := range hoops {
		up := legs + tall*float64(each+1)/float64(hoops+1)
		parts = append(parts, put("tank hoop", prism{Bottom: polygon{hoop},
			Top: polygon{hoop}, Height: drawing.fixed("tank-hoop")},
			looks.steel, middle.Add(maths.V(0, up, 0))))
	}
	return parts
}

// billboard is a billboard on two steel posts at the north edge of a
// roof, the west foot of its face at a place: its lit face toward the
// north and the street, and its bare steel back toward the roof, with a
// catwalk along the foot of its back.
func (tower *Tower) billboard(front maths.Vec, looks roofLooks) []placedElement {
	drawing := tower.drawing
	wide, tall := drawing.fixed("billboard-width"),
		drawing.fixed("billboard-height")
	lift, thick := drawing.fixed("billboard-lift"),
		drawing.fixed("billboard-thick")
	post, deep := drawing.fixed("billboard-post"),
		drawing.fixed("catwalk-depth")
	walk, face := drawing.fixed("catwalk-thick"),
		drawing.fixed("billboard-face")
	back := front.Z + thick
	parts := []placedElement{
		put("billboard back", Box{wide, tall, thick}, looks.steel,
			maths.V(front.X, front.Y+lift, front.Z)),
		put("billboard face", Box{wide, tall, face}, looks.board,
			maths.V(front.X, front.Y+lift, front.Z-face)),
		put("catwalk", Box{wide, walk, deep}, looks.steel,
			maths.V(front.X, front.Y+lift-walk, back))}
	for _, x := range []float64{front.X + wide/4, front.X + 3*wide/4} {
		parts = append(parts, put("billboard post", Box{post, lift - walk,
			post}, looks.steel, maths.V(x-post/2, front.Y, back)))
	}
	return parts
}
