package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// house is one mass with its face and its roof: a house in a row, one
// building of a block of flats, the garage beside a house, a tower. It
// stands with its origin at the back left corner on the ground, its
// width along x, its depth along z, and its front, with the openings,
// on the plane at z equal to Depth.
type house struct {
	// Depth is how far the mass reaches from its back to its front.
	Depth float64
	// Facade is the front, which says the width and the height.
	Facade facade
	// Roof stands over the footprint, which it must match.
	Roof roof
}

// parts is the house drawn: the wall behind the facade's cuts, the
// facade, and the roof on the wall's top.
func (house house) parts() ([]placedElement, error) {
	facade := house.Facade
	if facade.Level == demoLevel {
		facade.Cavity = facade.cavityFor(house.Depth)
	}
	if !(house.Depth > facade.Recess) {
		return nil, fmt.Errorf("cities: a house must be deeper than the "+
			"cuts in its front, not %v against %v", house.Depth,
			facade.Recess)
	}
	front, err := facade.parts()
	if err != nil {
		return nil, err
	}
	roof, err := house.Roof.parts()
	if err != nil {
		return nil, err
	}
	body := put("wall", Box{facade.Width, facade.Height,
		house.Depth - facade.shell()}, facade.Wall, maths.Vec{})
	parts := append([]placedElement{body}, within(front,
		standingAt(maths.V(0, 0, house.Depth)))...)
	return append(parts, within(roof, standingAt(
		maths.V(0, facade.Height, 0)))...), nil
}

// massShape is what a house is before its looks are drawn: its size and
// storeys, which way its door faces, and how far along its front the
// openings may stand.
type massShape struct {
	width, depth float64
	storeys      int
	door         doorKind
	// from is where along the front the openings begin, which is past
	// whatever else stands on the face, a stair core.
	from  float64
	tower bool
	// reach is how far the front of the highest storeys stands back from
	// the front of the lowest, where a building steps back: a stair core
	// is as much deeper, to reach back to them.
	reach float64
}

// doorKind is where a house has its door.
type doorKind int

const (
	// doorNone is a house with no door of its own on this mass.
	doorNone doorKind = iota
	// doorFront is a door in the front.
	doorFront
	// doorSide is a door in the side wall, on the free side.
	doorSide
)

// fail keeps the first thing that went wrong, which a drawer does not
// itself notice: an error from outside the culture.
func (drawing *drawer) fail(err error) {
	if drawing.trouble == nil {
		drawing.trouble = err
	}
}

// window is one window at a place: its size, its glass, lit or dark,
// its frame, and shutters if the place has them.
func (drawing *drawer) window(at place) facadeOpening {
	pane, err := drawing.glass(at)
	drawing.fail(err)
	frames := drawing.options("frame-colour")
	opening := facadeOpening{Width: drawing.number("window-width", at),
		Height: drawing.number("window-height", at),
		Sill:   drawing.fixed("window-sill"), Pane: pane,
		Frame: paint("frame", drawing.pick("frame-colour", frames, at))}
	if drawing.chance("shutters", at) {
		greens := drawing.options("shutter-colour")
		opening.Shutter = drawing.timbered(paint("shutters",
			drawing.pick("shutter-colour", greens, at)))
		opening.ShutterWidth = drawing.fixed("shutter-width")
	}
	if drawing.level == demoLevel {
		drawing.furnish(&opening, at)
		if !drawing.homes {
			drawing.fitted(&opening, at)
		}
	}
	return opening
}

// door is a door at a place, standing on the floor, in a frame.
func (drawing *drawer) door(at place) facadeOpening {
	frames := drawing.options("frame-colour")
	doors := drawing.options("door-colour")
	opening := facadeOpening{Width: drawing.fixed("door-width"),
		Height: drawing.fixed("door-height"), Door: true,
		Pane: drawing.timbered(paint("door", drawing.pick("door-colour",
			doors, at))),
		Frame: paint("frame", drawing.pick("frame-colour", frames, at))}
	if drawing.level == demoLevel {
		drawing.fittedDoor(&opening, at)
		if drawing.chance("arched-door", at) {
			opening.Arch = true
			opening.Height += opening.Width / half
		}
	}
	return opening
}

// shopfront is the one wide window of a ground storey that is a shop: as
// wide as the room beside the door leaves, tall, and low in the wall, with
// no shutters and nothing under it. It is nothing if there is no room.
func (drawing *drawer) shopfront(at place, shape massShape) []facadeOpening {
	room := shape.width - shape.from - drawing.fixed("door-width") -
		shopGaps*drawing.fixed("min-gap")
	if room < drawing.fixed("shop-least") {
		return nil
	}
	opening := drawing.window(at)
	opening.Width, opening.Height = room, drawing.fixed("shop-height")
	opening.Sill = drawing.fixed("shop-sill")
	opening.Shutter, opening.ShutterWidth = nil, 0
	opening.Extra = extra{}
	return []facadeOpening{opening}
}

// shopGaps is how many gaps the door and a shopfront leave between them and
// the edges of the face: one at each edge and one between.
const shopGaps = 3

// windows are the windows of a storey, as many as the culture draws for
// it, or the one bay window of a ground storey that has one.
func (drawing *drawer) windows(at place, shape massShape) []facadeOpening {
	if drawing.homes {
		return drawing.dwellings(at)
	}
	if at.Storey == 0 && drawing.chance("bay-window", at) {
		opening := drawing.window(at)
		opening.Width = drawing.fixed("bay-window-width")
		opening.Proud = drawing.fixed("bay-window-proud")
		return []facadeOpening{opening}
	}
	if at.Storey == 0 && drawing.level == demoLevel &&
		shape.door == doorFront && drawing.chance("shopfront", at) {
		if shop := drawing.shopfront(at, shape); shop != nil {
			return shop
		}
	}
	count := drawing.count("window-count", at)
	if shape.tower {
		count = drawing.count("tower-window-count", at)
	}
	windows := make([]facadeOpening, count)
	for index := range windows {
		at.Opening = index
		windows[index] = drawing.window(at)
	}
	return windows
}

// garageDoor is the door of a garage that takes the front of the ground
// storey, standing on the floor in its frame.
func (drawing *drawer) garageDoor(at place) facadeOpening {
	frames := drawing.options("frame-colour")
	doors := drawing.options("door-colour")
	return facadeOpening{Width: drawing.fixed("garage-door-width"),
		Height: drawing.fixed("garage-door-height"),
		Pane:   paint("door", drawing.pick("door-colour", doors, at)),
		Frame:  paint("frame", drawing.pick("frame-colour", frames, at))}
}

// span is the width an opening takes along the face, shutters and all.
func (opening facadeOpening) span() float64 {
	return opening.Width + 2*opening.ShutterWidth
}

// storey is the openings of one storey of a front, laid out along it: a
// door first on the ground storey if the shape has one in front, then
// the windows, all the same gap apart. If they do not fit with the
// culture's least gap between them, windows are dropped from the end
// until they do.
func (drawing *drawer) storey(at place, shape massShape) []facadeOpening {
	items := drawing.windows(at, shape)
	if at.Storey == 0 && shape.door == doorFront {
		items = append([]facadeOpening{drawing.door(at)}, items...)
	}
	if at.Storey == 0 && drawing.has("garage-under") &&
		drawing.chance("garage-under", at) {
		items = append([]facadeOpening{drawing.garageDoor(at)}, items...)
	}
	least := drawing.fixed("min-gap")
	for ; len(items) > 0; items = items[:len(items)-1] {
		widths := make([]float64, len(items))
		for index, item := range items {
			widths[index] = item.span()
		}
		lefts, gap := spread(shape.from, shape.width, widths)
		if gap < least {
			continue
		}
		for index := range items {
			items[index].Left = lefts[index] + items[index].ShutterWidth
		}
		return items
	}
	return nil
}

// facade is the front of a mass drawn: the openings of each storey, and
// the measures every front shares.
func (drawing *drawer) facade(at place, shape massShape,
	wall *thing.Material) facade {
	height := float64(shape.storeys) * drawing.number("storey-height", at)
	storeys := make([][]facadeOpening, shape.storeys)
	for index := range storeys {
		at.Storey = index + drawing.storeyBase
		storeys[index] = drawing.storey(at, shape)
	}
	var system system
	if drawing.homes {
		system = drawing.system(at, wall)
	}
	wall = drawing.plastered(wall)
	var ground float64
	if drawing.level == demoLevel {
		ground = drawing.fixed("ground-extra")
		height += ground
	}
	finish, rooms := finish{}, rooms{}
	var fixtures fixtures
	var iron *thing.Material
	if drawing.level == demoLevel {
		finish, rooms = drawing.finish(), drawing.rooms()
		fixtures = drawing.fixtures(at)
		iron = paint("iron", drawing.pick("iron-colour",
			drawing.options("iron-colour"), at))
	}
	return facade{Width: shape.width, Height: height, Level: drawing.level,
		Finish: finish, Rooms: rooms, Fixtures: fixtures, Iron: iron,
		Ground: ground, System: system,
		Measure:          drawing.fixed,
		Recess:           drawing.recess(),
		FrameWidth:       drawing.fixed("frame-width"),
		GlassThickness:   drawing.fixed("glass-thickness"),
		ShutterThickness: drawing.fixed("shutter-thickness"),
		Wall:             wall, Storeys: storeys}
}

// roofRun is the horizontal distance a roof of a form rises over.
func roofRun(roof roof) float64 {
	switch roof.Form {
	case gableAcross:
		return roof.Width / 2
	case gableAlong:
		return roof.Depth / 2
	case monoUp, monoDown:
		return roof.Depth
	case hip:
		return min(roof.Width, roof.Depth) / 2
	}
	return 0
}

// roof is the roof of a mass drawn: its form, its colour, its eave, its
// pitch. A flat roof has no eaves, and a parapet; a sloped one has an
// eave, and the pitch says how steeply it rises over its run.
func (drawing *drawer) roof(at place, shape massShape,
	gable *thing.Material) roof {
	forms := drawing.options("roof-form")
	covers := drawing.options("roof-colour")
	roof := roof{Form: roofForm(drawing.pick("roof-form", forms, at).Name),
		Width: shape.width, Depth: shape.depth,
		Slab: drawing.fixed("slab-thickness"), Gable: gable,
		Cover: paint("roof", drawing.pick("roof-colour", covers, at))}
	if roof.Form == roofFlat {
		roof.Parapet = drawing.fixed("parapet-height")
		roof.ParapetWall = drawing.fixed("parapet-wall")
		return roof
	}
	roof.Eave = drawing.number("eave-depth", at)
	roof.Rise = drawing.number("roof-pitch", at) * roofRun(roof)
	least := min(roof.Width, roof.Depth)/2 + roof.Eave
	roof.Inset = drawing.number("hip-point", at) * least
	return roof
}

// house is a mass drawn at a place: its wall, its front and its roof.
func (drawing *drawer) house(at place, shape massShape,
	wall *thing.Material) house {
	front := drawing.facade(at, shape, wall)
	return house{Depth: shape.depth, Facade: front,
		Roof: drawing.roof(at, shape, wall)}
}
