package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// retailBleed is how far a unit's shell stands past its own edges into its
// neighbours, in metres, so that the boxes of two units, in one material,
// never share an edge: the mesh refuses an edge that two faces run the
// same way.
const retailBleed = 0.01

// retailRecess is how far a floor stops short of the wall plane, in
// metres, so that its edge never lies in the same plane as a pier's face.
const retailRecess = 0.02

// retailLeaf is the thickness of a sticker or a tag, in metres.
const retailLeaf = 0.004

// retailStickerDraws is how many numbers a sticker draws: where along,
// how high, and which look.
const retailStickerDraws = 3

// retailMirrorShare is the share of units the seed turns about: a unit
// is as likely to stand one way as the other.
const retailMirrorShare = 0.5

// retailBoxRight is the place of a box's right edge among its numbers.
const retailBoxRight = 3

// The places in a seed a frontage draws from, one for each thing it draws.
const (
	retailPickDraw = iota + 1
	retailMirrorDraw
	retailPropDraw
	retailStickerDraw
	retailWidthDraw
	retailOptionDraw
	retailPaletteDraw
	retailBlockDraw
)

// retailLamp is a light a frontage carries, in its frame.
type retailLamp struct {
	at              maths.Vec
	heat, luminance float64
}

// FrontageSlot is a named place on a frontage's wall plane that another
// kind may fill: a lettered sign over a door. The frontage draws nothing
// in it. Place stands at the slot's lower corner nearest the frontage's
// left end, in the frontage's frame, facing out into the street.
type FrontageSlot struct {
	Unit          string
	Name          string
	Place         maths.Subspace
	Width, Height float64
}

// retailBuilder gathers one frontage: its surfaces by material, its
// lamps, and its slots, a unit at a time along the front. The width and
// the extra, the unit's width over its least, and the picks, the material
// it has from each palette, are those of the unit being drawn.
//
// A podium
// deeper than a unit's room is made up behind it with plain wall. The
// solids are how many closed pieces have been drawn, to say what a
// frontage costs.
type retailBuilder struct {
	book   retailBook
	level  int
	skins  map[string]surface
	lamps  []retailLamp
	slots  []FrontageSlot
	left   float64
	width  float64
	extra  float64
	podium float64
	mirror bool
	picks  map[string]string
	solids int
}

// corners are the low and high corners of each place a part is drawn.
func (part retailPart) corners() []maths.Vec {
	var found []maths.Vec
	for index := range max(part.Count, 1) {
		shift := retailPoint(part.Step).Scale(float64(index))
		low, high := part.extent()
		found = append(found, low.Add(shift), high.Add(shift))
	}
	return found
}

// extent is the low and high corners of a part's first place.
func (part retailPart) extent() (maths.Vec, maths.Vec) {
	if part.Box != nil {
		return maths.V(part.Box[0], part.Box[1], part.Box[2]),
			maths.V(part.Box[3], part.Box[4], part.Box[5])
	}
	if part.Slab != nil {
		slab := part.Slab
		return maths.V(slab[0], math.Min(slab[3], slab[5]),
				math.Min(slab[2], slab[4])),
			maths.V(slab[1], math.Max(slab[3], slab[5])+slab[6],
				math.Max(slab[2], slab[4]))
	}
	centre := maths.V(part.Prism[0], part.Prism[1], part.Prism[2])
	radius, length := part.Prism[3], part.Prism[4]
	reach := maths.V(radius, radius, radius)
	reach = reach.Add(retailAlong(part.Axis).Scale(length*spaceHalf - radius))
	return centre.Sub(reach), centre.Add(reach)
}

// retailPoint is a list of three numbers as a point, or the origin for
// no list.
func retailPoint(list []float64) maths.Vec {
	if len(list) != retailPointNumbers {
		return maths.Vec{}
	}
	return maths.V(list[0], list[1], list[2])
}

// retailAlong is the direction of a prism's axis, with every part of it
// the same in all directions but along: a unit vector.
func retailAlong(axis string) maths.Vec {
	switch axis {
	case "x":
		return maths.XAxis
	case "z":
		return maths.ZAxis
	}
	return maths.YAxis
}

// reaches is whether a level of detail draws what is drawn from a level.
func (builder *retailBuilder) reaches(from string) bool {
	return retailLevel(from) <= builder.level
}

// across is an x range in a unit's frame as it stands in the frontage:
// turned about if the unit is, and moved along to where the unit starts.
func (builder *retailBuilder) across(low, high float64) (float64, float64) {
	if builder.mirror {
		low, high = builder.width-high, builder.width-low
	}
	return builder.left + low, builder.left + high
}

// add puts a surface in a material, or in the material the unit picked
// from the palette that the name is of.
func (builder *retailBuilder) add(material string, skin surface) {
	material = builder.resolve(material)
	builder.skins[material] = append(builder.skins[material], skin...)
	builder.solids++
}

// solid puts a closed box between two corners, in a unit's frame, in a
// material.
func (builder *retailBuilder) solid(material string, low, high maths.Vec) {
	left, right := builder.across(low.X, high.X)
	builder.add(material, cityBox(left, right, low.Y, high.Y, low.Z, high.Z))
}

// round puts a closed prism about a point in a unit's frame, in a material.
func (builder *retailBuilder) round(material string, part retailPart,
	origin maths.Vec) {
	centre := maths.V(part.Prism[0], part.Prism[1], part.Prism[2]).Add(origin)
	across, _ := builder.across(centre.X, centre.X)
	centre.X = across
	sides := part.Sides
	if sides == 0 {
		sides = int(cityValue("plant-sides"))
	}
	builder.add(material, retailCylinder(centre, part.Axis, part.Prism[3],
		part.Prism[4], sides))
}

// plank puts a closed slab in a unit's frame, moved by a shift, in a
// material.
func (builder *retailBuilder) plank(material string, slab []float64,
	shift maths.Vec) {
	left, right := builder.across(slab[0]+shift.X, slab[1]+shift.X)
	builder.add(material, retailSlab(left, right, slab[2]+shift.Z,
		slab[3]+shift.Y, slab[4]+shift.Z, slab[5]+shift.Y, slab[6]))
}

// retailCylinder is a closed prism about a centre, its axis x, y or z, of
// as many sides as it is given.
func retailCylinder(centre maths.Vec, axis string, radius,
	length float64, sides int) surface {
	along := retailAlong(axis)
	first, second := retailAcross(axis)
	half := along.Scale(length * spaceHalf)
	ring := func(index int, end float64) maths.Vec {
		turn := fullTurn * float64(index) / float64(sides)
		return centre.Add(first.Scale(radius * math.Cos(turn))).Add(
			second.Scale(radius * math.Sin(turn))).Add(half.Scale(end))
	}
	var skin surface
	for index := range sides {
		low, high := ring(index, -1), ring(index, 1)
		nextLow, nextHigh := ring((index+1)%sides, -1), ring((index+1)%sides, 1)
		out := low.Add(nextLow).Sub(centre.Add(half.Neg()).Scale(cityWhole))
		skin = append(skin, cityQuad(low, nextLow, nextHigh, high, out)...)
		skin = append(skin, cityTriangle(centre.Add(half), high, nextHigh, along),
			cityTriangle(centre.Sub(half), nextLow, low, along.Neg()))
	}
	return skin
}

// retailAcross are the two directions across a prism's axis.
func retailAcross(axis string) (maths.Vec, maths.Vec) {
	switch axis {
	case "x":
		return maths.YAxis, maths.ZAxis
	case "z":
		return maths.XAxis, maths.YAxis
	}
	return maths.XAxis, maths.ZAxis
}

// retailSkin is a surface of one material that a good holds as a part.
type retailSkin struct{ triangles surface }

// skin is the surface, whatever the cut.
func (shape retailSkin) skin(cut) surface { return shape.triangles }

// part draws each place of a part at a level, about an origin that a
// prop or a module gives and a unit does not, as the part stands in a
// unit that has grown by grow metres.
func (builder *retailBuilder) part(piece retailPart, origin maths.Vec,
	grow float64) {
	if !builder.reaches(piece.From) {
		return
	}
	piece = piece.slid(grow)
	for index := range max(piece.Count, 1) {
		shift := origin.Add(retailPoint(piece.Step).Scale(float64(index)))
		switch {
		case piece.Box != nil:
			low := maths.V(piece.Box[0], piece.Box[1], piece.Box[2]).Add(shift)
			high := maths.V(piece.Box[3], piece.Box[4], piece.Box[5]).Add(shift)
			builder.solid(piece.Material, low, high)
		case piece.Slab != nil:
			builder.plank(piece.Material, piece.Slab, shift)
		default:
			builder.round(piece.Material, piece, shift)
		}
	}
}
