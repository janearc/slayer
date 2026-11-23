package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The axes an alley's gap may run across, as the index of the
// coordinate that is x or z.
const (
	acrossX = 0
	acrossZ = 1
)

// hang is the second half of growing a superblock: everything that is
// hung or stood on the buildings once they are built, in order.
func (growing *accretion) hang() {
	for index, alley := range growing.out.Alleys {
		growing.bridges(index, alley)
	}
	for index := range growing.out.Plots {
		for side := range sidesAround {
			growing.hangings(index, side)
		}
		growing.roofs(index)
	}
	for index, alley := range growing.out.Alleys {
		growing.lane(index, alley)
	}
}

// edgeOf is a rectangle's low and high edge along an axis.
func (box rect) edgeOf(axis int) (low, high float64) {
	if axis == acrossX {
		return box.X0, box.X1
	}
	return box.Z0, box.Z1
}

// gapAxis is the axis an alley is narrow across, which is the way its
// two walls face each other.
func (box rect) gapAxis() int {
	if box.width() < box.depth() {
		return acrossX
	}
	return acrossZ
}

// bridges joins pairs of buildings that face each other across an alley
// with an enclosed walkway, by the draw for each pair.
func (growing *accretion) bridges(index int, alley rect) {
	axis := alley.gapAxis()
	low, high := alley.edgeOf(axis)
	for first, one := range growing.out.Plots {
		_, top := one.Lot.edgeOf(axis)
		for second, two := range growing.out.Plots {
			bottom, _ := two.Lot.edgeOf(axis)
			if same(top, low) && same(bottom, high) {
				growing.bridge(place{Row: index, Bay: first*len(
					growing.out.Plots) + second}, alley, first, second)
			}
		}
	}
}

// bridge is a walkway between two buildings that stand against the two
// sides of an alley, if they face each other for long enough, the draw
// says so, and there is a storey both reach and room for it.
func (growing *accretion) bridge(at place, alley rect, first,
	second int) {
	drawing := growing.drawing
	plots := growing.out.Plots
	axis := alley.gapAxis()
	along := 1 - axis
	start, end := alley.edgeOf(along)
	for _, plot := range []plot{plots[first], plots[second]} {
		low, high := plot.Lot.edgeOf(along)
		start, end = math.Max(start, low), math.Min(end, high)
	}
	width := drawing.fixed("bridge-width")
	least := drawing.fixed("bridge-least-storey")
	top := min(plots[first].Storeys, plots[second].Storeys) - 1
	if end-start < drawing.fixed("bridge-overlap") || float64(top) < least ||
		!drawing.gate("bridge-chance", at, growing.grown, nil) {
		return
	}
	storey := int(least + math.Floor(drawing.number("bridge-high", at)*
		(float64(top)-least+1)))
	floor := math.Max(plots[first].floorOf(storey),
		plots[second].floorOf(storey))
	room := drawing.number("bridge-along", at) * (end - start - width)
	growing.span(at, alley, axis, floor, start+room)
}

// span lays the walkway: its box across the alley, from the floor, at a
// place along it, in the sheet the culture draws for additions.
func (growing *accretion) span(at place, alley rect, axis int, floor,
	place float64) {
	drawing := growing.drawing
	height, width := drawing.fixed("bridge-height"),
		drawing.fixed("bridge-width")
	size := Box{alley.width(), height, width}
	corner := maths.V(alley.X0, floor, place)
	if axis == acrossZ {
		size = Box{width, height, alley.depth()}
		corner = maths.V(place, floor, alley.Z0)
	}
	volume := volumeOf("bridge", -1, standingAt(maths.Vec{}), size, corner)
	if !growing.settle(volume) {
		return
	}
	sheet := paint("box", drawing.pick("box-colour",
		drawing.options("box-colour"), at))
	growing.parts = append(growing.parts, put("bridge", size, sheet, corner))
}

// hangings is what is hung on one wall of a building: for each storey
// above the ground, a cage or a box by one gate, and a sign by another.
func (growing *accretion) hangings(index, side int) {
	drawing, grown := growing.drawing, growing.grown
	plot := growing.out.Plots[index]
	frame, length := wallFrame(plot.Lot, side, drawing.fixed("recess"))
	for storey := 1; storey < plot.Storeys; storey++ {
		at := place{Row: side, Bay: index, Storey: storey}
		floor := plot.floorOf(storey)
		if drawing.gate("addition-share", at, grown, nil) {
			growing.addition(at, frame, length, floor)
		}
		if drawing.gate("sign-share", at, grown, nil) {
			growing.sign(at, frame, length, floor)
		}
	}
}

// mount is the volume of a box hung on a wall at a share of the room
// along it, from a floor, and whether the building is tall enough to
// hold it. It stands out of the wall by its depth.
func (growing *accretion) mount(kind string, at place,
	frame maths.Subspace, length, floor float64, size Box,
	place string) (Volume, maths.Vec, bool) {
	left := growing.drawing.number(place, at) * (length - size.Width)
	corner := maths.V(left, floor, 0)
	volume := volumeOf(kind, at.Bay, frame, size, corner)
	roof := growing.out.Plots[at.Bay].Height
	return volume, corner, floor+size.Height <= roof+meeting
}

// addition is one cage or box hung on a wall, if it fits there.
func (growing *accretion) addition(at place, frame maths.Subspace, length,
	floor float64) {
	drawing := growing.drawing
	kind := drawing.pick("addition-kind", drawing.options("addition-kind"),
		at).Name
	size := Box{drawing.number(kind+"-width", at),
		drawing.number(kind+"-height", at), drawing.number(kind+"-proud", at)}
	volume, corner, fits := growing.mount(kind, at, frame, length, floor,
		size, "addition-place")
	if !fits || !growing.settle(volume) {
		return
	}
	sheet := paint(kind, drawing.pick(kind+"-colour",
		drawing.options(kind+"-colour"), at))
	parts := []placedElement{put(kind, size, sheet, corner)}
	if kind == "cage" {
		parts = growing.caged(size, sheet, corner)
	}
	growing.parts = append(growing.parts, within(parts, frame)...)
}

// caged is a cage drawn as it is: a floor and a roof of bars' width and
// four posts, not a solid box, as wide and deep and tall as the cage.
func (growing *accretion) caged(size Box, steel *thing.Material,
	corner maths.Vec) []placedElement {
	bar := growing.drawing.fixed("mast-width")
	parts := []placedElement{
		put("cage", Box{size.Width, bar, size.Depth}, steel, corner),
		put("cage", Box{size.Width, bar, size.Depth}, steel,
			corner.Add(maths.V(0, size.Height-bar, 0)))}
	for _, post := range []maths.Vec{maths.V(0, 0, 0),
		maths.V(size.Width-bar, 0, 0), maths.V(0, 0, size.Depth-bar),
		maths.V(size.Width-bar, 0, size.Depth-bar)} {
		parts = append(parts, put("cage", Box{bar, size.Height, bar}, steel,
			corner.Add(post)))
	}
	return parts
}

// sign is a panel that glows, hung tall on a wall, if it fits there.
func (growing *accretion) sign(at place, frame maths.Subspace, length,
	floor float64) {
	drawing := growing.drawing
	size := Box{drawing.number("sign-width", at),
		drawing.number("sign-height", at), drawing.fixed("sign-proud")}
	volume, corner, fits := growing.mount("sign", at, frame, length, floor,
		size, "sign-place")
	if !fits || !growing.settle(volume) {
		return
	}
	lamp := drawing.pick("sign-colour", drawing.options("sign-colour"), at)
	glowing := &thing.Material{Name: "cities sign " + lamp.Name,
		Colour: colourOf(lamp), Gloss: lamp.Gloss,
		Glow: colourOf(lamp).Scale(drawing.fixed("sign-glow"))}
	growing.parts = append(growing.parts, within([]placedElement{put("sign", size,
		glowing, corner)}, frame)...)
}
