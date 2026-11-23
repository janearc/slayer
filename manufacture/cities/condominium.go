package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// condominiumCulture is the culture a condominium is drawn from: the
// one that says how tall its buildings are and how they stand about
// their court.
const condominiumCulture = "condominium"

// half is how many times a distance across is its distance from the
// middle: a court's sides stand half its width from its centre.
const half = 2

// court is the ground the buildings of a condominium share: its size,
// and the buildings drawn for it.
type court struct {
	width, depth float64
	colours      []option
	cladding     []option
	ground       Terrain
	// mounts collects the places the court's buildings keep a thing of a
	// kind the manufactory makes, a telescope on a roof.
	mounts *[]Mount
}

// Condominium is two to four buildings of stacked flats, four or five
// storeys each, drawn together around one court. The buildings stand on
// its sides, north first, then south, east and west, each with its front
// turned to the court. Each has a stair core, a vertical mass standing
// a little proud of its front, with the entrance in its foot and a
// slit of window at each landing.
//
// The court is paved and shared: no building has ground of its own. The
// north and south buildings are as long as the court is wide, and the
// east and west buildings as long as it is deep, so that the four make a
// block with the court at its middle.
func Condominium(seed Seed) (Design, error) {
	return CondominiumOn(seed, nil)
}

// CondominiumOn is a condominium on a slope: the court and each building
// stood on the ground under its own footprint, so that they step down a
// hillside. The ground is in the condominium's frame, its court's middle
// at the origin. With no ground it is the condominium on flat ground.
func CondominiumOn(seed Seed, ground Terrain) (Design, error) {
	return CondominiumAt(seed, ground, "")
}

// CondominiumAt is a condominium on a slope with its faces drawn at a
// detail level, as terraceAtDetail draws a terrace's.
func CondominiumAt(seed Seed, ground Terrain, level Detail) (Design, error) {
	drawing, err := newDrawer(condominiumCulture, seed)
	if err != nil {
		return Design{}, err
	}
	drawing.level = level
	drawing.homes = level == demoLevel
	at := place{}
	shared := court{width: drawing.number("court-width", at),
		depth: drawing.number("court-depth", at), ground: ground,
		mounts: new([]Mount)}
	count := drawing.count("buildings", at)
	shared.colours = drawing.colours(count)
	shared.cladding = drawing.options("cladding")
	parts := shared.paving(drawing)
	for index := range count {
		building := shared.building(drawing, index)
		parts = append(parts, building...)
	}
	if drawing.trouble != nil {
		return Design{}, drawing.trouble
	}
	return Design{Name: "condominium", Elements: parts,
		Mounts: *shared.mounts}, nil
}

// paving is the court's floor: a thin slab, in the paving's colour.
func (shared court) paving(drawing *drawer) []placedElement {
	slab := drawing.options("paving-colour")
	look := paint("paving", drawing.pick("paving-colour", slab, place{}))
	form := Box{shared.width, drawing.fixed("paving-thickness"),
		shared.depth}
	paved := []placedElement{put("court", form, look,
		maths.V(-shared.width/half, 0, -shared.depth/half))}
	if drawing.homes {
		paved = append(paved, shared.furnish(drawing)...)
	}
	parts, err := plant(paved, rect{-shared.width / half,
		-shared.depth / half, shared.width / half, shared.depth / half},
		shared.ground)
	drawing.fail(err)
	return parts
}

// building is the index'th building of the court: drawn in its own
// frame, with its front to positive z, and then stood on its side of the
// court, turned to face it.
func (shared court) building(drawing *drawer, index int) []placedElement {
	at := place{Bay: index}
	length := shared.width
	if index >= 2 {
		length = shared.depth
	}
	shape := shared.shapeOf(drawing, at, length)
	wall := wall(shared.colours[index], drawing.pick("cladding",
		shared.cladding, at))
	built := shared.stacked(drawing, at, shape, wall)
	shape.reach = built.reach
	parts := append(built.parts, shared.stair(drawing, at, shape, wall)...)
	if drawing.homes {
		parts = append(parts, shared.topAndBase(drawing, at, shape,
			built)...)
	}
	return shared.stood(drawing, index, shape, parts, built)
}

// shapeOf is the shape of a building along one side of the court, drawn
// from the culture: its length, depth and storeys, with its stair core
// at the left. A tall building, at the demo level, has more storeys and
// stands back from its front as it rises.
func (shared court) shapeOf(drawing *drawer, at place,
	length float64) massShape {
	shape := massShape{width: length, door: doorNone,
		depth:   drawing.number("building-depth", at),
		storeys: drawing.count("storeys", at)}
	shape.from = drawing.fixed("core-offset") + drawing.fixed("core-width") +
		drawing.fixed("min-gap")
	if drawing.homes && drawing.chance("tall-share", at) {
		shape.storeys = drawing.count("tall-storeys", at)
	}
	return shape
}

// stood is a building's parts stood on its side of the court, turned to
// face it, on the ground under its footprint. Where the building has
// telescopes, they are noted for the manufactory in the court's frame,
// each at its place and never turned: a dome is round, and the
// catalogue's observatory does not mesh when it is turned (see the
// manufactory's test of it).
func (shared court) stood(drawing *drawer, index int, shape massShape,
	parts []placedElement, built stack) []placedElement {
	turn := 0.0
	if index > 0 {
		turn = []float64{0, math.Pi, -quarterTurn, quarterTurn}[index]
	}
	space := maths.NewSubspace(shared.corner(index, shape.depth),
		maths.Rotation(maths.YAxis, turn), 0)
	for _, mount := range built.mounts {
		mount.Place = maths.NewSubspace(space.Out(mount.Place.Origin),
			maths.Identity, mount.Place.Scale)
		*shared.mounts = append(*shared.mounts, mount)
	}
	standing, err := plant(within(parts, space), footprintOf(space,
		shape.width, shape.depth+drawing.fixed("core-proud")), shared.ground)
	drawing.fail(err)
	return standing
}

// corner is where the back left corner of a building stands on the
// ground: the building's depth out from its side of the court. The turns
// of the west and east buildings make their lengths run along z.
func (shared court) corner(index int, depth float64) maths.Vec {
	across, along := shared.width/half, shared.depth/half
	switch index {
	case 0:
		return maths.V(-across, 0, -along-depth)
	case 1:
		return maths.V(across, 0, along+depth)
	case 2:
		return maths.V(across+depth, 0, -along)
	}
	return maths.V(-across-depth, 0, along)
}

// stair is a building's stair core: a box standing out of the front by
// the culture's proud, one storey taller than the building, with the
// entrance in the lowest storey and a narrow window in each of the rest
// but the top, the stair head. It is drawn as a house of its own.
func (shared court) stair(drawing *drawer, at place, shape massShape,
	wall *thing.Material) []placedElement {
	width := drawing.fixed("core-width")
	proud := drawing.fixed("core-proud")
	sunk := drawing.fixed("core-sunk")
	narrow := massShape{width: width, depth: proud + sunk + shape.reach,
		storeys: shape.storeys + 1, door: doorNone}
	core := drawing.house(at, narrow, wall)
	core.Facade.Storeys = shared.landings(drawing, at, narrow)
	parts, err := core.parts()
	drawing.fail(err)
	place := maths.V(drawing.fixed("core-offset"), 0,
		shape.depth-sunk-shape.reach)
	standing := within(parts, standingAt(place))
	if drawing.homes {
		standing = append(standing, drawing.canopy(shape.depth+proud,
			drawing.fixed("core-offset"))...)
	}
	return standing
}

// landings are the openings of a stair core's front: the entrance, then
// a narrow window for each landing, and none in the stair head.
func (shared court) landings(drawing *drawer, at place,
	shape massShape) [][]facadeOpening {
	storeys := make([][]facadeOpening, shape.storeys)
	door := drawing.door(at)
	door.Left = (shape.width - door.Width) / half
	door.Extra.Portal = true
	storeys[0] = []facadeOpening{door}
	for index := 1; index < shape.storeys-1; index++ {
		here := at
		here.Storey = index
		slit := drawing.window(here)
		slit.Width = drawing.fixed("landing-window-width")
		slit.Left = (shape.width - slit.Width) / half
		slit.Extra = extra{Under: guardUnder}
		storeys[index] = []facadeOpening{slit}
	}
	return storeys
}
