package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// rooms are the surfaces of the rooms behind a facade's windows at the
// demo level: the walls and ceiling, and the floor.
type rooms struct{ Wall, Floor *thing.Material }

// rooms draws the surfaces of the rooms behind a face from the culture.
func (drawing *drawer) rooms() rooms {
	at := place{}
	return rooms{
		Wall: paint("room wall", drawing.pick("room-wall-colour",
			drawing.options("room-wall-colour"), at)),
		Floor: drawing.timbered(paint("room floor", drawing.pick(
			"room-floor-colour", drawing.options("room-floor-colour"), at)))}
}

// glazing is clear glass at the demo level: it mirrors the sky and lets
// the room behind it through, in place of a pane that is dark or glows.
func (drawing *drawer) glazing() *thing.Material {
	tone := drawing.fixed("glazing-tone")
	return &thing.Material{Name: "cities clear glazing",
		Colour: maths.RGB(tone, tone, tone), Gloss: 1, Through: 1}
}

// furnish gives a window at the demo level what is behind its glass: if
// its light is on, a lamp that glows as the window did, and, by chance, a
// curtain over part of it. The glass is clear, so the room is seen
// through it.
func (drawing *drawer) furnish(opening *facadeOpening, at place) {
	if glow := opening.Pane.Glow; glow != (maths.Light{}) {
		opening.Lamp = &thing.Material{Name: opening.Pane.Name + " lamp",
			Colour: maths.RGB(1, 1, 1),
			Glow:   glow.Scale(drawing.fixed("lamp-glow"))}
		opening.Heat = drawing.heat(at)
	}
	opening.Pane = drawing.glazing()
	if drawing.chance("curtain", at) {
		opening.Curtain = paint("curtain", drawing.pick("curtain-colour",
			drawing.options("curtain-colour"), at))
		opening.Cover = drawing.number("curtain-cover", at)
	}
}

// cavityFor is how deep the rooms behind a face are cut into a house of a
// depth: the culture's depth, less if the house is shallow, and nothing
// if the house has not that and a back wall.
func (facade facade) cavityFor(depth float64) float64 {
	room := min(facade.Finish.RoomDepth,
		depth-facade.Recess-facade.Finish.BackWall)
	return max(room, 0)
}

// shell is how deep the wall of a face is: the recess and the rooms.
func (facade facade) shell() float64 { return facade.Recess + facade.Cavity }

// behind is what stands in the cut of an opening, behind the recess: a
// room, if the opening is a window that has one, else a plug of wall.
func (facade facade) behind(opening facadeOpening) []placedElement {
	if opening.Door || opening.Proud > 0 {
		return []placedElement{put("wall", Box{opening.Width, opening.Height,
			facade.Cavity}, facade.Wall, maths.V(opening.Left, opening.Sill,
			-facade.shell()))}
	}
	return facade.interior(opening)
}

// interior is the room behind a window, in the cut: a floor, a ceiling, a
// back wall and two side walls, as thick as the finish says, with the lamp
// hung from the ceiling and the curtain just behind the glass.
func (facade facade) interior(opening facadeOpening) []placedElement {
	fine, rooms := facade.Finish, facade.Rooms
	slab, deep := fine.RoomSlab, facade.Cavity
	left, foot := opening.Left, opening.Sill
	back := -facade.shell()
	tall := opening.Height - 2*slab
	parts := []placedElement{
		put("floor", Box{opening.Width, slab, deep}, rooms.Floor,
			maths.V(left, foot, back)),
		put("ceiling", Box{opening.Width, slab, deep}, rooms.Wall,
			maths.V(left, foot+opening.Height-slab, back)),
		put("back wall", Box{opening.Width, tall, slab}, rooms.Wall,
			maths.V(left, foot+slab, back)),
		put("side wall", Box{slab, tall, deep - slab}, rooms.Wall,
			maths.V(left, foot+slab, back+slab)),
		put("side wall", Box{slab, tall, deep - slab}, rooms.Wall,
			maths.V(left+opening.Width-slab, foot+slab, back+slab)),
	}
	return append(append(parts, facade.lamp(opening)...),
		facade.curtain(opening)...)
}

// lamp is the light hung from a room's ceiling, if the window's is on.
func (facade facade) lamp(opening facadeOpening) []placedElement {
	if opening.Lamp == nil {
		return nil
	}
	fine := facade.Finish
	at := maths.V(opening.Left+(opening.Width-fine.LampSize)/half,
		opening.Sill+opening.Height-fine.RoomSlab-fine.LampDrop-fine.LampSize,
		-facade.Recess-facade.Cavity/half)
	return []placedElement{{Place: standingAt(at), Element: solid{Name: "lamp",
		Form:     Box{fine.LampSize, fine.LampSize, fine.LampSize},
		Material: opening.Lamp,
		Lamp:     &Lamp{Heat: opening.Heat, Luminance: fine.LampStrength}}}}
}

// curtain is a drape just behind the glass, from one side of the window
// over the share of its width the opening says. Which side is by where
// the window stands along the face, so neighbours differ.
func (facade facade) curtain(opening facadeOpening) []placedElement {
	if opening.Curtain == nil {
		return nil
	}
	fine, bar := facade.Finish, facade.FrameWidth
	inner := opening.Width - 2*bar
	width := inner * opening.Cover
	left := opening.Left + bar
	if int(opening.Left*curtainSideScale)%half == 1 {
		left += inner - width
	}
	return []placedElement{put("curtain", Box{width, opening.Height - 2*bar,
		fine.CurtainThickness}, opening.Curtain, maths.V(left,
		opening.Sill+bar, -facade.Recess-fine.CurtainThickness))}
}

// curtainSideScale turns a window's place along the face into a whole
// number whose parity says which side its curtain hangs from. A choice,
// of a decimetre: windows a decimetre apart do not agree.
const curtainSideScale = 10
