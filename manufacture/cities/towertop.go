package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// crownLight is how bright a panel of a crown is, from nought to one, at
// a time in seconds. A band of light travels once round the crown's
// panels every period, and is as wide as the band says, in panels: a
// panel is at full brightness at the band's middle and dark a band's
// width away, either way round. The same panel at the same time is always
// the same brightness; the band is what moves.
func crownLight(panel, panels int, seconds, period, band float64) float64 {
	if panels < 1 || !(period > 0) || !(band > 0) {
		return 0
	}
	middle := math.Mod(seconds/period, 1) * float64(panels)
	away := math.Abs(float64(panel) - middle)
	away = math.Min(away, float64(panels)-away)
	return math.Max(0, 1-away/band)
}

// inradius is how far the centre of a plan is from the nearest edge of
// its part with the most room: the radius of the roundest crown it will
// hold.
func inradius(parts [][]outlineCorner) float64 {
	var best float64
	for _, part := range parts {
		centre, nearest := centroid(part), math.Inf(1)
		for index, corner := range part {
			next := part[(index+1)%len(part)]
			run := math.Hypot(next.X-corner.X, next.Z-corner.Z)
			across := (next.X-corner.X)*(corner.Z-centre.Z) -
				(next.Z-corner.Z)*(corner.X-centre.X)
			nearest = math.Min(nearest, math.Abs(across)/run)
		}
		best = math.Max(best, nearest)
	}
	return best
}

// top is what stands on a shaft: an observation deck, then a penthouse
// or a crown, a spire over those, and a beacon at the highest point if
// the shaft is tall enough. A deck laid across the tops replaces all of
// them.
func (tower *Tower) top(index int, looks towerLooks, base,
	seconds float64) []placedElement {
	if tower.Span == SpanDeck {
		return nil
	}
	shaft, drawing := tower.Shafts[index], tower.drawing
	_, ends, err := shaft.layer(len(shaft.Segments) - 1)
	drawing.fail(err)
	if err != nil {
		return nil
	}
	at := maths.V(shaft.Centre.X, tower.height(index), shaft.Centre.Z)
	room := inradius(ends)
	var parts []placedElement
	if shaft.Deck {
		parts, at = tower.observation(parts, at, room, looks, index)
	}
	if shaft.Penthouse {
		parts, at = tower.penthouse(parts, at, room, looks, index)
	}
	if shaft.Crown {
		parts, at = tower.crown(parts, at, room, looks, index, seconds)
	}
	if shaft.Spire {
		parts, at = tower.spire(parts, at, room, looks, index)
		return append(parts, tower.beacons(looks, at.Y,
			[]maths.Vec{at})...)
	}
	half := room * tower.drawing.number("crown-fill", place{Bay: index})
	parts = append(parts, tower.roofworks(tower.highest(shaft, at),
		tower.roofSpan(shaft, half, index))...)
	return append(parts, tower.beacons(looks, at.Y, []maths.Vec{
		maths.V(at.X-half, 0, at.Z-half), maths.V(at.X+half, 0, at.Z-half),
		maths.V(at.X-half, 0, at.Z+half),
		maths.V(at.X+half, 0, at.Z+half)})...)
}

// observation is a deck, a wide plate, on the shaft's top: wider than
// the crown it carries by an overhang all round.
func (tower *Tower) observation(parts []placedElement, at maths.Vec, room float64,
	looks towerLooks, index int) ([]placedElement, maths.Vec) {
	drawing := tower.drawing
	half := room*drawing.number("crown-fill", place{Bay: index}) +
		drawing.fixed("observation-overhang")
	thick := drawing.fixed("observation-thickness")
	parts = append(parts, put("observation deck", Box{2 * half, thick,
		2 * half}, looks.grid, maths.V(at.X-half, at.Y, at.Z-half)))
	return parts, maths.V(at.X, at.Y+thick, at.Z)
}

// penthouse is a slab for a terrace and, set back on it, a few storeys of
// glass, with the terrace left round it.
func (tower *Tower) penthouse(parts []placedElement, at maths.Vec, room float64,
	looks towerLooks, index int) ([]placedElement, maths.Vec) {
	drawing, place := tower.drawing, place{Bay: index}
	half := room * drawing.number("crown-fill", place)
	slab := drawing.fixed("terrace-thickness")
	inner := half * drawing.number("penthouse-fill", place)
	height := float64(drawing.count("penthouse-storeys", place)) *
		tower.storeyHeight()
	parts = append(parts,
		put("terrace", Box{2 * half, slab, 2 * half}, looks.grid,
			maths.V(at.X-half, at.Y, at.Z-half)),
		put("penthouse", Box{2 * inner, height, 2 * inner}, looks.glass,
			maths.V(at.X-inner, at.Y+slab, at.Z-inner)))
	return parts, maths.V(at.X, at.Y+slab+height, at.Z)
}

// crown is a block on the top, square, with panels set on its four
// faces. A lit crown's panels glow, each as bright as CrownLight says at
// the time; an unlit crown's do not.
func (tower *Tower) crown(parts []placedElement, at maths.Vec, room float64,
	looks towerLooks, index int, seconds float64) ([]placedElement, maths.Vec) {
	drawing, place := tower.drawing, place{Bay: index}
	half := room * drawing.number("crown-fill", place)
	height := drawing.number("crown-height", place)
	parts = append(parts, put("crown", Box{2 * half, height, 2 * half},
		looks.grid, maths.V(at.X-half, at.Y, at.Z-half)))
	parts = append(parts, tower.panels(at, half, height, looks, index,
		seconds)...)
	return parts, maths.V(at.X, at.Y+height, at.Z)
}

// panels are the panels of a crown's four faces, in order round it. The
// brightness of each is held to the culture's steps, so a crown has a few
// materials and not one for each panel.
func (tower *Tower) panels(at maths.Vec, half, height float64,
	looks towerLooks, index int, seconds float64) []placedElement {
	drawing, place := tower.drawing, place{Bay: index}
	gap := drawing.fixed("crown-gap")
	count := max(1, min(drawing.count("crown-panels", place),
		int(2*half/(gap*drawing.fixed("crown-panel-least")))))
	width := 2 * half / float64(count)
	lamps := tower.Shafts[index].LitCrown
	var parts []placedElement
	for face := range 4 {
		var ring []placedElement
		for step := range count {
			look := looks.panel
			if lamps {
				look = tower.glow(looks.panel, crownLight(face*count+step,
					4*count, seconds, drawing.fixed("crown-period"),
					drawing.fixed("crown-band")))
			}
			ring = append(ring, put("crown panel", Box{width - gap,
				height - 2*gap, drawing.fixed("crown-panel-thickness")},
				look, maths.V(-half+float64(step)*width+gap/2, gap, half)))
		}
		turn := maths.NewSubspace(at, maths.Rotation(maths.YAxis,
			float64(face)*quarterTurn), 0)
		parts = append(parts, within(ring, turn)...)
	}
	return parts
}

// glow is a panel's material lit to a share of full brightness, held to
// the culture's steps.
func (tower *Tower) glow(panel *thing.Material, share float64) *thing.Material {
	drawing := tower.drawing
	steps := drawing.fixed("crown-steps")
	held := math.Round(share*steps) / steps
	lit := *panel
	lit.Name = fmt.Sprintf("cities lit crown %.2f", held)
	colour := drawing.pick("crown-glow-colour",
		drawing.options("crown-glow-colour"), place{})
	lit.Glow = colourOf(colour).Scale(held * drawing.fixed("crown-glow"))
	return &lit
}

// spire is a thin square pyramid on the top, as tall as a share of the
// shaft's height, from a base of a share of the room to a thin tip.
func (tower *Tower) spire(parts []placedElement, at maths.Vec, room float64,
	looks towerLooks, index int) ([]placedElement, maths.Vec) {
	drawing, place := tower.drawing, place{Bay: index}
	base := room * drawing.number("spire-base-share", place)
	tip := drawing.fixed("spire-tip")
	height := tower.height(index) * drawing.number("spire-share", place)
	form := prism{Bottom: rectangle(-base, -base, base, base),
		Top: rectangle(-tip, -tip, tip, tip), Height: height}
	parts = append(parts, put("spire", form, looks.grid,
		maths.V(at.X, at.Y, at.Z)))
	return parts, maths.V(at.X, at.Y+height, at.Z)
}
