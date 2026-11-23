package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// facadeOpening is a window or a door in one face of a mass, as one storey of
// the face holds it. It is recessed: the wall is cut away round it and
// its glass or panel stands back in the cut, so that it casts a little
// shadow and is not stuck on.
type facadeOpening struct {
	// Left is how far along the face its left edge is, in metres.
	Left float64
	// Sill is how high its foot stands over the floor of its storey.
	Sill float64
	// Width and Height are the size of the opening, frame and all.
	Width, Height float64
	// Pane is what fills the frame: glass, lit or dark, or a door.
	Pane *thing.Material
	// Frame is what the frame round the pane is made of.
	Frame *thing.Material
	// Shutter is what the panels beside the opening are made of, and
	// ShutterWidth how wide each is. Nought is no shutters.
	Shutter      *thing.Material
	ShutterWidth float64
	// Proud is how far the opening stands out of the wall, as a bay
	// window does. Nought is an opening recessed as the facade says.
	Proud float64
	// Door says the opening is a door, drawn with its panels and its step
	// at the demo level.
	Door bool
	// Lamp, if the window's light is on, is what the lamp behind its glass
	// is made of; Curtain, if it has one, is the drape's cloth, and Cover
	// the share of the window's width it covers. They are drawn at the
	// demo level.
	Lamp, Curtain *thing.Material
	Cover         float64
	// Heat is the colour temperature of the lamp's light, in kelvin.
	Heat float64
	// Home, if given, is the household whose window this is: the windows
	// of one dwelling share it, and it draws what stands before them.
	Home *household
	// Extra is what has been added to the opening, at the demo level.
	Extra extra
	// Arch says a door's head is a round arch, a half circle over the
	// door as wide as the door: the opening is that much taller, and its
	// upper corners are filled in.
	Arch bool
	// Garage says the opening is a garage door, drawn at the demo level as
	// sections one over another, each with raised panels.
	Garage bool
}

// facade is one face of a mass: its width and height, how deep the cuts
// for the openings are, the measures of its frames and glass, and the
// openings of each storey from the ground up. It stands with its origin
// at the foot of its left edge, on the plane of the wall, facing along
// positive z, and the wall is behind it.
type facade struct {
	Width, Height, Recess float64
	// FrameWidth is the width of the bars of a frame, GlassThickness
	// the thickness of the pane, and ShutterThickness of a shutter.
	FrameWidth, GlassThickness, ShutterThickness float64
	Wall                                         *thing.Material
	// Level is how much of the face is drawn: the hull is the wall alone,
	// the demo level every part, and any other the cut wall with whole
	// openings. Finish is the measures the demo level needs.
	Level  Detail
	Finish finish
	// Cavity is how deep behind the recess the wall is hollow for rooms at
	// the demo level, and Rooms what the rooms are made of. Nought is a
	// solid wall.
	Cavity float64
	Rooms  rooms
	// Ground is how much taller the ground storey is than the others, at
	// the demo level, and is part of the face's height.
	Ground float64
	// Fixtures is what has been added to the whole face, Iron the metal
	// of it, and Measure gives the size of any of it by the name the
	// culture's data has for it. All are drawn at the demo level.
	Fixtures fixtures
	Iron     *thing.Material
	// System is the method the face is built by, and what shows of it, at
	// the demo level. Its zero value shows nothing.
	System system
	// Upper says the face is a tier that stands on another's roof: it has
	// no plinth.
	Upper   bool
	Measure func(name string) float64
	// Storeys are the openings of each storey, the lowest first, each
	// in order along the face and apart from the next.
	Storeys [][]facadeOpening
}

// check refuses a facade with no storeys, a face of no size, and an
// opening that is off the face or lies over the next.
func (facade facade) check() error {
	if len(facade.Storeys) == 0 || !(facade.Width > 0 &&
		facade.Height > 0 && facade.Recess > 0) {
		return fmt.Errorf("cities: a facade needs a size, a recess and a "+
			"storey, not %+v", facade.Width)
	}
	for index, openings := range facade.Storeys {
		storey := facade.heightOf(index)
		right := 0.0
		for _, opening := range openings {
			if opening.Left < right || opening.Left+opening.Width >
				facade.Width || opening.Sill+opening.Height > storey {
				return fmt.Errorf("cities: an opening at %v of width %v "+
					"is off the face or over its neighbour",
					opening.Left, opening.Width)
			}
			right = opening.Left + opening.Width
		}
	}
	return nil
}

// parts is the facade drawn: the wall's skin cut round the openings,
// then the openings, one storey a unit of a run up the face.
func (facade facade) parts() ([]placedElement, error) {
	if err := facade.check(); err != nil {
		return nil, err
	}
	if facade.Level == hullLevel {
		return facade.slab(), nil
	}
	var wall, filled []placedElement
	for index, openings := range facade.Storeys {
		storey := facade.heightOf(index)
		stand := standingAt(maths.V(0, facade.floorAt(index), 0))
		wall = append(wall, within(facade.skin(openings, storey), stand)...)
		filled = append(filled, within(facade.fill(openings, storey),
			stand)...)
		filled = append(filled, within(facade.system(openings, storey),
			stand)...)
	}
	if facade.Level == demoLevel {
		filled = append(filled, facade.cornice()...)
		filled = append(filled, facade.fixtures()...)
		if !facade.Upper {
			filled = append(filled, facade.plinth()...)
		}
	}
	return append(wall, filled...), nil
}

// storeyHeight is the height of every storey above the ground storey, and
// of the ground storey but for the extra it is given.
func (facade facade) storeyHeight() float64 {
	return (facade.Height - facade.Ground) / float64(len(facade.Storeys))
}

// heightOf is the height of the storey at an index, counted from the
// ground storey at nought, which is taller by the extra it has.
func (facade facade) heightOf(index int) float64 {
	if index == 0 {
		return facade.storeyHeight() + facade.Ground
	}
	return facade.storeyHeight()
}

// floorAt is the height of the floor of the storey at an index.
func (facade facade) floorAt(index int) float64 {
	if index == 0 {
		return 0
	}
	return facade.Ground + float64(index)*facade.storeyHeight()
}

// skin is the wall of one storey, in boxes as deep as the recess: the
// full height between the openings and at the edges, and over and under
// each opening what it leaves.
func (facade facade) skin(openings []facadeOpening, storey float64) []placedElement {
	var boxes []placedElement
	back := -facade.shell()
	right := 0.0
	for _, opening := range openings {
		boxes = append(boxes, facade.column(right, opening.Left, storey)...)
		if opening.Sill > 0 {
			boxes = append(boxes, put("wall", Box{opening.Width, opening.Sill,
				facade.shell()}, facade.Wall, maths.V(opening.Left, 0, back)))
		}
		head := opening.Sill + opening.Height
		if above := storey - head; above > 0 {
			boxes = append(boxes, put("wall", Box{opening.Width, above,
				facade.shell()}, facade.Wall, maths.V(opening.Left, head, back)))
		}
		right = opening.Left + opening.Width
	}
	return append(boxes, facade.column(right, facade.Width, storey)...)
}

// column is a full storey of wall between two places along the face,
// or nothing if they are the same.
func (facade facade) column(from, to, storey float64) []placedElement {
	if !(to > from) {
		return nil
	}
	return []placedElement{put("wall", Box{to - from, storey, facade.shell()},
		facade.Wall, maths.V(from, 0, -facade.shell()))}
}

// fill is the openings of one storey drawn: each one's frame, pane and
// shutters, or its projecting box.
func (facade facade) fill(openings []facadeOpening, storey float64) []placedElement {
	var parts []placedElement
	for _, opening := range openings {
		if opening.Proud > 0 {
			parts = append(parts, facade.projecting(opening)...)
		} else {
			parts = append(parts, facade.recessed(opening)...)
		}
		parts = append(parts, facade.shutters(opening)...)
		if facade.Cavity > 0 {
			parts = append(parts, facade.behind(opening)...)
		}
		if facade.Level == demoLevel && !opening.Door && opening.Proud == 0 {
			parts = append(parts, facade.surround(opening, storey)...)
		}
		if facade.Level == demoLevel && opening.Proud == 0 {
			parts = append(parts, facade.added(opening)...)
		}
		if facade.Level == demoLevel && opening.Arch {
			parts = append(parts, facade.spandrels(opening)...)
		}
	}
	if facade.Level == demoLevel {
		parts = append(parts, facade.homes(openings)...)
	}
	return parts
}

// recessed is an opening cut into the wall: four bars of frame as deep
// as the cut, and the pane standing on the back of it.
func (facade facade) recessed(opening facadeOpening) []placedElement {
	bar, recess := facade.FrameWidth, facade.frameDepth()
	left, foot := opening.Left, opening.Sill
	inner := opening.Height - 2*bar
	back := -facade.Recess
	parts := []placedElement{
		put("frame", Box{opening.Width, bar, recess}, opening.Frame,
			maths.V(left, foot, back)),
		put("frame", Box{opening.Width, bar, recess}, opening.Frame,
			maths.V(left, foot+opening.Height-bar, back)),
		put("frame", Box{bar, inner, recess}, opening.Frame,
			maths.V(left, foot+bar, back)),
		put("frame", Box{bar, inner, recess}, opening.Frame,
			maths.V(left+opening.Width-bar, foot+bar, back)),
	}
	switch {
	case facade.Level != demoLevel:
		return append(parts, put("pane", Box{opening.Width - 2*bar, inner,
			facade.GlassThickness}, opening.Pane,
			maths.V(left+bar, foot+bar, back)))
	case opening.Garage:
		return append(parts, facade.sectional(opening)...)
	case opening.Door:
		return append(parts, facade.leaf(opening)...)
	}
	return append(parts, facade.lights(opening)...)
}

// projecting is a bay window: a box in the frame's material standing out
// of the wall by the opening's proud, with the pane on its front face.
func (facade facade) projecting(opening facadeOpening) []placedElement {
	bar := facade.FrameWidth
	return []placedElement{
		put("bay", Box{opening.Width, opening.Height,
			opening.Proud + facade.Recess}, opening.Frame,
			maths.V(opening.Left, opening.Sill, -facade.Recess)),
		put("pane", Box{opening.Width - 2*bar, opening.Height - 2*bar,
			facade.GlassThickness}, opening.Pane,
			maths.V(opening.Left+bar, opening.Sill+bar, opening.Proud)),
	}
}

// shutters are the two panels beside an opening, standing on the wall,
// as tall as the opening. An opening with none has none.
func (facade facade) shutters(opening facadeOpening) []placedElement {
	if !(opening.ShutterWidth > 0) {
		return nil
	}
	if facade.Level == demoLevel {
		return append(facade.slatted(opening.Left-opening.ShutterWidth,
			opening.Sill, opening.Height, opening),
			facade.slatted(opening.Left+opening.Width, opening.Sill,
				opening.Height, opening)...)
	}
	panel := Box{opening.ShutterWidth, opening.Height,
		facade.ShutterThickness}
	return []placedElement{
		put("shutter", panel, opening.Shutter, maths.V(
			opening.Left-opening.ShutterWidth, opening.Sill, 0)),
		put("shutter", panel, opening.Shutter, maths.V(
			opening.Left+opening.Width, opening.Sill, 0)),
	}
}

// spread is the places along a face, between two, that a run of things
// of the widths given stand at, with the same gap before the first, after
// the last and between each; and that gap.
func spread(from, to float64, widths []float64) (lefts []float64,
	gap float64) {
	room := to - from
	for _, width := range widths {
		room -= width
	}
	gap = room / float64(len(widths)+1)
	next := from + gap
	for _, width := range widths {
		lefts = append(lefts, next)
		next += width + gap
	}
	return lefts, gap
}
