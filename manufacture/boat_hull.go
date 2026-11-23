package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The hull of a boat, lofted through stations along its length. Each
// station is a section across the boat: its half beam from the plan, its
// deck height from the sheer, its keel from the profile. The underbody
// runs from the keel to the waterline, the topsides from the waterline to
// the deck's edge, and a thin deck lies on them.

// The shape of the plan, the profile and the stem, all chosen: a boat is
// seen from outside and none of it is measured.
const (
	boatBowPower   = 2     // chosen: the bow's plan is an ellipse
	boatSternPower = 2.5   // chosen: the beam holds, then falls to the transom
	boatSternShare = 0.25  // chosen: how much of the bow's sheer rise the stern keeps
	boatKeelRun    = 0.35  // chosen: how much of the length the forefoot rises over
	boatSternKeel  = 0.25  // chosen: how much of its draught the keel loses at the transom
	boatKeelAft    = 0.8   // chosen: how far along the keel begins to rise to the transom
	boatRakeRun    = 0.2   // chosen: how much of the length the stem's lean dies out over
	boatFlareCurve = 1.5   // chosen: how fast a topside flares out as it rises
	boatSpringBow  = 3     // chosen: the power the bow's sheer rise falls away by
	boatDeckThick  = 0.06  // chosen: metres the deck's edge stands below the sheer
	boatDeckLift   = 0.02  // chosen: metres the deck's crown stands over the hull's
	boatDeckInset  = 0.985 // chosen: how much of the sheer's half beam the deck takes
)

// boatStations and boatSections are how finely a hull is cut by the
// rank of the detail, from the silhouette up: the stations along its
// length and the corners up each side of a section. Chosen, enough for a
// bow's curve to read as one at 2K.
var (
	boatStations = [...]int{8, 14, 22, 34}
	boatSections = [...]int{3, 4, 6, 9}
)

// boatHull is a hull's figures, drawn from a seed: its length and beam;
// its freeboard and draught amidships; how much its sheer rises at the
// bow, how much its topsides flare there, how wide its transom is as a
// share of its beam, how full its bilge is; how far its stem leans and
// how much its deck is crowned; how much of its draught is left at the
// forefoot; how much of its length is bow.
//
// And how many stations and
// section corners it is cut into.
type boatHull struct {
	length, beam, freeboard, draught, spring, flare float64
	transom, bilge, rake, camber, forefoot, bow     float64
	rail                                            float64
	stations, sides                                 int
}

// newBoatHull is the hull a seed draws from a design, cut as finely as
// the rank of the detail says.
func newBoatHull(design boatDesign, from seed, rank int) boatHull {
	length := design.pick("length", from)
	return boatHull{length: length,
		beam:      length / design.pick("slenderness", from),
		freeboard: design.pick("freeboard", from),
		draught:   design.pick("draught", from),
		spring:    design.pick("spring", from),
		flare:     design.pick("flare", from),
		transom:   design.pick("transom", from),
		bilge:     design.pick("bilge", from),
		rake:      design.pick("rake", from),
		camber:    design.pick("camber", from),
		forefoot:  design.pick("forefoot", from),
		bow:       design.pick("bow", from),
		rail:      design.pick("rail", from),
		stations:  boatStations[rank], sides: boatSections[rank]}
}

// z is how far along the boat a station stands, a share from the bow at
// nought to the transom at one: bow toward minus z, the middle at nought.
func (hull boatHull) z(at float64) float64 {
	return (at - spaceHalf) * hull.length
}

// plan is how wide the hull is at a station as a share of its beam: an
// ellipse over the bow, then the whole beam, falling to the transom's.
func (hull boatHull) plan(at float64) float64 {
	if at < hull.bow {
		ahead := spaceOne - at/hull.bow
		return math.Pow(spaceOne-math.Pow(ahead, boatBowPower),
			spaceOne/boatBowPower)
	}
	aft := (at - hull.bow) / (spaceOne - hull.bow)
	return spaceOne - (spaceOne-hull.transom)*math.Pow(aft, boatSternPower)
}

// half is half the hull's beam at the waterline of a station.
func (hull boatHull) half(at float64) float64 {
	return hull.beam * spaceHalf * hull.plan(at)
}

// flaring is how much wider than at the waterline the deck's edge is at
// a station: the flare, most at the bow and gone amidships.
func (hull boatHull) flaring(at float64) float64 {
	return spaceOne + hull.flare*math.Pow(spaceOne-at, 2)
}

// sheer is how high the deck's edge stands over the waterline at a
// station: the freeboard, and the rise at the bow and a little at the
// stern.
func (hull boatHull) sheer(at float64) float64 {
	rise := math.Pow(spaceOne-at, boatSpringBow) +
		boatSternShare*math.Pow(at, boatSpringBow)
	return hull.freeboard + hull.spring*rise
}

// deck is how high the middle of the deck stands at a station, over its
// edge by the crown.
func (hull boatHull) deck(at float64) float64 {
	return hull.sheer(at) + hull.camber
}

// keel is how deep the keel lies under the waterline at a station: the
// draught amidships, less toward the bow where the forefoot rises and a
// little at the transom.
func (hull boatHull) keel(at float64) float64 {
	share := float64(spaceOne)
	if at < boatKeelRun {
		share = hull.forefoot + (spaceOne-hull.forefoot)*
			maths.Smooth(0, boatKeelRun, at)
	}
	if at > boatKeelAft {
		share -= boatSternKeel * maths.Smooth(boatKeelAft, spaceOne, at)
	}
	return hull.draught * share
}

// lean is how far forward a point at a height stands of its station, so
// that the stem leans: nought at the keel's depth, the whole rake at the
// bow's top, and dying out aft of the bow.
func (hull boatHull) lean(at, up float64) float64 {
	run := math.Max(spaceZero, spaceOne-at/boatRakeRun)
	height := (up + hull.draught) / (hull.draught + hull.freeboard)
	return hull.rake * height * run * run
}

// point is a place of the hull: at a station, so far across and so high.
func (hull boatHull) point(at, across, up float64) maths.Vec {
	return maths.V(across, up, hull.z(at)-hull.lean(at, up))
}

// lowerAt is a place of the underbody's side at a station, a step of
// the way from the keel to the waterline: across to starboard, and up.
func (hull boatHull) lowerAt(at float64, step int) (across, up float64) {
	rise := float64(step) / float64(hull.sides)
	return hull.half(at) * (spaceOne - math.Pow(spaceOne-rise, hull.bilge)),
		-hull.keel(at) * (spaceOne - rise)
}

// lowerRing is the underbody's section at a station: the keel, up the
// starboard side to the waterline, and down the port side again.
func (hull boatHull) lowerRing(at float64) []maths.Vec {
	ring := make([]maths.Vec, 0, 2*hull.sides+spaceOne)
	for step := range hull.sides + spaceOne {
		across, up := hull.lowerAt(at, step)
		ring = append(ring, hull.point(at, across, up))
	}
	for step := hull.sides; step >= spaceOne; step-- {
		across, up := hull.lowerAt(at, step)
		ring = append(ring, hull.point(at, -across, up))
	}
	return ring
}

// upperAt is a place of the topsides' side at a station, a step of the
// way from the waterline to the deck's edge.
func (hull boatHull) upperAt(at float64, step int) (across, up float64) {
	rise := float64(step) / float64(hull.sides)
	flare := spaceOne + (hull.flaring(at)-spaceOne)*
		math.Pow(rise, boatFlareCurve)
	return hull.half(at) * flare, hull.sheer(at) * rise
}

// upperRing is the topsides' section at a station: up the starboard
// side from the waterline, over the crown, and down the port side.
func (hull boatHull) upperRing(at float64) []maths.Vec {
	ring := make([]maths.Vec, 0, 2*hull.sides+3)
	for step := range hull.sides + spaceOne {
		across, up := hull.upperAt(at, step)
		ring = append(ring, hull.point(at, across, up))
	}
	ring = append(ring, hull.point(at, spaceZero, hull.deck(at)))
	for step := hull.sides; step >= spaceZero; step-- {
		across, up := hull.upperAt(at, step)
		ring = append(ring, hull.point(at, -across, up))
	}
	return ring
}

// deckRing is the deck's section at a station: a slab a little inside
// the edge, over the topsides and a little above them.
func (hull boatHull) deckRing(at float64) []maths.Vec {
	edge := hull.half(at) * hull.flaring(at) * boatDeckInset
	sheer, crown := hull.sheer(at), hull.deck(at)+boatDeckLift
	return []maths.Vec{hull.point(at, edge, sheer-boatDeckThick),
		hull.point(at, edge, sheer+boatDeckLift),
		hull.point(at, spaceZero, crown),
		hull.point(at, -edge, sheer+boatDeckLift),
		hull.point(at, -edge, sheer-boatDeckThick)}
}

// boatLoft is a solid lofted through rings, as the Alfa's sail is, with
// the faces that cover nothing left out: at the bow the rings close to a
// line, and what they join there has no area.
type boatLoft struct{ alfaLoft }

// skin is the loft's faces, those with an area.
func (shape boatLoft) skin(fine cut) surface {
	return shape.alfaLoft.skin(fine).withArea()
}

// lofted is a solid through the sections a ring makes at each station.
func (hull boatHull) lofted(ring func(float64) []maths.Vec) boatLoft {
	rings := make([][]maths.Vec, 0, hull.stations+spaceOne)
	for step := range hull.stations + spaceOne {
		rings = append(rings, ring(float64(step)/float64(hull.stations)))
	}
	return boatLoft{alfaLoft{rings: rings}}
}

// halfAt is half the deck's width at a place along the boat, in metres
// from its middle: where its edge stands out from the centreline.
func (hull boatHull) halfAt(z float64) float64 {
	at := math.Min(spaceOne, math.Max(spaceZero, z/hull.length+spaceHalf))
	return hull.half(at) * hull.flaring(at) * boatDeckInset
}

// deckAt is how high the middle of the deck stands at a place along the
// boat, in metres from its middle.
func (hull boatHull) deckAt(z float64) float64 {
	at := math.Min(spaceOne, math.Max(spaceZero, z/hull.length+spaceHalf))
	return hull.deck(at) + boatDeckLift
}

// room is the narrowest the deck is between two places along the boat,
// half across: what a house or a canopy that long may not be wider than.
func (hull boatHull) room(first, last float64) float64 {
	narrow := hull.halfAt(first)
	for step := range boatRoomSteps + spaceOne {
		z := first + (last-first)*float64(step)/boatRoomSteps
		narrow = math.Min(narrow, hull.halfAt(z))
	}
	return narrow
}

// boatRoomSteps is how many places along a stretch room looks at:
// chosen, enough that no bend of the plan hides between them.
const boatRoomSteps = 12
