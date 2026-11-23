package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// frameOf is the four bars round an opening of a width and a height: a
// sill bar below, a head bar above and a jamb each side, all outside the
// opening. They are in the frame of the pane that fills it: centred
// across, standing on its sill, through the wall's plane.
func (dress winnebagoDress) frameOf(width, height float64) []placedElement {
	trim := dress.fine.trim
	bars := dress.clothes.of(roleTrim)
	across := width + bothSides*trim.bar
	level := Box{across, trim.bar, trim.depth}
	upright := Box{trim.bar, height, trim.depth}
	back := -trim.depth / half
	return []placedElement{
		part("frame", level, bars,
			standingAt(maths.V(-across/half, -trim.bar, back))),
		part("frame", level, bars,
			standingAt(maths.V(-across/half, height, back))),
		part("frame", upright, bars,
			standingAt(maths.V(-across/half, 0, back))),
		part("frame", upright, bars,
			standingAt(maths.V(width/half, 0, back)))}
}

// framed is the frame round an opening of a shape, standing at a place.
func (dress winnebagoDress) framed(shape opening,
	place maths.Subspace) []placedElement {
	return within(dress.frameOf(shape.width, shape.height), place)
}

// glazed is a pane of glass of a shape at a place, with its frame.
func (dress winnebagoDress) glazed(shape opening,
	place maths.Subspace) []placedElement {
	return append(dress.framed(shape, place),
		placedElement{Element: paneOf(shape, roleGlazing), Place: place})
}

// screen is the windscreen, drawn again for the rounded nose: as wide as
// the flat of the cab's front, less its frame.
func (dress winnebagoDress) screen() []placedElement {
	sizes := dress.sizes
	shape := sizes.openings.screen
	shape.width = sizes.body.cabWidth - bothSides*(dress.fine.nose.radius+
		dress.fine.trim.bar)
	return dress.glazed(shape, standingAt(maths.V(0, shape.sill,
		sizes.frontFace())))
}

// windowFrames are the frames round the side windows, which are drawn as
// they were.
func (dress winnebagoDress) windowFrames() []placedElement {
	sizes := dress.sizes
	openings := sizes.openings
	var frames []placedElement
	for index := range openings.windowCount {
		along := openings.windowFirst + float64(index)*openings.windowGap
		for _, side := range []float64{kerbSide, offSide} {
			frames = append(frames, dress.framed(openings.window,
				sizes.onSide(side, openings.window.sill, along))...)
		}
	}
	return frames
}

// cabWindows are the window in each side of the cab, with its frame.
func (dress winnebagoDress) cabWindows() []placedElement {
	body, shape := dress.sizes.body, dress.fine.trim.cab
	along := (body.length + body.cabLength) / half
	var windows []placedElement
	for _, side := range []float64{kerbSide, offSide} {
		place := maths.NewSubspace(maths.V(side*body.cabWidth/half,
			shape.sill, along), maths.Rotation(maths.YAxis, side*quarterTurn),
			0)
		windows = append(windows, dress.glazed(shape, place)...)
	}
	return windows
}

// rearWindow is the window in the rear of the body, facing back, with its
// frame.
func (dress winnebagoDress) rearWindow() []placedElement {
	body, shape := dress.sizes.body, dress.fine.trim.rear
	place := maths.NewSubspace(maths.V(0, shape.sill, -body.length/half),
		maths.Rotation(maths.YAxis, math.Pi), 0)
	return dress.glazed(shape, place)
}

// doorFittings are the frame round the entry door, its handle, and the
// step before it, as they stand in the door's own place. The step is two
// treads: the lower one stands out, and the upper one, half as deep, rises
// from the ground to the door's sill, so that nothing is left open under
// the door.
func (dress winnebagoDress) doorFittings() []placedElement {
	sizes, trim := dress.sizes, dress.fine.trim
	entry := sizes.openings.entry
	handle := Box{trim.handleWidth, trim.handleHeight, trim.handleProud}
	step := Box{entry.width, trim.stepTop, trim.stepDepth}
	riser := Box{entry.width, entry.sill, trim.stepDepth / half}
	parts := append(dress.frameOf(entry.width, entry.height),
		part("handle", handle, dress.clothes.of(roleFitting),
			standingAt(maths.V(entry.width/half-bothSides*trim.handleWidth,
				trim.handleAt, entry.thickness/half))),
		part("step", step, dress.clothes.of(roleStep),
			standingAt(maths.V(-entry.width/half, -entry.sill, 0))),
		part("riser", riser, dress.clothes.of(roleStep),
			standingAt(maths.V(-entry.width/half, -entry.sill, 0))))
	return within(parts, sizes.entryElement().Place)
}
