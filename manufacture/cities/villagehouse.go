package cities

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/scene/thing"
)

// A village house is the walk-up of an urban village: a concrete frame of
// six to ten storeys built to the edges of a small plot, faced in small
// tiles, flat roofed, with windows on every wall, since every wall looks
// onto an alley or a lane.
//
// Its windows are aluminium in a grid, many behind a steel cage that
// stands out of the wall, with an air conditioner under or beside them,
// its drip pipe hanging under it, and clothes poles out over the alley.
// On the roof are water tanks, a stair head, a shack or planters.
//
// On a wall that faces a lane the ground storey is a shop, open and lit
// or behind a roller shutter, and the signs of the street hang there. The
// stair door has a lamp over it, so each house lights its own alley.
//
// A house's draws are made at the place whose row is the house: a row
// scope element is the house's, a bay scope one the wall's, and below
// that the storeys and openings of the wall. Every number is the
// culture's, recalled or picked, and none is measured.

// villageCulture is the culture a village and its houses are drawn from.
const villageCulture = "village"

// shopfront is what a shop's opening says is under it, so that it is told
// from a window; the facade draws nothing for a word it does not know.
const shopfront = "shopfront"

// signShift is how far a house's number is shifted up in the seed of its
// signs, past the digits a district shares, so that no two houses of a
// village hang the same signs.
const signShift = 16

// tiered is a part of a house and the coarsest level it is drawn at.
type tiered struct {
	placedElement
	least Detail
}

// villageHouse is one house drawn and ready to stand at a level: its lot,
// its footing on the ground, its height and storeys, the walls that face
// a lane, and what it is made of. Lot is in the frame of what holds it.
type villageHouse struct {
	Lot     rect
	Footing groundFooting
	Height  float64
	Storeys int
	Lanes   [sidesAround]bool
	faces   [sidesAround]facade
	shell   float64
	wall    *thing.Material
	tiers   []tiered
	signs   Seed
}

// houseJob is a house being drawn: the drawer, the place its draws are
// made at, the house so far, the wall its stair door is in, the height of
// a storey and the extra of the ground storey, and the steel of its cages.
type houseJob struct {
	drawing        *drawer
	at             place
	house          *villageHouse
	door           int
	rise, ground   float64
	recess, cavity float64
	steel          *thing.Material
}

// VillageHouseAt is one village house standing alone, its lot centred on
// the origin and its front, the wall to positive z, on a lane, drawn at a
// level from a seed and stood on the ground given, or on flat ground if
// none. It refuses a level a building is not drawn at.
func VillageHouseAt(seed Seed, ground Terrain, level Detail) (Design,
	error) {
	drawing, err := villageDrawer(seed, level)
	if err != nil {
		return Design{}, err
	}
	width := drawing.number("plot-size", place{Row: acrossX})
	depth := drawing.number("plot-size", place{Row: acrossZ})
	lot := rect{-width / half, -depth / half, width / half, depth / half}
	lanes := [sidesAround]bool{frontSide: true}
	house, err := drawing.villageHouse(0, lot, lanes, ground, seed)
	if err != nil {
		return Design{}, err
	}
	return Design{Name: "village house",
		Elements: []placedElement{house.placed()}}, nil
}

// villageDrawer is the drawer of the village culture for a seed, drawing
// at a level: the quick level if none is given. It refuses a level that
// is not one a building is drawn at.
func villageDrawer(seed Seed, level Detail) (*drawer, error) {
	if level == "" {
		level = quickLevel
	}
	if !slices.Contains(facadeLevels, level) {
		return nil, fmt.Errorf("cities: a village is drawn at one of %v, "+
			"not %q", facadeLevels, level)
	}
	drawing, err := newDrawer(villageCulture, seed)
	if err != nil {
		return nil, err
	}
	drawing.level = level
	return drawing, nil
}

// villageHouse is the house of a number on a lot, with the walls that
// face a lane, on the ground given or on flat ground, drawn at the
// drawer's level: its shape, its four walls and what is fitted to them,
// and its roof.
func (drawing *drawer) villageHouse(index int, lot rect,
	lanes [sidesAround]bool, ground Terrain, seed Seed) (villageHouse,
	error) {
	footing, err := footingUnder(lot, ground)
	if err != nil {
		return villageHouse{}, err
	}
	house := &villageHouse{Lot: lot, Footing: footing, Lanes: lanes,
		signs: Seed(uint64(seed)<<signShift + uint64(index))}
	job := houseJob{drawing: drawing, at: place{Row: index}, house: house}
	job.shape()
	job.walls()
	job.roofed()
	return *house, drawing.trouble
}

// footingUnder is the footing of a lot on the ground, or the footing of
// flat ground where there is none.
func footingUnder(lot rect, ground Terrain) (groundFooting, error) {
	if ground == nil {
		return groundFooting{}, nil
	}
	return footingOn(lot, ground)
}

// shape draws the house's storeys and their heights, its tiles, how deep
// its walls are, which wall its stair door is in and its cages' steel.
func (job *houseJob) shape() {
	drawing, house := job.drawing, job.house
	house.Storeys = drawing.count("storeys", job.at)
	job.rise = drawing.number("storey-height", job.at)
	job.ground = drawing.number("ground-rise", job.at)
	house.Height = float64(house.Storeys)*job.rise + job.ground
	house.wall = drawing.tile(job.at)
	job.recess = drawing.recess()
	if drawing.level == demoLevel {
		narrow := math.Min(house.Lot.width(), house.Lot.depth()) / half
		job.cavity = math.Max(math.Min(drawing.fixed("room-depth"),
			narrow-job.recess-drawing.fixed("back-wall")), 0)
	}
	house.shell = job.recess + job.cavity
	job.door = job.doorSide()
	job.steel = paint("steel cage", drawing.pick("cage-colour",
		drawing.options("cage-colour"), job.at))
}

// doorSide is the wall the stair door is in: the first that faces a lane,
// front, east, back and west in turn, or one the seed draws.
func (job *houseJob) doorSide() int {
	for side, lane := range job.house.Lanes {
		if lane {
			return side
		}
	}
	return job.drawing.count("door-side", job.at)
}

// tile is the tile a house is faced in: one of the culture's colours,
// darkened by the grime of its age, held to steps so that a village has
// a few tiles and not one for every house.
func (drawing *drawer) tile(at place) *thing.Material {
	colour := drawing.pick("tile-colour", drawing.options("tile-colour"),
		at)
	step := drawing.fixed("grime-step")
	grime := math.Round(drawing.number("tile-grime", at)/step) * step
	return &thing.Material{Name: fmt.Sprintf("cities tile %s %.2f",
		colour.Name, grime), Colour: colourOf(colour).Scale(grime),
		Gloss: colour.Gloss}
}

// walls draws the face of each wall and what is fitted to it.
func (job *houseJob) walls() {
	for side := range sidesAround {
		frame, width := wallFrame(job.house.Lot, side, job.house.shell)
		face := job.facade(side, width)
		job.house.faces[side] = face
		job.fit(side, frame, face)
	}
}

// facade is the face of one wall, as wide as given: the measures every
// face shares, at the demo level its rooms, finish and fixtures, and the
// openings of each storey.
func (job *houseJob) facade(side int, width float64) facade {
	drawing := job.drawing
	face := facade{Width: width, Height: job.house.Height,
		Recess: job.recess, Ground: job.ground, Level: drawing.level,
		FrameWidth:       drawing.fixed("frame-width"),
		GlassThickness:   drawing.fixed("glass-thickness"),
		ShutterThickness: drawing.fixed("shutter-thickness"),
		Wall:             job.house.wall, Measure: drawing.fixed}
	if drawing.level == demoLevel {
		wall := place{Row: job.at.Row, Bay: side}
		face.Finish, face.Rooms = drawing.finish(), drawing.rooms()
		face.Cavity, face.Fixtures = job.cavity, drawing.fixtures(wall)
		face.Iron = job.steel
	}
	face.Storeys = make([][]facadeOpening, job.house.Storeys)
	for storey := range face.Storeys {
		at := place{Row: job.at.Row, Bay: side, Storey: storey}
		face.Storeys[storey] = job.storey(at, side, width)
	}
	return face
}

// storey is the openings of one storey of a wall, laid along it: on the
// ground storey the stair door if the wall has it and a shop if the wall
// faces a lane and the draw says so, and windows everywhere else.
func (job *houseJob) storey(at place, side int, width float64) []facadeOpening {
	var items []facadeOpening
	if at.Storey == 0 && side == job.door {
		items = append(items, job.drawing.door(at))
	}
	// a gap before each opening and one after the shop
	room := width - 2*job.drawing.fixed("window-edge") -
		float64(len(items)+shopGaps-1)*job.drawing.fixed("min-gap") -
		meeting
	for _, item := range items {
		room -= item.span()
	}
	if at.Storey == 0 && job.house.Lanes[side] &&
		job.drawing.chance("shop", at) &&
		room >= job.drawing.fixed("shop-least") {
		return job.laid(append(items, job.shop(at, room)), width)
	}
	return job.laid(append(items, job.windows(at, width,
		len(items))...), width)
}

// windows are the windows of a storey of a wall as wide as given: as many
// as the culture's pitch fits between the edges, less the places already
// taken.
func (job *houseJob) windows(at place, width float64,
	taken int) []facadeOpening {
	drawing := job.drawing
	edge, pitch := drawing.fixed("window-edge"),
		drawing.number("window-pitch", at)
	count := int((width-2*edge)/pitch) - taken
	windows := make([]facadeOpening, max(count, 0))
	for index := range windows {
		at.Opening = taken + index
		windows[index] = drawing.window(at)
	}
	return windows
}

// shop is the opening of a shop as wide as the room given: low in the
// wall and tall, and open and lit, or behind a roller shutter. Open, at
// the demo level, it has the room and lamp behind it a window has.
func (job *houseJob) shop(at place, room float64) facadeOpening {
	drawing := job.drawing
	frames := drawing.options("frame-colour")
	shop := facadeOpening{Width: room, Height: drawing.fixed("shop-height"),
		Sill: drawing.fixed("shop-sill"), Extra: extra{Under: shopfront},
		Frame: paint("frame", drawing.pick("frame-colour", frames, at)),
		Pane: paint("roller shutter", drawing.pick("roller-colour",
			drawing.options("roller-colour"), at))}
	if !drawing.chance("shop-open", at) {
		return shop
	}
	dark, err := material(roleGlazing)
	drawing.fail(err)
	if dark == nil {
		return shop
	}
	shop.Pane = drawing.litGlass(dark, at)
	if drawing.level == demoLevel {
		drawing.furnish(&shop, at)
		shop.Curtain = nil
	}
	return shop
}

// laid is a storey's openings set along a wall as wide as given, between
// the culture's edges, the same gap apart; if they do not fit with the
// least gap between them, openings are dropped from the end until they
// do.
func (job *houseJob) laid(items []facadeOpening, width float64) []facadeOpening {
	edge := job.drawing.fixed("window-edge")
	least := job.drawing.fixed("min-gap")
	for ; len(items) > 0; items = items[:len(items)-1] {
		widths := make([]float64, len(items))
		for index, item := range items {
			widths[index] = item.span()
		}
		lefts, gap := spread(edge, width-edge, widths)
		if gap < least {
			continue
		}
		for index := range items {
			items[index].Left = lefts[index]
		}
		return items
	}
	return nil
}
