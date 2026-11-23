package cities

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
)

// A habit is how a people lays out what it builds. It is a culture's way
// of arranging, not a random distribution: a motorhome park is rows of
// pitches with a lane between each pair of rows, every vehicle standing
// along its lane with its door to the same side; gers are scattered
// about the steppe, and each turns its door to the south.
//
// The same number of the same thing can be laid out as a park or as a camp, and
// the two look nothing alike.
//
// A habit is given the ground a stand may use, the facts of the item from
// that item's own data, and a seed, and hands back sites: a place and a
// way to face for each. It builds nothing. What stands on the sites is
// the stand's business, and a habit that is asked for the same ground,
// item and seed gives the same sites each time.
//
// Places are metres east and north on the map, and a way to face is a
// compass bearing in degrees, clockwise from north: the way the front of
// the item looks.

// Ground is the region a stand may use: a wedge about a centre, between
// two bearings and between a nearest and a farthest distance, as a
// stand's within says it.
type Ground struct {
	// East and North are the centre of the region.
	East, North float64
	// From and To are the bearings the wedge runs between, clockwise
	// from north, and round past north if To is the smaller. Equal
	// bearings, or a turn of a whole circle, make the whole circle.
	From, To float64
	// Near and Far are the nearest and the farthest the region reaches
	// from its centre, in metres.
	Near, Far float64
	// Box is a rectangle of the map in place of the wedge, for the
	// habits that fill one; the grid is one.
	Box *Quarter
	// Height is the height of the ground at a place on the map, for the
	// habits that follow it; the contour is one. Beyond the ground it
	// gives the height of the nearest ground there is.
	Height func(east, north float64) float64
	// Climb is how many rows of terraces climb a slope and how they
	// stand, for the contour.
	Climb *Climb
	// Frontage is the line of a channel a frontage stands along.
	Frontage *Frontage
}

// OnFrontage is the ground of a frontage: the line its sites stand along
// and nothing else.
func OnFrontage(along Frontage) Ground {
	return Ground{Frontage: &along}
}

// pitch is the ground one item takes in a layout in lanes, and which way
// the lanes run. All are in the item's data.
type pitch struct {
	// Width is how wide a pitch is across its lane, in metres.
	Width float64
	// Depth is how long a pitch is along its lane, in metres.
	Depth float64
	// Lane is how wide the lane between two rows is, in metres.
	Lane float64
	// Heading is the bearing the lanes run along, and so the way each
	// item in them faces.
	Heading float64
}

// habitFacing is the way a people prefer an item to face, and how far they
// stray from it, as the item's data says.
type habitFacing struct {
	// Heading is the preferred bearing of the item's front.
	Heading float64
	// Spread is how far either side of it an item may face, in degrees.
	Spread float64
}

// Item is what a habit needs to know of what it lays out, from the
// item's own data. A fact the data does not give is not had.
type Item struct {
	// Kind is the manufactory's kind of the item, for messages.
	Kind string
	// Pitch is the ground one takes, if Pitched.
	Pitch   pitch
	Pitched bool
	// Block is the room it takes in a grid of blocks, if Blocked.
	Block   block
	Blocked bool
	// Facing is the way it is preferred to face, if Faced.
	Facing habitFacing
	Faced  bool
	// Run is the room the item numbered from nought takes, for the habits
	// that lay items end to end; a row of terraces is one. It is nothing
	// for an item that is not drawn in a row.
	Run func(index int) (Run, error)
	// Spacing is the room one of the item takes along a line, in metres,
	// if Spaced: a tree on a quay, a boat along a wall.
	Spacing float64
	Spaced  bool
	// Turn is how far clockwise of the line's direction the item's front
	// looks, in degrees, if Turned. An item that is not Turned is left
	// to face as the one who stands it chooses.
	Turn   float64
	Turned bool
}

// Site is one place for one item, and the way it faces.
type Site struct {
	// East and North are where it stands.
	East, North float64
	// Heading is the bearing its front looks along. It is only meant
	// where Faced; a site that is not Faced leaves the facing to the
	// one who stands the item.
	Heading float64
	Faced   bool
}

// Laid is what a habit hands back: the sites, in the order it lays them,
// and a line saying what it did, for the one who runs it to read.
type Laid struct {
	Sites []Site
	Said  string
}

// Habit lays out sites for an item on some ground. Most is how many the
// stand asks for, nought for as many as fit.
type Habit interface {
	Sites(ground Ground, item Item, seed Seed, most int) (Laid, error)
}

// The names of the habits, as a stand's habit key says them.
const (
	// habitLattice is the habit of rows of pitches along lanes.
	habitLattice = "lattice"
	// HabitGrid is the habit of rows of houses fronting the streets
	// round blocks.
	HabitGrid = "grid"
	// habitScatter is the habit of places spread evenly over the ground.
	habitScatter = "scatter"
	// HabitContour is the habit of rows of terraces climbing a slope
	// along its contours.
	HabitContour = "contour"
	// HabitFrontage is the habit of items standing along the line of a
	// channel: terraces end to end with their fronts to the water, or
	// trees and boats at a spacing.
	HabitFrontage = "frontage"
)

// HabitNames are the names a stand may give its habit.
func HabitNames() []string {
	return []string{habitLattice, HabitGrid, habitScatter, HabitContour,
		HabitFrontage}
}

// HabitNamed is the habit of a name, or why there is none.
func HabitNamed(name string) (Habit, error) {
	switch name {
	case habitLattice:
		return lattice{}, nil
	case HabitGrid:
		return grid{}, nil
	case habitScatter:
		return scatter{}, nil
	case HabitContour:
		return contour{}, nil
	case HabitFrontage:
		return frontage{}, nil
	}
	return nil, fmt.Errorf("cities: no habit named %q, only %v", name,
		HabitNames())
}

// fullDegrees and quarterDegrees are a whole turn and a quarter of one,
// in degrees, which is how bearings are said.
const (
	fullDegrees    = 360
	quarterDegrees = 90
)

// The names of the facts an item gives, after its kind and a hyphen.
const (
	pitchWidthName   = "-pitch-width"
	pitchDepthName   = "-pitch-depth"
	laneWidthName    = "-lane-width"
	laneHeadingName  = "-lane-heading"
	facingHeadName   = "-facing-heading"
	facingSpreadName = "-facing-spread"
	blockLengthName  = "-block-length"
	blockDepthName   = "-block-depth"
	streetWidthName  = "-street-width"
	rowLengthName    = "-row-length"
	rowDepthName     = "-row-depth"
	gridTurnName     = "-grid-turn"

	frontageSpacingName = "-frontage-spacing"
	frontageTurnName    = "-frontage-turn"
)

// ItemOf is what the data says of a kind's footprint and facing. A kind
// that gives no pitch is not Pitched, and one that gives no facing is not
// Faced; one that gives only part of either is refused.
func ItemOf(kind string) (Item, error) {
	reader := &choiceReader{}
	item := Item{Kind: kind}
	if reader.has(kind + pitchWidthName) {
		item.Pitched = true
		item.Pitch = pitch{Width: reader.of(kind + pitchWidthName),
			Depth:   reader.of(kind + pitchDepthName),
			Lane:    reader.of(kind + laneWidthName),
			Heading: reader.of(kind + laneHeadingName)}
	}
	if reader.has(kind + blockLengthName) {
		item.Blocked = true
		item.Block = block{Length: reader.of(kind + blockLengthName),
			Depth:     reader.of(kind + blockDepthName),
			Street:    reader.of(kind + streetWidthName),
			RowLength: reader.of(kind + rowLengthName),
			RowDepth:  reader.of(kind + rowDepthName),
			Turn:      reader.of(kind + gridTurnName)}
	}
	if reader.has(kind + facingHeadName) {
		item.Faced = true
		item.Facing = habitFacing{Heading: reader.of(kind + facingHeadName),
			Spread: reader.of(kind + facingSpreadName)}
	}
	if reader.has(kind + frontageSpacingName) {
		item.Spaced = true
		item.Spacing = reader.of(kind + frontageSpacingName)
	}
	if reader.has(kind + frontageTurnName) {
		item.Turned = true
		item.Turn = reader.of(kind + frontageTurnName)
	}
	if reader.trouble != nil {
		return Item{}, reader.trouble
	}
	return item, nil
}

// has is whether the data names a choice.
func (reader *choiceReader) has(name string) bool {
	if choiceBookError != nil {
		reader.trouble = choiceBookError
		return false
	}
	_, known := choiceBook[name]
	return known
}

// bearingOf is the bearing of a direction given east and north, in
// degrees from zero up to a whole turn.
func bearingOf(east, north float64) float64 {
	degrees := math.Atan2(east, north) * halfTurn / math.Pi
	return math.Mod(degrees+fullDegrees, fullDegrees)
}

// toward is the unit direction, east and north, of a bearing.
func toward(bearing float64) (east, north float64) {
	radians := bearing * math.Pi / halfTurn
	return math.Sin(radians), math.Cos(radians)
}

// turn is how far the wedge turns, in degrees: a whole circle when its
// bearings are the same.
func (ground Ground) turn() float64 {
	turned := math.Mod(ground.To-ground.From+fullDegrees, fullDegrees)
	if turned == 0 {
		return fullDegrees
	}
	return turned
}

// contains is whether a place is in the region: no nearer than Near, no
// farther than Far, and inside the wedge's bearings.
func (ground Ground) contains(east, north float64) bool {
	across, up := east-ground.East, north-ground.North
	out := math.Hypot(across, up)
	if out < ground.Near || out > ground.Far {
		return false
	}
	if ground.turn() >= fullDegrees {
		return true
	}
	round := math.Mod(bearingOf(across, up)-ground.From+fullDegrees,
		fullDegrees)
	return round <= ground.turn()
}

// lattice lays pitches in rows. A lane runs between every pair of rows,
// and every item stands along its lane, facing the same way, so that all
// their doors are on the same side. It lays as many pitches as fit
// wholly inside the ground, and if the stand asks for fewer takes that
// many of them, by seed, and if for more says that not so many fit.
type lattice struct{}

// Sites is the pitches that fit in the ground, one site at the middle of
// each, or why the item cannot be laid in lanes.
func (lattice) Sites(ground Ground, item Item, seed Seed,
	most int) (Laid, error) {
	if !item.Pitched {
		return Laid{}, fmt.Errorf("cities: a lattice lays out pitches, and "+
			"the data gives %s none", item.Kind)
	}
	pitch := item.Pitch
	if !(pitch.Width > 0 && pitch.Depth > 0 && pitch.Lane >= 0) {
		return Laid{}, fmt.Errorf("cities: a pitch needs a positive width "+
			"and depth and a lane that is not negative, not %+v", pitch)
	}
	fits := pitches(ground, pitch)
	chosen := taken(fits, most, seed)
	said := fmt.Sprintf("a lattice laid %d pitches of %g by %g m in lanes "+
		"%g m wide, %d fit", len(chosen), pitch.Width, pitch.Depth,
		pitch.Lane, len(fits))
	if most > len(fits) {
		said += fmt.Sprintf(", fewer than the %d asked for", most)
	}
	return Laid{Sites: chosen, Said: said}, nil
}

// pitches is every pitch of the lattice that lies wholly in the ground,
// row by row. The lattice is pinned to the ground's centre: one pitch is
// centred there.
func pitches(ground Ground, pitch pitch) []Site {
	laneEast, laneNorth := toward(pitch.Heading)
	rowEast, rowNorth := toward(pitch.Heading + quarterDegrees)
	rowStep := pitch.Width + pitch.Lane
	rows := int(math.Ceil(ground.Far / rowStep))
	slots := int(math.Ceil(ground.Far / pitch.Depth))
	var sites []Site
	for row := -rows; row <= rows; row++ {
		for slot := -slots; slot <= slots; slot++ {
			along, across := float64(slot)*pitch.Depth, float64(row)*rowStep
			site := Site{Heading: pitch.Heading, Faced: true,
				East:  ground.East + along*laneEast + across*rowEast,
				North: ground.North + along*laneNorth + across*rowNorth}
			if holds(ground, site, pitch) {
				sites = append(sites, site)
			}
		}
	}
	return sites
}

// holds is whether the ground holds the whole of a pitch at a site: its
// four corners and the middles of its four sides.
func holds(ground Ground, site Site, pitch pitch) bool {
	laneEast, laneNorth := toward(pitch.Heading)
	rowEast, rowNorth := toward(pitch.Heading + quarterDegrees)
	for _, along := range []float64{-1, 0, 1} {
		for _, across := range []float64{-1, 0, 1} {
			deep, wide := along*pitch.Depth/2, across*pitch.Width/2
			if !ground.contains(site.East+deep*laneEast+wide*rowEast,
				site.North+deep*laneNorth+wide*rowNorth) {
				return false
			}
		}
	}
	return true
}

// taken is at most that many of the sites, chosen by the seed and kept in
// the order they were laid; all of them if most is nought or more than
// there are.
func taken(sites []Site, most int, seed Seed) []Site {
	if most <= 0 || most >= len(sites) {
		return sites
	}
	picks := rand.New(rand.NewPCG(uint64(seed), 1)).Perm(len(sites))[:most]
	slices.Sort(picks)
	chosen := make([]Site, most)
	for index, pick := range picks {
		chosen[index] = sites[pick]
	}
	return chosen
}

// scatter lays places evenly over the ground by the seed, as a stand
// without a habit does: member n is placed from the stream of the seed and
// n alone, so its place does not depend on how many others are laid. The
// ground is a wedge about a centre, or a rectangle.
//
// Each is turned to face the way the item's people prefer, within the
// spread, if the data says one, by a draw from its own stream after its
// place. An item with no preferred facing faces as the stand would have it.
type scatter struct{}

// scattered is a place the seed's stream offers over the ground: in the
// rectangle, if the ground is one, and else in the wedge, the bearing
// drawn first and then how far out.
func (ground Ground) scattered(places *rand.Rand) (east, north float64) {
	if box := ground.Box; box != nil {
		return box.West + places.Float64()*(box.East-box.West),
			box.South + places.Float64()*(box.North-box.South)
	}
	bearing := ground.From + places.Float64()*ground.turn()
	out := math.Sqrt(ground.Near*ground.Near +
		places.Float64()*(ground.Far*ground.Far-ground.Near*ground.Near))
	across, up := toward(bearing)
	return ground.East + out*across, ground.North + out*up
}

// Sites is as many places as the stand asks for, which it must. Member n
// draws its place and then its facing from MemberStream of the seed and n.
func (scatter) Sites(ground Ground, item Item, seed Seed,
	most int) (Laid, error) {
	if most < 1 {
		return Laid{}, fmt.Errorf("cities: a scatter lays out as many as " +
			"it is asked for, and the stand gives no count")
	}
	sites := make([]Site, most)
	for index := range sites {
		member := MemberStream(uint64(seed), index)
		east, north := ground.scattered(member)
		sites[index] = Site{East: east, North: north}
		if item.Faced {
			sites[index].Faced = true
			sites[index].Heading = faced(item.Facing, member.Float64())
		}
	}
	return Laid{Sites: sites}, nil
}

// faced is a bearing within the spread of the preferred one, a share of
// the way across the spread from zero to one, kept from zero up to a
// whole turn.
func faced(facing habitFacing, share float64) float64 {
	heading := facing.Heading + (2*share-1)*facing.Spread
	return math.Mod(heading+fullDegrees, fullDegrees)
}
