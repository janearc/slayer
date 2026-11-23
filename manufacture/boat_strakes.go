package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The planks and plates of a hull, its bulwark and what stands on it:
// every strake a thin loft laid along the side, proud of it; the wall of
// the bulwark and its cap; a stanchion at each pitch; freeing ports; and
// the streaks of rust that run from them. Each is a real part. Every
// proportion is chosen.
const (
	boatPlankThick  = 0.014 // chosen: metres a wooden plank stands proud
	boatPlateThick  = 0.007 // chosen: metres a steel plate stands proud
	boatStrakeGap   = 0.006 // chosen: metres between two strakes
	boatWallThick   = 0.04  // chosen: metres a bulwark's wall is thick
	boatCapThick    = 0.05  // chosen: metres a cap rail is deep
	boatCapOver     = 0.02  // chosen: metres a cap rail stands out past its wall, each side
	boatRubDepth    = 0.12  // chosen: metres a rubbing strake is tall
	boatRubDrop     = 0.3   // chosen: metres its top is under the deck's edge
	boatRubThick    = 0.05  // chosen: metres it stands proud
	boatBootTall    = 0.3   // chosen: metres a boot topping is tall, from just under the waterline
	boatBootThick   = 0.01  // chosen: metres a boot topping stands proud
	boatStanchion   = 0.06  // chosen: metres across a stanchion, across the boat
	boatPortLong    = 0.45  // chosen: metres a freeing port is long
	boatPortTall    = 0.16  // chosen: metres a freeing port is tall
	boatStreakLong  = 0.7   // chosen: metres a rust streak runs down from a port
	boatStreakWide  = 0.05  // chosen: metres a rust streak is wide
	boatStreakThick = 0.012 // chosen: metres a rust streak stands proud
)

// boatSunk is how far a strip's inner face is sunk into the hull, in
// metres: chosen, less than any strip's own thickness.
const boatSunk = 0.003

// skin is how far out the hull's side stands to starboard at a station
// and a height over the waterline: flaring up to the deck's edge and
// straight above it.
func (hull boatHull) skin(at, up float64) float64 {
	rise := math.Min(math.Max(up/hull.sheer(at), spaceZero), spaceOne)
	flare := spaceOne + (hull.flaring(at)-spaceOne)*
		math.Pow(rise, boatFlareCurve)
	return hull.half(at) * flare
}

// strip is a thin band along a side of the hull, between two heights
// that are each a function of the station, set in or out of the skin by
// inset and standing thick out from there: the side is turn, one for
// starboard and minus one for port. Its inner face is sunk a little into
// the hull, so that no corner of it is a corner of the hull's: two solids
// that shared an edge would not mesh.
func (hull boatHull) strip(turn float64, low, high func(float64) float64,
	inset, thick float64) boatLoft {
	inset, thick = inset-boatSunk, thick+boatSunk
	return hull.lofted(func(at float64) []maths.Vec {
		down, up := low(at), high(at)
		in := turn * (hull.skin(at, down) + inset)
		top := turn * (hull.skin(at, up) + inset)
		return []maths.Vec{hull.point(at, in, down),
			hull.point(at, in+turn*thick, down),
			hull.point(at, top+turn*thick, up), hull.point(at, top, up)}
	})
}

// level is a height that is a fixed distance over the deck's edge.
func (hull boatHull) level(over float64) func(float64) float64 {
	return func(at float64) float64 { return hull.sheer(at) + over }
}

// height is a share of the hull's side, from the waterline to the top of
// its bulwark, less a gap if it is the top of a strake.
func (hull boatHull) height(share, gap float64) func(float64) float64 {
	return func(at float64) float64 {
		return (hull.sheer(at)+hull.rail)*share - gap
	}
}

// both adds a strip to each side of the hull.
func (craft *boat) both(low, high func(float64) float64, inset, thick float64,
	material slot) {
	for index := range 2 {
		craft.assembly.add(craft.hull.strip(boatSide(index), low, high,
			inset, thick), maths.Local(), material)
	}
}

// cladding is the planks of a wooden hull or the plates of a steel one:
// as many strakes as the design says from the waterline up the side and
// the bulwark, a pair to the strake, one a side. A silhouette has none.
func (craft *boat) cladding() {
	if !craft.detailed(boatClumps) {
		return
	}
	count := craft.design.count("strakes", craft.from)
	thick := boatPlateThick
	if craft.row.design == boatSampan {
		thick = boatPlankThick
	}
	for strake := range count {
		low := float64(strake) / float64(count)
		high := float64(strake+spaceOne) / float64(count)
		craft.both(craft.hull.height(low, 0), craft.hull.height(high,
			boatStrakeGap), 0, thick, boatPaint)
	}
}

// waterline is the boot topping, a band at the waterline, and, on a
// steel boat, a rubbing strake near the deck's edge.
func (craft *boat) waterline() {
	hull := craft.hull
	if !craft.detailed(boatClumps) {
		return
	}
	craft.both(func(float64) float64 { return -boatBootTall / 6 },
		func(float64) float64 { return boatBootTall * 5 / 6 }, 0,
		boatBootThick, boatBoot)
	if craft.row.design != boatSampan {
		craft.both(hull.level(-boatRubDrop-boatRubDepth),
			hull.level(-boatRubDrop), 0, boatRubThick, boatSteel)
	}
}

// hullWork is all that is done to the hull of any design before its own
// fittings: its planks or plates, the waterline, the bulwark with its
// stanchions, the cleats along it, and the rudder and propeller.
func (craft *boat) hullWork() {
	craft.cladding()
	craft.waterline()
	craft.bulwark()
	craft.stanchions()
	craft.cleats()
	craft.screw()
}

// bulwark is the wall along each side over the deck's edge and the cap
// rail on its top, where the design has a rail.
func (craft *boat) bulwark() {
	hull := craft.hull
	if hull.rail <= 0 {
		return
	}
	craft.both(hull.level(0), hull.level(hull.rail), -boatWallThick,
		boatWallThick-boatSunk, boatPaint)
	if !craft.detailed(boatClumps) {
		return
	}
	craft.both(hull.level(hull.rail), hull.level(hull.rail+boatCapThick),
		-boatWallThick-boatCapOver, boatWallThick+2*boatCapOver, boatDeck)
}

// stations is the places along the boat a fitting at a pitch stands, from
// the bow and the stern each a share of the length in: a pitch the detail
// stretches, so that a silhouette has fewer.
func (craft *boat) stations(pitch, fore, aft float64) []float64 {
	spread := [...]float64{boatSparse, boatThin, boatLess, spaceOne}
	step := pitch * spread[craft.rank]
	long := craft.hull.length * (aft - fore)
	count := max(int(long/step), spaceOne)
	places := make([]float64, 0, count+spaceOne)
	for index := range count + spaceOne {
		along := craft.hull.length * (fore - spaceHalf)
		places = append(places, along+float64(index)*long/float64(count))
	}
	return places
}

// How much a pitch stretches at each rank of detail: chosen.
const (
	boatSparse = 6.0
	boatThin   = 3.0
	boatLess   = 1.5
)

// stanchions stand along the outside of the bulwark, a flat bar at each
// pitch, and freeing ports cut low in it, each with the rust that runs
// down the hull from it.
func (craft *boat) stanchions() {
	hull := craft.hull
	pitch := craft.design.pick("stanchion-pitch", craft.from)
	if hull.rail <= 0 || pitch <= 0 || !craft.detailed(boatClumps) {
		return
	}
	for _, along := range craft.stations(pitch, boatBowEnd, boatSternEnd) {
		at := along/hull.length + spaceHalf
		for index := range 2 {
			turn := boatSide(index)
			craft.block(turn*(hull.skin(at, hull.sheer(at))+boatStanchion/2),
				hull.sheer(at), along, boatStanchion, hull.rail, boatStanchion,
				boatSteel)
		}
	}
	craft.ports()
}

// boatBowEnd and boatSternEnd are how far in from the bow and the stern,
// as shares of the length, a row of fittings along a side begins and ends.
// Chosen.
const (
	boatBowEnd   = 0.08
	boatSternEnd = 0.95
)

// ports are the freeing ports, as many on a side as the seed gives, and
// the streaks of rust below them. They keep to the middle of the length,
// where the side is straight.
func (craft *boat) ports() {
	hull := craft.hull
	count := boatPorts + int(craft.from.draw(housing, boatDrawGear, 0)*boatPorts)
	if !craft.detailed(boatQuick) {
		count = spaceOne
	}
	for index := range count * 2 {
		turn := boatSide(index)
		along := hull.length * (boatPortFore + boatPortSpan*
			(float64(index/2)+spaceHalf)/float64(count))
		at := along/hull.length + spaceHalf
		out := turn * (hull.skin(at, hull.sheer(at)) + boatStreakThick)
		craft.block(out, hull.sheer(at)+boatDeckLift, along, boatStreakThick*2,
			boatPortTall, boatPortLong, boatGlass)
		craft.block(out, hull.sheer(at)-boatStreakLong, along,
			boatStreakThick, boatStreakLong, boatStreakWide, boatRust)
	}
}

// The freeing ports: the fewest on a side, where along the length the
// first stands and how much of it they spread over, as shares. Chosen.
const (
	boatPorts    = 2
	boatPortFore = -0.25
	boatPortSpan = 0.5
)
