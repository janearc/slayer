package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The sampan: a canopy of matting or tarpaulin, arched or flat, over the
// middle of a low wooden hull; an outboard motor or a sculling oar at the
// stern; tyres or rope along the sides; a lantern under the canopy; and
// crates and nets on the deck. Every proportion here is chosen.
const (
	boatCanopyAt   = 0.05  // chosen: how far aft of the middle the canopy's middle stands, as a share of the length
	boatCanopyWide = 0.85  // chosen: how much of the deck's room across the canopy takes
	boatArchSpring = 0.45  // chosen: how much of the canopy's height its arch springs from
	boatArchLap    = 1.1   // chosen: how much longer than its share of the arch a slab is, so no gap shows
	boatSlab       = 0.04  // chosen: metres a canopy's slab is thick
	boatPost       = 0.03  // chosen: metres a canopy's post is across, in radius
	boatCargoFore  = 0.38  // chosen: how far forward of the middle crates may stand, as a share of the length
	boatTyre       = 0.25  // chosen: metres across a tyre's radius
	boatTyreWide   = 0.2   // chosen: metres a tyre is wide
	boatRope       = 0.025 // chosen: metres a rope is across, in radius
	boatRopeLegs   = 6     // chosen: straight legs a rope along a side is laid in
	boatTransomGap = 0.2   // chosen: metres a stern fitting stands inside the transom
)

// boatEngine is an outboard motor's box: its width, height and length,
// chosen.
var boatEngine = maths.V(0.28, 0.45, 0.3)

// sampan builds the sampan's fittings and decides where its lights go.
func (craft *boat) sampan() {
	hull := craft.hull
	long := hull.length * craft.design.pick("canopy-length", craft.from)
	centre := hull.length * boatCanopyAt
	first, last := centre-long*spaceHalf, centre+long*spaceHalf
	wide := 2 * boatCanopyWide * hull.room(first, last)
	deck := hull.deckAt(centre)
	high := craft.design.pick("canopy-height", craft.from)
	craft.canopy(centre, long, wide, deck, high)
	craft.sampanLights(first, wide, deck, high, centre)
	craft.propulsion()
	craft.fenders()
	craft.loaded("crates", -boatCargoFore*hull.length, first-boatLoadGap,
		boatDrawCrates, maths.V(0.5, 0.3, 0.4), boatCrate)
	craft.loaded("nets", last+boatLoadGap,
		hull.length*spaceHalf-boatNetAft, boatDrawNets,
		maths.V(0.9, 0.2, 0.6), boatNet)
	craft.interior(first, last, centre, high)
	craft.bowNumbers(hull.rail * boatNumberTall)
}

// boatLoadGap and boatNetAft are how far a load stands from the canopy
// and how far forward of the transom the nets stop. Chosen.
const (
	boatLoadGap = 0.3
	boatNetAft  = 1.4
)

// canopy is the roof over the middle, arched of slabs or flat, on posts.
func (craft *boat) canopy(centre, long, wide, deck, high float64) {
	reach := high
	if craft.design.chance("canopy-arched", craft.from) {
		craft.arch(centre, long, wide, deck, high)
		craft.hoops(centre, long, wide, deck, high)
		reach = boatArchSpring * high
	} else {
		craft.block(0, deck+high, centre, wide, boatSlab, long, boatCanopy)
		craft.laths(maths.V(0, deck+high+boatSlab, centre), maths.XAxis,
			maths.YAxis, centre, long, wide)
		craft.beams(centre, long, wide, deck, high)
	}
	if !craft.detailed(boatClumps) {
		return
	}
	for index := range 4 {
		across := boatSide(index) * wide * spaceHalf * boatPostInset
		along := centre + boatSide(index/2)*long*spaceHalf*boatPostInset
		craft.rod(maths.V(across, deck-boatDeckLift, along),
			maths.V(across, deck+reach, along), boatPost, boatDeck)
	}
}

// boatPostInset is how far in from the canopy's corner a post stands,
// as a share of its half width. Chosen.
const boatPostInset = 0.95

// arch is the canopy as slabs along an arc: the arc's rise is a part of
// the canopy's height, and no more than the half width it spans, so the
// arc is no more than a half circle.
func (craft *boat) arch(centre, long, wide, deck, high float64) {
	radius, middle, reach := archShape(wide, deck, high)
	slabs := boatArchSlabs + 2*craft.rank
	for index := range slabs {
		turn := -reach + 2*reach*(float64(index)+spaceHalf)/float64(slabs)
		sin, cos := math.Sincos(turn)
		chord := 2 * radius * math.Sin(reach/float64(slabs)) * boatArchLap
		normal := maths.V(sin, cos, 0)
		at := maths.V(radius*sin, middle+radius*cos, centre)
		craft.assembly.add(boxShape(chord, boatSlab, long),
			maths.NewSubspace(at.Sub(normal.Scale(boatSlab)),
				maths.Orientation{X: maths.V(cos, -sin, 0), Y: normal,
					Z: maths.ZAxis}, spaceOne), boatCanopy)
		craft.laths(at, maths.V(cos, -sin, 0), normal, centre, long, chord)
	}
}

// laths are the thin bamboo battens laid along a canopy's length over a
// slab, which hold the matting or the tarpaulin down: a few across a slab's
// chord, each on its surface, standing a little proud of it.
func (craft *boat) laths(at, across, normal maths.Vec, centre, long,
	chord float64) {
	if !craft.detailed(boatQuick) {
		return
	}
	for lath := range boatLaths {
		spot := at.Add(across.Scale(chord * (float64(lath) + spaceHalf -
			boatLaths*spaceHalf) / boatLaths)).Add(normal.Scale(boatLathRod))
		craft.rod(spot.Add(maths.V(0, 0, -long*spaceHalf)), spot.Add(
			maths.V(0, 0, long*spaceHalf)), boatLathRod, boatBamboo)
	}
}

// How many laths lie across a slab and how thick each is, in metres.
// Chosen.
const (
	boatLaths   = 3
	boatLathRod = 0.012
)

// archShape is the circle a canopy's arch lies on: its radius, the height
// of its centre, and the angle each side stands at from the top.
func archShape(wide, deck, high float64) (radius, middle, reach float64) {
	half := wide * spaceHalf
	rise := math.Min((spaceOne-boatArchSpring)*high, half*boatArchFlat)
	radius = (rise*rise + half*half) / (2 * rise)
	return radius, deck + high - radius, math.Asin(half / radius)
}

// hoops are the bent hoops under an arched canopy, a few along its length,
// each a run of rods round the arc, and, at a higher detail, the lashings
// that hold the tarpaulin over them, a run of rods outside the slabs.
func (craft *boat) hoops(centre, long, wide, deck, high float64) {
	if !craft.detailed(boatClumps) {
		return
	}
	radius, middle, reach := archShape(wide, deck, high)
	count := craft.design.count("hoops", craft.from)
	for hoop := range count {
		along := centre + long*((float64(hoop)+spaceHalf)/float64(count)-
			spaceHalf)
		craft.arc(along, middle, radius-boatSlab*boatHoopIn, reach, boatHoopRod,
			boatDeck)
		if craft.detailed(boatQuick) {
			craft.arc(along+boatLashStep, middle, radius+boatSlab*boatHoopIn,
				reach, boatRope, boatNet)
			craft.ties(along+boatLashStep, radius*math.Sin(reach), middle+
				radius*math.Cos(reach))
		}
	}
}

// beams are what a flat canopy has for hoops: a beam across under its roof
// at each place the design's hoops would stand, and, at a higher detail, a
// lashing of rope over the roof and a tie down to each gunwale.
func (craft *boat) beams(centre, long, wide, deck, high float64) {
	if !craft.detailed(boatClumps) {
		return
	}
	count := craft.design.count("hoops", craft.from)
	for beam := range count {
		along := centre + long*((float64(beam)+spaceHalf)/float64(count)-
			spaceHalf)
		craft.rod(maths.V(-wide*spaceHalf, deck+high-boatHoopRod, along),
			maths.V(wide*spaceHalf, deck+high-boatHoopRod, along), boatHoopRod,
			boatDeck)
		if craft.detailed(boatQuick) {
			over := deck + high + boatSlab + boatRope
			craft.rod(maths.V(-wide*spaceHalf, over, along+boatLashStep),
				maths.V(wide*spaceHalf, over, along+boatLashStep), boatRope,
				boatNet)
			craft.ties(along+boatLashStep, wide*spaceHalf, deck+high)
		}
	}
}

// ties are the two ropes that hold the tarpaulin down at a lashing: one
// from each edge of the canopy to the gunwale below it.
func (craft *boat) ties(along, reach, level float64) {
	hull := craft.hull
	at := along/hull.length + spaceHalf
	for index := range 2 {
		turn := boatSide(index)
		foot := maths.V(turn*hull.skin(at, hull.sheer(at)), hull.sheer(at)+
			hull.rail, along)
		craft.rod(maths.V(turn*reach, level, along), foot, boatTieRod, boatNet)
	}
}

// boatTieRod is how thick a tie is, in metres: chosen.
const boatTieRod = 0.01

// arc is a run of rods round an arc of a circle in the plane across the
// boat, from one side of the canopy over the top to the other, at a place
// along it.
func (craft *boat) arc(along, middle, radius, reach, thick float64,
	material slot) {
	sides := boatArcSides + 2*craft.rank
	point := func(step int) maths.Vec {
		sin, cos := math.Sincos(-reach + 2*reach*float64(step)/float64(sides))
		return maths.V(radius*sin, middle+radius*cos, along)
	}
	for step := range sides {
		craft.rod(point(step), point(step+spaceOne), thick, material)
	}
}

// The hoops: how many slabs' thickness in under the arch a hoop runs and out
// over it a lashing, a hoop's radius, how far a lashing stands along from its
// hoop, and the fewest rods an arc is made of. All chosen.
const (
	boatHoopIn   = 1.5
	boatHoopRod  = 0.02
	boatLashStep = 0.06
	boatArcSides = 4
)

// boatArchSlabs is the fewest slabs an arch is laid in, and boatArchFlat
// the share of its half width its rise may come to. Chosen.
const (
	boatArchSlabs = 3
	boatArchFlat  = 0.9
)

// sampanLights decides where the sampan's lights stand: the side lights
// on the canopy's front corners, a lantern under it, a lamp on a pole at
// the stern, and a mast pole over the canopy for the top light.
func (craft *boat) sampanLights(first, wide, deck, high, centre float64) {
	reach := wide*spaceHalf + boatSideLamp
	level := deck + boatSideHeight*high
	craft.mounts.port = maths.V(-reach, level, first+boatSideLamp)
	craft.mounts.starboard = maths.V(reach, level, first+boatSideLamp)
	craft.mounts.working = []boatWorking{{lamp: "lantern", lens: boatWarmLens,
		at: maths.V(0, deck+boatLanternHeight*high, centre), toward: maths.Up}}
	stern := craft.hull.length*spaceHalf - boatTransomGap
	level = craft.hull.deckAt(stern)
	craft.mounts.stern = maths.V(boatSternPost, level+boatPostHigh, stern)
	craft.rod(maths.V(boatSternPost, level-boatDeckLift, stern),
		maths.V(boatSternPost, level+boatPostHigh, stern), boatPost, boatSteel)
	pole := maths.V(0, deck+high-boatDeckLift, first+boatPoleBack)
	craft.mounts.top = pole.Add(maths.V(0, craft.design.pick("mast-height",
		craft.from), 0))
	craft.rod(pole, craft.mounts.top, boatPost, boatDeck)
}

// Where the sampan's lights stand: the side lights' distance off the
// canopy and their height as a share of the canopy's, the lantern's
// height as a share of it, the stern lamp's post off the centreline and
// its height, and how far aft of the canopy's front the pole stands.
// All chosen.
const (
	boatSideLamp      = 0.05
	boatSideHeight    = 0.55
	boatLanternHeight = 0.7
	boatSternPost     = 0.3
	boatPostHigh      = 0.7
	boatPoleBack      = 0.25
)

// propulsion is an outboard motor on the transom, which makes the
// sampan a power vessel, or else a sculling oar over the stern.
func (craft *boat) propulsion() {
	stern := craft.hull.length*spaceHalf - boatTransomGap
	deck := craft.hull.deckAt(stern)
	if craft.design.chance("outboard", craft.from) {
		craft.mounts.power = true
		craft.block(0, deck, stern, boatEngine.X, boatEngine.Y, boatEngine.Z,
			boatSteel)
		leg := stern + boatTransomGap
		craft.rod(maths.V(0, deck+boatEngine.Y*spaceHalf, leg),
			maths.V(0, -boatLeg, leg), boatPost, boatSteel)
		craft.outboard(stern, deck)
		return
	}
	craft.sculling(stern, deck)
}

// outboard is what goes with a motor: a cowling's cap, a tiller out ahead of
// it along the boat, a handle across the tiller's end, and a fuel can on the
// sole beside it.
func (craft *boat) outboard(stern, deck float64) {
	if !craft.detailed(boatClumps) {
		return
	}
	top := deck + boatEngine.Y
	craft.block(0, top, stern, boatEngine.X*boatCowlWide, boatCowlTall,
		boatEngine.Z*boatCowlWide, boatSteel)
	end := maths.V(0, top*boatTillerRise, stern-boatTillerLong)
	craft.rod(maths.V(0, deck+boatEngine.Y*spaceHalf, stern-boatEngine.Z*
		spaceHalf), end, boatRodThin, boatSteel)
	craft.rod(end.Add(maths.V(-boatTillerGrip, 0, 0)), end.Add(maths.V(
		boatTillerGrip, 0, 0)), boatRodThin, boatRubber)
	across := boatFuelAside * craft.hull.halfAt(stern)
	craft.block(across, deck-boatDeckLift, stern-boatFuelBack, boatFuel.X,
		boatFuel.Y, boatFuel.Z, boatBucket)
	craft.rod(maths.V(across, deck+boatFuel.Y, stern-boatFuelBack),
		maths.V(across, deck+boatFuel.Y+boatCowlTall, stern-boatFuelBack),
		boatRodThin, boatSteel)
}

// sculling is the oar of a boat with no motor: a post with a pin at the
// stern, a rope loop over the pin, and the oar itself with a blade.
func (craft *boat) sculling(stern, deck float64) {
	pin := maths.V(boatOarPin, deck+boatOarUp, stern-boatOarIn)
	tip := maths.V(boatOarPin+boatOarAside, -boatOarDeep, stern+boatOarOut)
	craft.rod(pin, tip, boatPost, boatDeck)
	if !craft.detailed(boatClumps) {
		return
	}
	craft.rod(maths.V(boatOarPin, deck-boatDeckLift, stern-boatOarIn),
		pin.Add(maths.V(0, boatOarLock, 0)), boatPost*boatCapWide, boatDeck)
	craft.ring(pin, maths.XAxis, maths.ZAxis, boatPost*boatCapWide,
		craft.round(), boatRodThin, boatNet)
	craft.block(tip.X, tip.Y, tip.Z, boatBladeWide, boatBladeLong,
		boatBladeLong, boatDeck)
}

// A motor's fittings: how much its cowling's cap is wider than the motor,
// how tall it is, the tiller's rise as a share of the motor's height, its
// length and its grip's half width, and a fuel can's size, where it stands
// across as a share of the half width and how far forward of the stern.
// An oar's lock stands this tall over its pin. All chosen.
const (
	boatCowlWide   = 1.05
	boatCowlTall   = 0.06
	boatTillerRise = 0.9
	boatTillerLong = 0.9
	boatTillerGrip = 0.12
	boatFuelAside  = 0.45
	boatFuelBack   = 0.8
	boatOarLock    = 0.08
)

// boatFuel is a fuel can's width, height and length. Chosen.
var boatFuel = maths.V(0.2, 0.3, 0.3)

// The motor's leg and the oar: how far the leg reaches under the water,
// and the oar's pin off the centreline, its rise over the deck, its
// distance inside the stern, how far it goes out aft, aside and down.
// All chosen.
const (
	boatLeg      = 0.35
	boatOarPin   = 0.1
	boatOarUp    = 0.4
	boatOarIn    = 0.1
	boatOarOut   = 2.0
	boatOarAside = 0.25
	boatOarDeep  = 0.5
)

// fenders hang along the sides: tyres, or else a rope in straight legs.
func (craft *boat) fenders() {
	if !craft.detailed(boatQuick) {
		return
	}
	if craft.design.chance("tyre-fenders", craft.from) {
		craft.tyres()
		return
	}
	for index := range 2 {
		craft.rope(boatSide(index))
	}
}

// tyres hangs the design's tyres along both sides, a pair of them at
// each of its places, a tyre's axle across the boat.
func (craft *boat) tyres() {
	hull := craft.hull
	count := craft.design.count("tyres", craft.from)
	pairs := (count + 1) / 2
	for index := range count {
		place := (float64(index/2) + spaceHalf) / float64(pairs)
		along := hull.length * (boatTyreFore + boatTyreSpan*place)
		turn := boatSide(index)
		at := along/hull.length + spaceHalf
		up := hull.sheer(at) + hull.rail - boatTyre
		inner := turn * (hull.skin(at, up) + boatPlankThick - boatTyreSink)
		craft.assembly.rodAlong(maths.V(inner, up, along),
			maths.V(turn*boatTyreWide, 0, 0), boatTyre, boatRubber)
	}
}

// Where tyres hang: the first place's distance from the middle as a
// share of the length, forward, the span of them, and how far a tyre's
// inner face is sunk into the hull's side. Chosen.
const (
	boatTyreFore = -0.38
	boatTyreSpan = 0.76
	boatTyreSink = 0.02
)

// rope is a rope along one side, in straight legs that follow the
// deck's edge a little below it.
func (craft *boat) rope(side float64) {
	hull := craft.hull
	point := func(leg int) maths.Vec {
		along := hull.length * (boatTyreFore + boatTyreSpan*float64(leg)/
			boatRopeLegs)
		return maths.V(side*(hull.halfAt(along)+boatRope), hull.deckAt(along)-
			hull.camber-boatRopeDrop, along)
	}
	for leg := range boatRopeLegs {
		craft.rod(point(leg), point(leg+1), boatRope, boatNet)
	}
}

// boatRopeDrop is how far below the deck a rope hangs, in metres.
// Chosen.
const boatRopeDrop = 0.15

// interior is what is in the open boat: ribs up the inside of the wall,
// thwarts across, a bamboo pole along the gunwale, coils and buckets on the
// sole, and under the canopy a stove with a kettle and a lantern on its hook.
func (craft *boat) interior(first, last, centre, high float64) {
	if !craft.detailed(boatClumps) {
		return
	}
	craft.ribs()
	craft.thwarts()
	craft.pole()
	hull := craft.hull
	sole := hull.deckAt(centre) - boatDeckLift
	craft.hook(centre, sole, high)
	if !craft.detailed(boatQuick) {
		return
	}
	craft.stove(first, sole)
	coils := craft.design.count("coils", craft.from)
	for index := range coils {
		along := hull.length * (-spaceHalf + boatBowEnd + boatCoilFore*
			(float64(index)+spaceHalf)/float64(coils))
		craft.coil(boatSide(index)*hull.halfAt(along)*boatSoleAside,
			hull.deckAt(along)-boatDeckLift, along)
	}
	for index := range craft.design.count("buckets", craft.from) {
		along := last + boatBucketGap*float64(index+spaceOne)
		craft.bucket(boatSide(index)*hull.halfAt(along)*boatSoleAside,
			hull.deckAt(along)-boatDeckLift, along)
	}
}

// Where the sampan's gear lies on the sole: the span of the bow's coils as a
// share of the length, how far out across they and the buckets stand as a
// share of the half width, and the distance between buckets aft of the
// canopy. All chosen.
const (
	boatCoilFore  = 0.18
	boatSoleAside = 0.5
	boatBucketGap = 0.35
)

// ribs are the frames of the hull, seen inside the open boat: one up each
// side of the wall at every pitch, and a floor timber across the sole
// between them.
func (craft *boat) ribs() {
	hull := craft.hull
	count := craft.design.count("ribs", craft.from)
	if !craft.detailed(boatQuick) {
		count /= 2
	}
	for rib := range count {
		along := hull.length * (boatBowEnd - spaceHalf + (boatSternEnd-
			boatBowEnd)*(float64(rib)+spaceHalf)/float64(count))
		sole := hull.deckAt(along) - boatDeckLift
		half := hull.halfAt(along) - boatWallThick
		for index := range 2 {
			craft.rod(maths.V(boatSide(index)*half, sole, along),
				maths.V(boatSide(index)*half, sole+hull.rail, along), boatRib,
				boatDeck)
		}
		craft.block(0, sole, along, 2*half, boatRib, boatRib*2, boatDeck)
		if !craft.detailed(boatQuick) {
			continue
		}
		if rib%2 == 0 {
			craft.tholes(along, along/hull.length+spaceHalf)
		}
		pitch := hull.length * (boatSternEnd - boatBowEnd) / float64(count)
		craft.boards(along-pitch*spaceHalf, pitch-boatBoardGap, half)
	}
}

// boards are the boards of the sole between two ribs, laid along the boat
// side by side across it, a bay's worth at a place of a length.
func (craft *boat) boards(along, long, half float64) {
	sole := craft.hull.deckAt(along) - boatDeckLift + boatRib
	wide := 2 * half / boatBoards
	for board := range boatBoards {
		craft.block((float64(board)+spaceHalf-boatBoards*spaceHalf)*wide, sole,
			along, wide-boatBoardGap, boatBoardTall, long, boatDeck)
	}
}

// How many boards lie across the sole, how thick a board is and the gap
// between boards, in metres. Chosen.
const (
	boatBoards    = 3
	boatBoardTall = 0.015
	boatBoardGap  = 0.01
)

// tholes are the pins in the gunwale an oar or a line is made fast to: one
// on each side at a rib, standing up from the cap rail.
func (craft *boat) tholes(along, station float64) {
	hull := craft.hull
	top := hull.sheer(station) + hull.rail + boatCapThick
	for index := range 2 {
		across := boatSide(index) * (hull.skin(station, hull.sheer(station)) -
			boatWallThick)
		craft.rod(maths.V(across, top-boatCapThick, along), maths.V(across, top+
			boatTholeTall, along), boatTholeRod, boatDeck)
	}
}

// A thole pin's height over the cap rail and its thickness, in metres.
// Chosen.
const (
	boatTholeTall = 0.12
	boatTholeRod  = 0.015
)

// boatRib is how thick a rib is, in metres: chosen.
const boatRib = 0.03

// thwarts are the seats across the boat: a plank laid across at the height
// of the wall's middle, one at each of a few places, a knee under each end.
func (craft *boat) thwarts() {
	hull := craft.hull
	for _, place := range []float64{boatThwartBow, boatThwartMid,
		boatThwartStern} {
		along := hull.length * place
		half := hull.halfAt(along) - boatWallThick
		up := hull.deckAt(along) - boatDeckLift + hull.rail*spaceHalf
		craft.block(0, up, along, 2*half, boatSlab, boatThwartLong, boatDeck)
		for index := range 2 {
			craft.block(boatSide(index)*(half-boatRib), up-hull.rail*
				spaceHalf, along, boatRib, hull.rail*spaceHalf, boatRib*2,
				boatDeck)
		}
	}
}

// Where the thwarts stand, as shares of the length from the middle, and
// how long a plank is along the boat. Chosen.
const (
	boatThwartBow   = -0.3
	boatThwartMid   = 0.2
	boatThwartStern = 0.35
	boatThwartLong  = 0.25
)

// pole is a bamboo pole laid along the starboard gunwale, from near the bow
// to past the stern, resting a little higher aft than forward.
func (craft *boat) pole() {
	hull := craft.hull
	fore, aft := hull.length*boatPoleFore, hull.length*boatPoleAft
	level := func(along float64) maths.Vec {
		at := along/hull.length + spaceHalf
		return maths.V(hull.skin(at, hull.sheer(at))*boatPoleAside,
			hull.sheer(at)+hull.rail+boatCapThick+boatPoleRadius, along)
	}
	craft.rod(level(fore), level(aft), boatPoleRadius, boatBamboo)
}

// Where the bamboo pole lies: its ends as shares of the length from the
// middle, how far in across it rests as a share of the gunwale, and its
// radius in metres. Chosen.
const (
	boatPoleFore   = -0.4
	boatPoleAft    = 0.55
	boatPoleAside  = 0.9
	boatPoleRadius = 0.03
)

// hook is the lantern's hook: a rod down from the canopy's underside and a
// short turn at its foot, where the lantern hangs.
func (craft *boat) hook(centre, sole, high float64) {
	top := sole + high
	foot := sole + boatLanternHeight*high + boatHookDrop
	craft.rod(maths.V(0, top, centre), maths.V(0, foot, centre), boatRodThin*
		spaceHalf, boatSteel)
	craft.rod(maths.V(0, foot, centre), maths.V(0, foot, centre+boatHookTurn),
		boatRodThin*spaceHalf, boatSteel)
	if craft.detailed(boatQuick) {
		craft.lantern(centre, foot-boatHookDrop)
	}
}

// lantern is the frame of the lantern that hangs on the hook: a base, and a
// post at each corner, the lamp itself standing in it. It has no cover over
// the lamp, which shines up.
func (craft *boat) lantern(centre, level float64) {
	craft.block(0, level-boatLanternTall*spaceHalf, centre, boatLanternWide,
		boatRodThin*spaceHalf, boatLanternWide, boatSteel)
	for index := range 4 {
		craft.rod(maths.V(boatSide(index)*boatLanternWide*spaceHalf,
			level-boatLanternTall*spaceHalf, centre+boatSide(index/2)*
				boatLanternWide*spaceHalf), maths.V(boatSide(index)*
			boatLanternWide*spaceHalf, level+boatLanternTall*spaceHalf, centre+
			boatSide(index/2)*boatLanternWide*spaceHalf), boatRodThin*spaceHalf,
			boatSteel)
	}
}

// A lantern's width and height, in metres. Chosen.
const (
	boatLanternWide = 0.1
	boatLanternTall = 0.2
)

// A hook's depth under the lantern's own height, and how far its turn runs
// along the boat. Chosen.
const (
	boatHookDrop = 0.12
	boatHookTurn = 0.08
)

// stove is a small stove with a kettle on it, on the sole under the front of
// the canopy: a box, a pipe and a kettle's body, spout and handle.
func (craft *boat) stove(first, sole float64) {
	along := first + boatStoveIn
	across := craft.hull.halfAt(along) * boatStoveAside
	craft.block(across, sole, along, boatStove.X, boatStove.Y, boatStove.Z,
		boatSteel)
	top := sole + boatStove.Y
	craft.rod(maths.V(across, top, along), maths.V(across, top+boatKettle,
		along), boatKettle*spaceHalf, boatSteel)
	craft.rod(maths.V(across, top+boatKettle, along), maths.V(across+
		boatKettle, top+boatKettle*boatSpout, along), boatRodThin, boatSteel)
	craft.rod(maths.V(across-boatKettle*spaceHalf, top+boatKettle, along),
		maths.V(across+boatKettle*spaceHalf, top+boatKettle, along), boatRodThin,
		boatSteel)
}

// boatStove is the stove's width, height and length. Where it stands: how
// far in from the canopy's front and how far across as a share of the half
// width. The kettle's size in metres, and how high its spout stands as a
// share of it. All chosen.
var boatStove = maths.V(0.3, 0.25, 0.3)

const (
	boatStoveIn    = 0.6
	boatStoveAside = 0.35
	boatKettle     = 0.18
	boatSpout      = 0.7
)
