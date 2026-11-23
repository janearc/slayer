package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The ferry: a long open-sided cabin on the deck, its roof on pillars over
// sills and headers, so that a person on the quay sees through its windows
// the rows of seats and the lit ceiling; a wheelhouse on the front of its
// roof where the seed gives one; railings round every deck, a gate in the
// starboard rail, lifebuoys, a funnel, a name on a board on each side of the
// cabin, and a number on each bow. Every proportion here is chosen.
const (
	boatWall        = 0.06 // chosen: metres a cabin's wall is thick
	boatPillar      = 0.12 // chosen: metres a cabin's pillar is wide, along the wall
	boatSash        = 0.62 // chosen: how high up a window its drop-light rail runs, as a share of its height
	boatAisle       = 0.7  // chosen: metres of aisle between the two sides' seats
	boatSeatTall    = 0.45 // chosen: metres a seat is tall
	boatSeatLong    = 0.45 // chosen: metres a seat is long
	boatSeatGap     = 0.04 // chosen: metres between two seats side by side
	boatSeatEnd     = 0.5  // chosen: metres from a cabin's end wall to its first row
	boatBackTall    = 0.4  // chosen: metres a seat's back stands over its seat
	boatBackThick   = 0.08 // chosen: metres a seat's back is thick
	boatStripWide   = 0.12 // chosen: metres a ceiling light's strip is wide
	boatStripDepth  = 0.03 // chosen: metres a ceiling light's strip is deep
	boatStrips      = 3    // chosen: strips of light along a cabin's ceiling
	boatRail        = 0.9  // chosen: metres a deck railing is tall
	boatRailSet     = 0.06 // chosen: metres a deck railing stands in from the deck's edge
	boatGateWide    = 1.0  // chosen: metres the gangway gate is wide
	boatGateAfter   = 1.2  // chosen: metres aft of the cabin the gate begins
	boatGateGuard   = 0.3  // chosen: metres the gate stays from the stern's railing
	boatBuoyFore    = 0.9  // chosen: metres forward of the cabin the first lifebuoy hangs
	boatBuoyGap     = 1.1  // chosen: metres between lifebuoys along a rail
	boatBuoyUp      = 0.75 // chosen: metres over the deck a lifebuoy's middle hangs
	boatBoardIn     = 0.04 // chosen: metres a name board stands proud of the wall
	boatBoardLong   = 0.5  // chosen: how much of the cabin's length a name board takes
	boatBoardLetter = 0.6  // chosen: how much of the board's height its capitals take
	boatFerryNumber = 0.35 // chosen: metres tall a ferry's bow number is
)

// ferry builds the ferry's fittings and decides where its lights go:
// the long cabin, a wheelhouse over its front where the seed gives one, a
// funnel aft of that, a mast on the top, railings and the gear on them.
func (craft *boat) ferry() {
	hull, design := craft.hull, craft.design
	long := hull.length * design.pick("house-length", craft.from)
	centre := hull.z(design.pick("house-position", craft.from))
	wide := math.Min(hull.beam*design.pick("house-width", craft.from),
		2*boatHouseReach*hull.room(centre-long*spaceHalf, centre+long*spaceHalf))
	high := design.pick("house-height", craft.from)
	base, roof := craft.cabin(centre, long, wide, high)
	top, front, across := craft.upper(centre, long, wide, base, roof)
	craft.funnel(0, centre+long*boatFunnelAft, roof)
	craft.ferryMast(centre-long*boatUpperFore, top)
	craft.ferryLights(front, across, top)
	craft.cabinLamps(centre, long, wide, base, roof)
	craft.deckwork(centre, long)
	craft.nameBoards(centre, long, wide, base, high)
	craft.tyres()
	craft.bowNumbers(boatFerryNumber)
	craft.sternName(boatNameTall)
}

// upper is the wheelhouse over the front of the cabin's roof, where the seed
// gives one, with the gear of the roof on top of it, or else the gear on the
// cabin's own roof. It gives the top of all, the front of the top storey and
// its width.
func (craft *boat) upper(centre, long, wide, base, roof float64) (top, front,
	across float64) {
	if !craft.design.chance("upper-deck", craft.from) {
		craft.roofGear(centre, long, wide, base, roof)
		return roof, centre - long*spaceHalf, wide
	}
	middle, longer := centre-long*boatUpperFore, long*boatUpperLong
	across = wide * boatUpperWide
	top = craft.storey(middle, longer, across, roof, boatUpperHigh*(roof-base))
	if craft.detailed(boatClumps) {
		craft.roofRailing(centre, long, wide, roof)
	}
	craft.roofGear(middle, longer, across, roof, top)
	return top, middle - longer*spaceHalf, across
}

// The ferry's upper storey: how far forward of the cabin's middle it
// stands, how long and how wide it is as shares of the cabin's, how tall
// as a share of the cabin's height; and how far aft of the cabin's
// middle its funnel stands as a share of the cabin's length. All chosen.
const (
	boatUpperFore = 0.2
	boatUpperLong = 0.4
	boatUpperWide = 0.72
	boatUpperHigh = 0.85
	boatFunnelAft = 0.22
)

// cabin is the open-sided cabin: its roof, its two end walls and its two
// sides of sills, headers and pillars with the windows open between them,
// and inside, a lit ceiling and rows of seats. It gives the height of its
// floor and of the top of its roof. A silhouette has a solid house.
func (craft *boat) cabin(centre, long, wide, high float64) (base,
	roof float64) {
	hull := craft.hull
	if !craft.detailed(boatClumps) {
		return craft.house(centre, long, wide, high)
	}
	front, back := centre-long*spaceHalf, centre+long*spaceHalf
	base = math.Min(hull.deckAt(front), hull.deckAt(back)) - boatDeckLift
	craft.block(0, base+high, centre, wide+boatEaves, boatRoofThick,
		long+boatEaves, boatSteel)
	craft.ends(centre, long, wide, base, high)
	craft.sides(centre, long, wide, base, high)
	craft.lining(centre, long, wide, base, high)
	return base, base + high + boatRoofThick
}

// ends are a cabin's front and back walls: the front with its three
// framed panes, the back with its door.
func (craft *boat) ends(centre, long, wide, base, high float64) {
	front, back := centre-long*spaceHalf, centre+long*spaceHalf
	craft.block(0, base, front, wide, high, boatWall, boatHouse)
	craft.block(0, base, back, wide, high, boatWall, boatHouse)
	level, tall := base+boatWindowLow*high, boatWindowTall*high
	for index := range boatFrontPanes {
		across := (float64(index) - spaceOne) * wide * boatWindowRun /
			boatFrontPanes
		craft.pane(across, level, front, false, wide*boatWindowRun/
			boatFrontPanes-3*boatFrameWide, tall, craft.glowing(index))
	}
	craft.door(wide, back, base)
}

// sides are a cabin's long walls: a sill under the windows, a header over
// them, a pillar at each pitch between, and, in each opening, no glass
// but a drop-light rail, so that the inside shows.
func (craft *boat) sides(centre, long, wide, base, high float64) {
	level, tall := base+boatWindowLow*high, boatWindowTall*high
	count := max(int(long*boatWindowRun/boatPanePitch), 2)
	for index := range 2 {
		across := boatSide(index) * wide * spaceHalf
		craft.block(across, base, centre, boatWall, level-base, long, boatHouse)
		craft.block(across, level+tall, centre, boatWall,
			base+high-level-tall, long, boatHouse)
		for pillar := range count + spaceOne {
			along := centre - long*spaceHalf + long*float64(pillar)/float64(count)
			craft.block(across, level, along, boatWall, tall, boatPillar,
				boatHouse)
		}
		if craft.detailed(boatQuick) {
			craft.block(across, level+tall*boatSash, centre, boatFrameWide,
				boatFrameWide, long, boatSteel)
		}
	}
}

// lining is what is inside a cabin: strips of light along its ceiling,
// and, from a modest detail, its seats.
func (craft *boat) lining(centre, long, wide, base, high float64) {
	inner := wide - 2*boatWall
	for strip := range boatStrips {
		across := (float64(strip) - spaceOne) * inner / boatStrips
		craft.block(across, base+high-boatStripDepth, centre, boatStripWide,
			boatStripDepth, long-2*boatWall, boatWindow)
	}
	if craft.detailed(boatQuick) {
		craft.seats(centre, long, inner, base)
	}
}

// seats are the rows of seats, at the design's pitch from the front of the
// cabin to its back, two seats on each side of the aisle, each a cushion
// with a back behind it.
func (craft *boat) seats(centre, long, inner, base float64) {
	pitch := craft.design.pick("seat-pitch", craft.from)
	rows := int((long - 2*boatSeatEnd) / pitch)
	group := (inner - boatAisle) / 2
	seat := (group - boatSeatGap) / 2
	for row := range rows {
		along := centre - long*spaceHalf + boatSeatEnd + (float64(row)+
			spaceHalf)*pitch
		for index := range 4 {
			across := boatSide(index) * ((boatAisle+group)*spaceHalf +
				boatSide(index/2)*(seat+boatSeatGap)*spaceHalf)
			craft.block(across, base, along, seat, boatSeatTall, boatSeatLong,
				boatSeat)
			craft.block(across, base+boatSeatTall, along+boatSeatLong*spaceHalf,
				seat, boatBackTall, boatBackThick, boatSeat)
		}
	}
}

// ferryMast is the mast on the ferry's top roof, its top the top light.
func (craft *boat) ferryMast(along, roof float64) {
	craft.mounts.top = maths.V(0, roof+craft.design.pick("mast-height",
		craft.from), along)
	craft.rod(maths.V(0, roof-boatDeckLift, along), craft.mounts.top,
		boatMastRadius, boatSteel)
	craft.mounts.power = true
}

// ferryLights decides where the side lights and the stern lamp stand:
// the side lights at the front corners of the top roof, the stern lamp
// on a post at the transom.
func (craft *boat) ferryLights(front, across, roof float64) {
	reach := across*spaceHalf + boatEaves*spaceHalf + boatSideLamp
	craft.mounts.port = maths.V(-reach, roof+boatSideLamp, front)
	craft.mounts.starboard = maths.V(reach, roof+boatSideLamp, front)
	craft.sternPost()
}

// cabinLamps are the lamps under the cabin's eaves, which light the
// deck outside its windows: one at each end of each side.
func (craft *boat) cabinLamps(centre, long, wide, base, roof float64) {
	for index := range boatCabinLamps {
		turn := boatSide(index)
		along := centre + boatSide(index/2)*long*boatCabinAt
		craft.mounts.working = append(craft.mounts.working, boatWorking{
			lamp: "cabin", lens: boatWarmLens,
			at: maths.V(turn*(wide*spaceHalf+boatCabinOut), roof-
				(roof-base)*boatCabinDown, along),
			toward: maths.V(turn*boatFloodOut, -spaceOne, 0).Unit()})
	}
}

// The ferry's cabin lamps: how many, how far along the cabin as a share
// of its length, how far off its wall, and how far below its roof as a
// share of its height. All chosen.
const (
	boatCabinLamps = 4
	boatCabinAt    = 0.3
	boatCabinOut   = 0.3
	boatCabinDown  = 0.1
)

// deckwork is what is on the ferry's decks outside its cabin: the
// railings round them, the gate in the starboard rail, and the lifebuoys.
func (craft *boat) deckwork(centre, long float64) {
	if !craft.detailed(boatClumps) {
		return
	}
	hull := craft.hull
	pitch := craft.design.pick("stanchion-pitch", craft.from)
	fore, aft := hull.length*(boatBowEnd-spaceHalf), hull.length*(boatSternEnd-
		spaceHalf)
	gate := math.Min(centre+long*spaceHalf+boatGateAfter, aft-boatGateWide-
		boatGateGuard)
	craft.rails(craft.edge(spaceOne, fore, gate, pitch), boatRail)
	craft.rails(craft.edge(spaceOne, gate+boatGateWide, aft, pitch), boatRail)
	craft.rails(craft.edge(-spaceOne, fore, aft, pitch), boatRail)
	craft.rails(craft.transom(aft, pitch), boatRail)
	craft.gate(gate)
	craft.lifebuoys(centre - long*spaceHalf)
}

// edge is the places along a side of the deck a railing's posts stand at,
// at about a pitch, from a place along the boat to another, just inside the
// deck's edge: the side is turn, one for starboard and minus one for port.
func (craft *boat) edge(turn, from, to, pitch float64) []maths.Vec {
	hull := craft.hull
	count := max(int((to-from)/pitch), spaceOne)
	points := make([]maths.Vec, 0, count+spaceOne)
	for step := range count + spaceOne {
		along := from + (to-from)*float64(step)/float64(count)
		at := along/hull.length + spaceHalf
		points = append(points, maths.V(turn*(hull.halfAt(along)-boatRailSet),
			hull.sheer(at), along))
	}
	return points
}

// transom is the places a railing's posts stand at across the stern, from
// port to starboard.
func (craft *boat) transom(along, pitch float64) []maths.Vec {
	hull := craft.hull
	at := along/hull.length + spaceHalf
	half := hull.halfAt(along) - boatRailSet
	count := max(int(2*half/pitch), spaceOne)
	points := make([]maths.Vec, 0, count+spaceOne)
	for step := range count + spaceOne {
		points = append(points, maths.V(-half+2*half*float64(step)/
			float64(count), hull.sheer(at), along))
	}
	return points
}

// gate is the gangway gate in the starboard rail, from a place along the
// boat for a gate's width: two heavier posts, a top rail and a middle rail,
// and a diagonal brace across them.
func (craft *boat) gate(along float64) {
	hull := craft.hull
	at := along/hull.length + spaceHalf
	across, level := hull.halfAt(along)-boatRailSet, hull.sheer(at)
	for index := range 2 {
		post := along + float64(index)*boatGateWide
		craft.rod(maths.V(across, level-boatDeckLift, post), maths.V(across,
			level+boatRail, post), boatRodThin*2, boatSteel)
	}
	for _, share := range []float64{spaceOne, spaceHalf} {
		craft.rod(maths.V(across, level+boatRail*share, along), maths.V(across,
			level+boatRail*share, along+boatGateWide), boatRodThin, boatSteel)
	}
	craft.rod(maths.V(across, level, along), maths.V(across, level+boatRail,
		along+boatGateWide), boatRodThin*spaceHalf, boatSteel)
}

// lifebuoys hang on the rails forward of the cabin, one side then the other,
// as many as the design says, each a pitch further forward.
func (craft *boat) lifebuoys(front float64) {
	hull := craft.hull
	for index := range craft.design.count("lifebuoys", craft.from) {
		along := front - boatBuoyFore - boatBuoyGap*float64(index/2)
		at := along/hull.length + spaceHalf
		craft.lifebuoy(boatSide(index)*(hull.halfAt(along)-boatRailSet),
			hull.sheer(at)+boatBuoyUp, along)
	}
}

// nameBoards are the boards on each side of the cabin, over its windows,
// with the ferry's name painted on them.
func (craft *boat) nameBoards(centre, long, wide, base, high float64) {
	if !craft.detailed(boatQuick) {
		return
	}
	face, words := craft.name()
	level := base + (boatWindowLow+boatWindowTall)*high
	tall := base + high - level
	wall := wide*spaceHalf + boatWall*spaceHalf
	for index, facing := range []maths.Orientation{boatStarboardFace,
		boatPortFace} {
		turn := boatSide(index)
		craft.block(turn*(wall+boatBoardIn*spaceHalf), level, centre,
			boatBoardIn, tall, long*boatBoardLong, boatSteel)
		craft.paint(face, words, long*boatBoardLong, tall*boatBoardLetter,
			maths.V(turn*(wall+boatBoardIn), level+tall*spaceHalf, centre),
			facing)
	}
}
