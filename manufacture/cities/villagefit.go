package cities

import (
	"fmt"
	"math"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// What a village house has fitted to its walls and stood on its roof. Each
// part is one of a few sizes a house or the culture fixes, so that a cage
// placed forty times on a house is one bar and one plate placed many
// times, and an air conditioner one box for the whole village.

// quick is a part drawn at the quick level and finer.
func quick(part placedElement) tiered { return tiered{placedElement: part, least: quickLevel} }

// fine is a part drawn at the demo level only.
func fine(part placedElement) tiered { return tiered{placedElement: part, least: demoLevel} }

// tiers is each part given, drawn at a level and finer.
func tiers(parts []placedElement, least Detail) []tiered {
	out := make([]tiered, len(parts))
	for index, part := range parts {
		out[index] = tiered{placedElement: part, least: least}
	}
	return out
}

// add keeps parts of the house, each drawn at its level and finer.
func (job *houseJob) add(parts []tiered) {
	job.house.tiers = append(job.house.tiers, parts...)
}

// fit is what is fitted to each opening of a wall, in the wall's frame,
// kept in the house's.
func (job *houseJob) fit(side int, frame maths.Subspace, face facade) {
	for storey, openings := range face.Storeys {
		floor := face.floorAt(storey)
		for index, opening := range openings {
			right := face.Width
			if index+1 < len(openings) {
				right = openings[index+1].Left
			}
			at := place{Row: job.at.Row, Bay: side, Storey: storey,
				Opening: index}
			for _, one := range job.fitted(at, opening, floor, right) {
				one.placedElement = liftedInto(one.placedElement, frame)
				job.add([]tiered{one})
			}
		}
	}
}

// fitted is what one opening has, in its wall's frame: a lamp over the
// stair door, a light over an open shop, and at a window what the draws
// give it of a cage, an air conditioner and clothes poles. Right is where
// the next opening along begins, or the end of the wall.
func (job *houseJob) fitted(at place, opening facadeOpening, floor,
	right float64) []tiered {
	drawing := job.drawing
	switch {
	case opening.Door:
		return []tiered{job.doorLamp(opening, floor)}
	case opening.Extra.Under == shopfront:
		return job.shopLamp(at, opening, floor)
	}
	var parts []tiered
	caged := drawing.chance("cage", at)
	if caged {
		parts = append(parts, job.cage(opening, floor)...)
	}
	if drawing.chance("aircon", at) {
		parts = append(parts, job.aircon(at, opening, floor, caged,
			right)...)
	}
	if at.Storey > 0 && drawing.chance("poles", at) {
		parts = append(parts, job.poles(at, opening, floor)...)
	}
	return parts
}

// cage is the steel cage over a window, standing out of the wall: a plate
// under it and one over it, and bars up its front, at the quick level;
// and at the demo level bars up its sides and a rail across its middle.
func (job *houseJob) cage(opening facadeOpening, floor float64) []tiered {
	drawing, steel := job.drawing, job.steel
	side, rise := drawing.fixed("cage-side"), drawing.fixed("cage-rise")
	deep, bar := drawing.fixed("cage-depth"), drawing.fixed("cage-bar")
	left, foot := opening.Left-side, floor+opening.Sill-rise
	wide, high := opening.Width+2*side, opening.Height+2*rise
	plate := Box{wide, bar, deep}
	upright := Box{bar, high - 2*bar, bar}
	parts := []tiered{
		quick(put("cage", plate, steel, maths.V(left, foot, 0))),
		quick(put("cage", plate, steel, maths.V(left, foot+high-bar, 0)))}
	for _, along := range evenly(wide-bar, drawing.fixed("cage-pitch")) {
		parts = append(parts, quick(put("cage bar", upright, steel,
			maths.V(left+along, foot+bar, deep-bar))))
	}
	for _, back := range evenly(deep-bar, drawing.fixed("cage-pitch")) {
		parts = append(parts,
			fine(put("cage bar", upright, steel, maths.V(left, foot+bar,
				back))),
			fine(put("cage bar", upright, steel, maths.V(left+wide-bar,
				foot+bar, back))))
	}
	return append(parts, fine(put("cage rail", Box{wide, bar, bar}, steel,
		maths.V(left, foot+high/half, deep-bar))))
}

// evenly are the places from nought to a length, both ends included, at
// most a pitch apart and all the same distance apart.
func evenly(length, pitch float64) []float64 {
	steps := max(int(length/pitch+1-meeting), 1)
	places := make([]float64, steps+1)
	for index := range places {
		places[index] = length * float64(index) / float64(steps)
	}
	return places
}

// aircon is the outdoor box of an air conditioner hung on the wall by a
// window: under it, under its cage if it has one, or beside it if there
// is room before the next opening and the draw says so. At the demo level
// it has its fan's grille, two brackets and its drip pipe.
func (job *houseJob) aircon(at place, opening facadeOpening, floor float64,
	caged bool, right float64) []tiered {
	drawing := job.drawing
	size := Box{drawing.fixed("unit-width"), drawing.fixed("unit-height"),
		drawing.fixed("unit-depth")}
	side := drawing.fixed("cage-side")
	corner := maths.V(opening.Left+(opening.Width-size.Width)/half,
		floor+opening.Sill-drawing.fixed("aircon-drop")-size.Height, 0)
	if caged {
		corner.Y -= drawing.fixed("cage-rise")
	}
	beside := opening.Left + opening.Width + side +
		drawing.fixed("aircon-beside")
	if drawing.pick("aircon-place", drawing.options("aircon-place"),
		at).Name == "beside" && beside+size.Width+side <= right {
		corner = maths.V(beside, floor+opening.Sill, 0)
	}
	if corner.Y < drawing.fixed("aircon-lowest") {
		return nil
	}
	unit := paint("air conditioner", drawing.pick("unit-colour",
		drawing.options("unit-colour"), at))
	parts := []tiered{quick(put("air conditioner", size, unit, corner))}
	return append(parts, job.airconDetail(at, size, corner)...)
}

// airconDetail is what an air conditioner has at the demo level: the
// grille of its fan on its front, two brackets under it, and its drip
// pipe hanging from under one end, of a length held to steps.
func (job *houseJob) airconDetail(at place, size Box,
	corner maths.Vec) []tiered {
	drawing := job.drawing
	radius, bracket := drawing.fixed("grille-radius"),
		drawing.fixed("bracket-size")
	grille := paint("grille", drawing.pick("grille-colour",
		drawing.options("grille-colour"), at))
	face := maths.NewSubspace(corner.Add(maths.V(size.Width*grilleAlong,
		size.Height/half, size.Depth)), maths.Rotation(maths.XAxis,
		quarterTurn), 0)
	parts := []tiered{fine(placedElement{Place: face, Element: solid{
		Name: "grille", Material: grille, Form: prism{Bottom: outlineCircle{radius},
			Top: outlineCircle{radius}, Height: bracket}}})}
	for _, along := range []float64{bracket, size.Width - 2*bracket} {
		parts = append(parts, fine(put("bracket", Box{bracket, bracket,
			size.Depth}, job.steel, corner.Add(maths.V(along, -bracket,
			0)))))
	}
	step, pipe := drawing.fixed("drip-step"), drawing.fixed("drip-size")
	drop := max(roundTo(drawing.number("drip-length", at), step), step)
	return append(parts, fine(put("drip pipe", Box{pipe, drop, pipe},
		job.steel, corner.Add(maths.V(size.Width-2*bracket-pipe,
			-drop, size.Depth/half)))))
}

// grilleAlong is how far along an air conditioner's front its fan's
// grille stands, as a share of its width: off to one side, as a fan is.
const grilleAlong = 0.35

// roundTo is a number held to the nearest of its steps.
func roundTo(value, step float64) float64 {
	return math.Round(value/step) * step
}

// poles are the clothes poles out over the alley from over a window, one
// near each side of it, and the washing hung along them.
func (job *houseJob) poles(at place, opening facadeOpening,
	floor float64) []tiered {
	drawing := job.drawing
	reach, size := drawing.fixed("pole-reach"), drawing.fixed("pole-size")
	inset := drawing.fixed("pole-inset")
	high := floor + opening.Sill + opening.Height + drawing.fixed("pole-rise")
	var parts []tiered
	for index, along := range []float64{opening.Left + inset,
		opening.Left + opening.Width - inset - size} {
		parts = append(parts, quick(put("clothes pole", Box{size, size,
			reach}, job.steel, maths.V(along, high, 0))))
		hung := at
		hung.Dwelling = index
		parts = append(parts, job.washing(hung, along+size/half, high,
			reach)...)
	}
	return parts
}

// washing is the clothes hung on hangers along a pole, each across it,
// spread along its reach, each a cloth the draw gives.
func (job *houseJob) washing(at place, middle, high,
	reach float64) []tiered {
	drawing := job.drawing
	wide, drop := drawing.fixed("cloth-width"), drawing.fixed("cloth-drop")
	thin := drawing.fixed("cloth-thick")
	count := drawing.count("washing", at)
	cloths := drawing.options("cloth-colour")
	var parts []tiered
	for index := range count {
		along := reach * float64(index+1) / float64(count+1)
		cloth := drawing.choose("cloth-colour", scopeDwelling,
			fmt.Sprint(index), cloths, at, nil)
		parts = append(parts, quick(put("washing", Box{wide, drop, thin},
			paint("cloth", cloth), maths.V(middle-wide/half, high-drop,
				along))))
	}
	return parts
}

// glowLamp is a part that glows at a heat and is a light of a luminance:
// a lamp at a door or over a shop.
func glowLamp(name string, form Box, heat, glow, luminance float64,
	at maths.Vec) placedElement {
	look := &thing.Material{Name: fmt.Sprintf("cities %s %.0fK", name, heat),
		Colour: maths.RGB(1, 1, 1),
		Glow:   swatches.Light(swatch.Planckian(heat)).Scale(glow)}
	return placedElement{Place: standingAt(at), Element: solid{Name: name,
		Form: form, Material: look,
		Lamp: &Lamp{Heat: heat, Luminance: luminance}}}
}

// doorLamp is the lamp over the stair door, which lights the alley.
func (job *houseJob) doorLamp(opening facadeOpening, floor float64) tiered {
	drawing := job.drawing
	size := drawing.fixed("door-lamp-size")
	return quick(glowLamp("door lamp", Box{size, size / half, size},
		drawing.fixed("door-lamp-heat"), drawing.fixed("door-lamp-glow"),
		drawing.fixed("door-lamp-strength"), maths.V(opening.Left+
			(opening.Width-size)/half, floor+opening.Sill+opening.Height+
			drawing.fixed("door-lamp-rise"), 0)))
}

// shopLamp is the strip of light over an open shop, at the heat of its
// lit glass; a shuttered shop has none.
func (job *houseJob) shopLamp(at place, opening facadeOpening,
	floor float64) []tiered {
	lit := opening.Pane != nil && opening.Pane.Glow != (maths.Light{})
	if !lit && opening.Lamp == nil {
		return nil
	}
	drawing := job.drawing
	wide, size := drawing.fixed("shop-lamp-width"),
		drawing.fixed("door-lamp-size")
	return []tiered{quick(glowLamp("shop lamp", Box{wide, size, size},
		drawing.heat(at), drawing.fixed("door-lamp-glow"),
		drawing.fixed("shop-lamp-strength"), maths.V(opening.Left+
			(opening.Width-wide)/half, floor+opening.Sill+opening.Height+
			drawing.fixed("door-lamp-rise"), 0)))}
}
