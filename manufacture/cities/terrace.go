package cities

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// noTower is the tower index of a row that has no tower.
const noTower = -1

// rowPlan is what a row of bays is before any bay is drawn: how many,
// how wide, how deep, which are alike by handing, where the tower is,
// and the colour of each bay's wall. Colours are drawn together because
// each bay's depends on the last one's.
type rowPlan struct {
	bays         int
	width, depth float64
	yard, garage float64
	lead, pitch  float64
	handed       bool
	tower        int
	colours      []option
	cladding     option
}

// Terrace is a row of bays drawn from a culture by a seed: houses side
// by side, a unit repeated along the row with its handing and its ends,
// each bay with its own colour, storeys, roof and openings. The row's
// front faces along positive z and it runs along x from nought.
//
// A culture's rules are kept in the drawing. No two adjacent bays share
// a colour where the culture says so. A flat roof has no eaves. A tower
// stands only at an end of the row. A chimney stands on a party wall.
func Terrace(culture string, seed Seed) (Design, error) {
	return TerraceOn(culture, seed, nil)
}

// TerraceOn is a terrace on a slope: the same row, each bay stood on the
// ground under its own footprint, so that a row on a hillside steps down
// it one bay at a time. The ground is in the row's frame. With no ground
// it is the terrace on flat ground.
func TerraceOn(culture string, seed Seed, ground Terrain) (Design, error) {
	return terraceAtDetail(culture, seed, ground, "")
}

// terraceAtDetail is a terrace on a slope with its faces drawn at a detail
// level: at the hull only the wall of each; at the demo level each window
// of its lights, bars, sill and head, each shutter of its slats, each
// door of its panels and step, and a cornice. Any other level is the
// terrace as TerraceOn draws it.
func terraceAtDetail(culture string, seed Seed, ground Terrain,
	level Detail) (Design, error) {
	return TerraceFounded(culture, seed, ground, level, FoundHighest)
}

// TerraceFounded is TerraceAt with the floor of each bay founded as said:
// at the highest ground under it, or at the ground outside its front
// door, the highest along its front wall.
func TerraceFounded(culture string, seed Seed, ground Terrain,
	level Detail, founding Founding) (Design, error) {
	drawing, err := newDrawer(culture, seed)
	if err != nil {
		return Design{}, err
	}
	drawing.level = level
	plan := drawing.row()
	if drawing.trouble != nil {
		return Design{}, drawing.trouble
	}
	ordinary := func(index int) ([]placedElement, error) {
		return plan.planted(drawing, index, false, ground, founding)
	}
	tower := func(index int) ([]placedElement, error) {
		return plan.planted(drawing, index, true, ground, founding)
	}
	run := repeat{Count: plan.bays, Spacing: plan.pitch, Axis: maths.XAxis,
		Handed: plan.handed, Unit: ordinary}
	if plan.tower == 0 {
		run.First = tower
	} else if plan.tower > 0 {
		run.Last = tower
	}
	parts, err := run.parts()
	if err != nil {
		return Design{}, err
	}
	return Design{Name: "terrace " + culture, Elements: parts}, nil
}

// row is the plan of a row: everything that is drawn for the row as a
// whole, before any bay.
func (drawing *drawer) row() rowPlan {
	at := place{}
	plan := rowPlan{bays: drawing.count("bays", at),
		width: drawing.number("bay-width", at),
		depth: drawing.number("bay-depth", at),
		yard:  drawing.fixed("yard"), garage: drawing.fixed("garage-width"),
		handed: drawing.chance("handing", at), tower: noTower}
	drawing.coarse("bay-width", scopeRow)
	drawing.coarse("bay-depth", scopeRow)
	plan.lead = plan.yard + plan.garage
	plan.pitch = plan.lead + plan.width
	if drawing.chance("tower", at) {
		plan.tower = 0
		if !drawing.chance("tower-first", at) {
			plan.tower = max(plan.bays-1, 0)
		}
	}
	plan.colours = drawing.colours(plan.bays)
	plan.cladding = drawing.pick("cladding", drawing.options("cladding"), at)
	return plan
}

// coarse notes a refusal if the element of a name is drawn at a finer
// scope than the limit. A row's bays stand one pitch apart, so the width
// of a bay cannot be drawn anew for each.
func (drawing *drawer) coarse(name string, limit scope) {
	each, found := drawing.found(name, drawRange)
	if found && slices.Index(scopes, each.Scope) >
		slices.Index(scopes, limit) {
		drawing.trouble = fmt.Errorf("cities: %s is drawn at scope %s, and "+
			"must be no finer than %s", name, each.Scope, limit)
	}
}

// colours are the wall colours of a row's bays: the palette the
// district uses, and from it a colour for each bay. If the culture says
// neighbours differ, a bay is never the colour of the one before it.
func (drawing *drawer) colours(bays int) []option {
	district := place{}
	spread := drawing.count("palette-spread", district)
	palette := drawing.subset("palette", spread, district)
	differ := drawing.chance("distinct-neighbours", district)
	drawing.differed = differ
	colours := make([]option, bays)
	for index := range colours {
		var skip []string
		if differ && index > 0 {
			skip = []string{colours[index-1].Name}
		}
		colours[index] = drawing.pick("palette", palette, place{Bay: index},
			skip...)
	}
	return colours
}

// mirrored is whether handing turns a bay about: every other one.
func (plan rowPlan) mirrored(index int) bool {
	return plan.handed && index%2 == 1
}

// free is whether a bay's side wall, the one a side door is in, has air
// outside it: a bay set apart by a yard or a garage, or the last one of
// a row that is not handed. A handed row's bays pair on their side walls.
func (plan rowPlan) free(index int) bool {
	return !plan.handed && (plan.lead > 0 || index == plan.bays-1)
}

// partyWall is where along a bay's own space a chimney may stand on a
// wall it shares with a bay beside it, and whether it has one: the
// boundary between this bay and the one before it or after it, never the
// end of the row.
func (plan rowPlan) partyWall(index int) (float64, bool) {
	switch {
	case plan.lead == 0 && plan.mirrored(index):
		return 0, index+1 < plan.bays
	case plan.lead == 0:
		return 0, index > 0
	case plan.handed && index%2 == 0:
		return plan.pitch, index+1 < plan.bays
	}
	return 0, false
}

// shape is the mass a bay is before its looks: a tower is as wide and
// as deep as a bay's space and stands against the front, and a house
// stands after its yard and garage. A bay drawn without a front door
// has one in its side if the side is free, and in front if not.
func (plan rowPlan) shape(drawing *drawer, at place, tower bool) massShape {
	if tower {
		return massShape{width: plan.pitch, depth: plan.pitch,
			storeys: drawing.count("tower-storeys", at), tower: true}
	}
	shape := massShape{width: plan.width, depth: plan.depth,
		storeys: drawing.count("storeys", at), door: doorFront}
	if !drawing.chance("door-front", at) && plan.free(at.Bay) {
		shape.door = doorSide
	}
	return shape
}

// bay is one unit of the row: the house, or the tower, with its garage,
// chimney and side door.
func (plan rowPlan) bay(drawing *drawer, index int,
	tower bool) ([]placedElement, error) {
	at := place{Bay: index}
	drawing.era = drawing.eraAt(at)
	defer func() { drawing.era = "" }()
	shape := plan.shape(drawing, at, tower)
	wall := wall(plan.eraColour(drawing, at), plan.cladding)
	house := drawing.house(at, shape, wall)
	if tower {
		plan.crown(drawing, at, &house)
	}
	if drawing.trouble != nil {
		return nil, drawing.trouble
	}
	parts, err := house.deferred()
	if err != nil {
		return nil, err
	}
	left, back := plan.lead, 0.0
	if tower {
		left, back = 0, plan.depth-shape.depth
	}
	parts = within(parts, standingAt(maths.V(left, 0, back)))
	if !tower {
		parts = append(parts, plan.extras(drawing, at, shape, house)...)
	}
	return []placedElement{deferred(plan.bayNode(parts, house))}, drawing.trouble
}

// bayNode is a bay as a node: the box of a pitch of the row as deep as
// the row and as high as the house's ridge, holding the bay's parts.
func (plan rowPlan) bayNode(parts []placedElement, house house) *node {
	top := house.Facade.Height + house.Roof.Rise
	return &node{Bounds: maths.Box{Max: maths.V(plan.pitch, top,
		math.Max(plan.depth, house.Depth))},
		Draw: func(Detail) ([]placedElement, error) { return parts, nil }}
}

// planted is a bay stood on the ground under its footprint, which is a
// pitch wide and a bay deep, or a tower's depth for a tower, standing
// against the front. The ground is in the row's frame, so it is read a
// pitch further along for each bay. With no ground the bay is as drawn.
func (plan rowPlan) planted(drawing *drawer, index int, tower bool,
	ground Terrain, founding Founding) ([]placedElement, error) {
	parts, err := plan.bay(drawing, index, tower)
	if err != nil || ground == nil {
		return parts, err
	}
	back := 0.0
	if tower {
		back = plan.depth - plan.pitch
	}
	front := plan.depth
	if tower {
		front = back + plan.pitch
	}
	along := float64(index) * plan.pitch
	return plantFounded(parts, rect{0, back, plan.pitch, front},
		func(x, z float64) float64 { return ground(x+along, z) }, founding)
}

// Run is the room a terrace takes before any of it is drawn: the length
// of its row of bays, and how far it reaches from the line of its front
// to the back of its deepest part. It also says the colours of the walls
// of the first and the last bay, and whether the culture says that no two
// neighbours share a colour, so that a row made of several terraces can
// keep that rule where one ends and the next begins.
type Run struct {
	Length, Depth float64
	// First and Last are the names of the colours of the first and the
	// last bay of the terrace, along x from nought.
	First, Last string
	// Differ is whether the culture says no two neighbours share a colour.
	Differ bool
}

// TerraceRun is the room the terrace of a culture and a seed takes, and
// the colours of its ends. It draws the row's plan and none of its bays,
// and is the room the same seed's terrace has in TerraceAt. A tower stands
// against the front of the row and as deep as a bay is wide, so it may
// reach further back than the bays do.
func TerraceRun(culture string, seed Seed) (Run, error) {
	drawing, err := newDrawer(culture, seed)
	if err != nil {
		return Run{}, err
	}
	plan := drawing.row()
	if drawing.trouble != nil {
		return Run{}, drawing.trouble
	}
	depth := plan.depth
	if plan.tower != noTower {
		depth = math.Max(depth, plan.pitch)
	}
	run := Run{Length: float64(plan.bays) * plan.pitch, Depth: depth,
		Differ: drawing.differed}
	if len(plan.colours) > 0 {
		run.First = plan.colours[0].Name
		run.Last = plan.colours[len(plan.colours)-1].Name
	}
	return run, nil
}

// crown is a tower's roof: a hip drawn in almost to a point, as high as
// the culture says.
func (plan rowPlan) crown(drawing *drawer, at place, house *house) {
	roof := &house.Roof
	roof.Form, roof.Parapet = hip, 0
	roof.Eave = drawing.number("eave-depth", at)
	roof.Rise = drawing.number("tower-roof-rise", at)
	roof.Inset = drawing.number("tower-point", at) *
		(roof.Width/2 + roof.Eave)
}

// extras are what stands about a house and is not the house: its
// garage, a chimney on a party wall, a door in the side wall.
func (plan rowPlan) extras(drawing *drawer, at place, shape massShape,
	house house) []placedElement {
	var parts []placedElement
	if plan.garage > 0 && drawing.chance("garage", at) {
		parts = append(parts, plan.garaged(drawing, at, shape)...)
	}
	if along, found := plan.partyWall(at.Bay); found &&
		drawing.chance("chimney", at) {
		parts = append(parts, plan.chimney(drawing, house, along))
	}
	if shape.door == doorSide {
		parts = append(parts, plan.sideDoor(drawing, at, shape))
	}
	return parts
}

// garaged is a garage in the space before the house: one storey, with
// a door in its front the width of the culture's garage door.
func (plan rowPlan) garaged(drawing *drawer, at place,
	shape massShape) []placedElement {
	garage := massShape{width: plan.garage, depth: shape.depth, storeys: 1}
	wall := wall(plan.colours[at.Bay], plan.cladding)
	size := drawing.fixed("garage-door-width")
	door := facadeOpening{Left: (plan.garage - size) / 2, Width: size,
		Height: drawing.fixed("garage-door-height"), Garage: true,
		Pane: paint("door", drawing.pick("door-colour",
			drawing.options("door-colour"), at)),
		Frame: paint("frame", drawing.pick("frame-colour",
			drawing.options("frame-colour"), at))}
	house := drawing.house(at, garage, wall)
	house.Facade.Height = drawing.fixed("garage-height")
	house.Facade.Ground = 0
	house.Facade.Storeys = [][]facadeOpening{{door}}
	parts, err := house.parts()
	drawing.fail(err)
	return within(parts, standingAt(maths.V(plan.yard, 0, 0)))
}

// chimney is a box standing on the party wall at a place along the bay,
// up from the top of the wall to above the ridge by the culture's proud.
func (plan rowPlan) chimney(drawing *drawer, house house,
	along float64) placedElement {
	width, depth := drawing.fixed("chimney-width"),
		drawing.fixed("chimney-depth")
	height := house.Roof.Rise + drawing.fixed("chimney-proud")
	at := maths.V(along-width/2, house.Facade.Height,
		house.Depth/2-depth/2)
	return put("chimney", Box{width, height, depth}, house.Facade.Wall, at)
}

// sideDoor is a door stood on the side wall of a house, on the right of
// it, facing along positive x. It stands half its thickness out of the
// wall and is not cut into it.
func (plan rowPlan) sideDoor(drawing *drawer, at place,
	shape massShape) placedElement {
	width := drawing.fixed("door-width")
	thickness := drawing.fixed("glass-thickness")
	doors := drawing.options("door-colour")
	door := paint("door", drawing.pick("door-colour", doors, at))
	turn := maths.Rotation(maths.YAxis, quarterTurn)
	face := plan.lead + shape.width
	return placedElement{Element: solid{Name: "side door", Material: door,
		Form: Box{width, drawing.fixed("door-height"), thickness}},
		Place: maths.NewSubspace(maths.V(face-thickness/2, 0,
			shape.depth/2+width/2), turn, 0)}
}
