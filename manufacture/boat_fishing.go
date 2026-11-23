package manufacture

import (
	"github.com/janearc/slayer/maths"
)

// The fishing boat: a steel inshore boat with its wheelhouse aft of the
// middle and everything a person on a quay would see on it.
//
// The hull's plating, its bulwark and stanchions, an anchor in its hawse
// pipe with a windlass and chain, bollards and cleats, hatches with
// coamings, a net drum, piled nets, floats, crates, coils of rope and
// buckets, tyres hung as fenders, lifebuoys and a liferaft, the mast with
// its booms and rigging, floodlights on arms, and a number on each bow
// and a name on the stern. Every proportion here is chosen.

// fishing builds the fishing boat's fittings and decides where its lights
// go.
func (craft *boat) fishing() {
	hull, design := craft.hull, craft.design
	at := design.pick("house-position", craft.from)
	long := hull.length * design.pick("house-length", craft.from)
	wide := hull.beam * design.pick("house-width", craft.from)
	centre := hull.z(at)
	base, roof := craft.house(centre, long, wide,
		design.pick("house-height", craft.from))
	craft.roofGear(centre, long, wide, base, roof)
	craft.funnel(wide*boatFunnelAside, centre+long*spaceHalf+boatFunnelBack,
		roof)
	craft.rigging(at, centre, long, roof)
	craft.floods(centre, long, wide, roof)
	craft.fishingLights(centre, long, wide, base, roof)
	craft.foredeck(at)
	craft.afterdeck(centre, long)
	craft.housegear(centre, long, wide, base, roof)
	craft.tyres()
	craft.bowNumbers(hull.rail * boatNumberTall)
	craft.sternName(boatNameTall)
}

// The painted letters of a fishing boat: how tall a number is as a share
// of the bulwark, and how tall a name is, in metres. Chosen.
const (
	boatNumberTall = 0.6
	boatNameTall   = 0.35
)

// boatFunnelAside is how far off the centreline a fishing boat's funnel
// stands, as a share of the house's width. Chosen.
const boatFunnelAside = 0.25

// foredeck is what is forward of the wheelhouse: an anchor in its hawse pipe,
// the windlass and its chain, bollards, hatches, piled nets and coils.
func (craft *boat) foredeck(at float64) {
	hull := craft.hull
	craft.anchorGear()
	for index := range 2 {
		craft.bollard(boatSide(index)*boatBollardAside, hull.z(boatBollardAt))
	}
	if !craft.detailed(boatClumps) {
		return
	}
	for index := range craft.design.count("hatches", craft.from) {
		craft.hatch(hull.z(at - boatHatchGap - float64(index)*boatHatchStep))
	}
	craft.loaded("nets", hull.z(boatNetFore), hull.z(at-boatMastFore*2),
		boatDrawNets, maths.V(1.2, 0.3, 0.8), boatNet)
	craft.floatLine()
	for index := range craft.design.count("coils", craft.from) {
		along := hull.z(boatBollardAt + boatCoilStep*float64(index/2+1))
		craft.coil(boatSide(index)*boatCoilAside, hull.deckAt(along)-
			boatDeckLift, along)
	}
}

// The foredeck: how far off the centreline the bollards stand and how far
// aft of the stem as a share of the length; how far forward of the
// wheelhouse the first hatch stands and how much further each next one
// does; where the nets begin; and where the coils lie. All chosen.
const (
	boatBollardAside = 0.4
	boatBollardAt    = 0.07
	boatHatchGap     = 0.1
	boatHatchStep    = 0.15
	boatNetFore      = 0.12
	boatCoilStep     = 0.03
	boatCoilAside    = 0.9
)

// anchorGear is a hawse pipe through each side of the bulwark near the bow,
// an anchor hung in the starboard one, the windlass behind it and the chain
// laid from the windlass to the hawse, and the rust that runs down from it.
func (craft *boat) anchorGear() {
	if !craft.detailed(boatQuick) {
		return
	}
	hull := craft.hull
	along := hull.z(boatAnchorAt)
	at := along/hull.length + spaceHalf
	deck, edge := hull.deckAt(along), hull.skin(at, hull.sheer(at))
	level := deck + hull.rail*boatHawseHigh
	for index := range 2 {
		turn := boatSide(index)
		craft.rod(maths.V(turn*(edge-boatHawseIn), level+boatHawseFall, along),
			maths.V(turn*(edge+boatHawseOut), level-boatHawseFall, along),
			boatBollard, boatSteel)
	}
	hang := edge + boatHawseOut + boatPlateThick
	craft.anchor(hang, level-boatHawseFall-boatHawseOut, along)
	craft.windlass(along + boatWindlassBack)
	craft.chain(maths.V(boatChainStart, deck+boatChainLift, along+
		boatWindlassBack), maths.V(edge-boatHawseIn, level+boatHawseFall, along))
	craft.block(edge+boatStreakThick, level-boatStreakLong, along, boatStreakThick,
		boatStreakLong, boatStreakWide*2, boatRust)
}

// The anchor and what it hangs from: how far aft of the stem the hawse
// stands as a share of the length, how high up the bulwark, how far its
// pipe reaches in and out, how much it falls to the outside, how far aft
// of it the windlass stands, where the chain starts across the deck and its
// height over it. All chosen.
const (
	boatAnchorAt     = 0.07
	boatHawseHigh    = 0.55
	boatHawseIn      = 0.25
	boatHawseOut     = 0.15
	boatHawseFall    = 0.08
	boatWindlassBack = 1.6
	boatChainStart   = 0.35
	boatChainLift    = 0.25
)

// anchor is an anchor hung below its hawse: a shank, a stock across its
// top, two arms and a fluke at the end of each.
func (craft *boat) anchor(across, top, along float64) {
	foot := top - boatShank
	craft.rod(maths.V(across, top, along), maths.V(across, foot, along),
		boatRodThin*1.5, boatSteel)
	craft.rod(maths.V(across, top-boatRodThin, along-boatStock), maths.V(
		across, top-boatRodThin, along+boatStock), boatRodThin, boatSteel)
	for index := range 2 {
		turn := boatSide(index)
		tip := maths.V(across, foot+boatFluke, along+turn*boatStock)
		craft.rod(maths.V(across, foot, along), tip, boatRodThin*1.5, boatSteel)
		craft.block(across, tip.Y, tip.Z, boatRodThin*2, boatFluke, boatFluke,
			boatSteel)
	}
}

// An anchor's shank, its stock's half length and a fluke's size, in
// metres. Chosen.
const (
	boatShank = 0.9
	boatStock = 0.3
	boatFluke = 0.14
)

// windlass is the windlass: a base on the deck, a drum across it, a
// gypsy at each end and a handle.
func (craft *boat) windlass(along float64) {
	deck := craft.hull.deckAt(along) - boatDeckLift
	craft.block(0, deck, along, boatWindlass.X, boatWindlass.Y, boatWindlass.Z,
		boatSteel)
	drum := deck + boatWindlass.Y + boatDrumRadius
	craft.rod(maths.V(-boatWindlass.X*spaceHalf, drum, along), maths.V(
		boatWindlass.X*spaceHalf, drum, along), boatDrumRadius, boatSteel)
	for index := range 2 {
		end := boatSide(index) * boatWindlass.X * spaceHalf
		craft.rod(maths.V(end, drum, along), maths.V(end+boatSide(index)*
			boatRodThin*2, drum, along), boatDrumRadius*boatGypsy, boatRust)
	}
	craft.rod(maths.V(boatWindlass.X*spaceHalf, drum, along), maths.V(
		boatWindlass.X*spaceHalf+boatHandle, drum+boatHandle, along),
		boatRodThin, boatSteel)
}

// The windlass's base, width, height and length; its drum's radius, how
// much wider a gypsy is and how long the handle is. Chosen.
var boatWindlass = maths.V(1.0, 0.25, 0.5)

const (
	boatDrumRadius = 0.13
	boatGypsy      = 1.5
	boatHandle     = 0.4
)

// chain is a chain laid in a run of links between two places on the deck,
// each link a short rod, in turn one on its side and one flat.
func (craft *boat) chain(from, to maths.Vec) {
	run := to.Sub(from)
	for link := range boatLinks {
		start := from.Add(run.Scale(float64(link) / boatLinks))
		end := from.Add(run.Scale((float64(link) + boatLinkFill) / boatLinks))
		lift := maths.V(0, boatSide(link)*boatRodThin, 0)
		craft.rod(start.Add(lift), end.Add(lift), boatRodThin*boatLinkThick,
			boatSteel)
	}
}

// How many links a chain has laid out, how much of its pitch each takes and
// how thick a link is, as a share of a rod. Chosen.
const (
	boatLinks     = 10
	boatLinkFill  = 0.75
	boatLinkThick = 0.9
)

// floatLine is a line of net floats along the starboard bulwark, inside it:
// a line along the rail and a float hung below it at every fifth of a metre.
func (craft *boat) floatLine() {
	hull := craft.hull
	count := craft.design.count("floats", craft.from)
	first, last := hull.z(boatFloatFore), hull.z(boatFloatAft)
	at := first/hull.length + spaceHalf
	across := hull.skin(at, hull.sheer(at)) - boatFloatIn
	level := hull.deckAt(first) + hull.rail*boatFloatHigh
	craft.rod(maths.V(across, level, first), maths.V(across, level, last),
		boatRodThin*spaceHalf, boatNet)
	for index := range count {
		along := first + (last-first)*(float64(index)+spaceHalf)/float64(count)
		craft.block(across, level-boatFloatSize, along, boatFloatSize,
			boatFloatSize, boatFloatSize, boatLifebuoy)
	}
}

// Where the floats hang: how far aft of the stem the line begins and ends,
// as shares of the length, how far inside the bulwark and how high on it,
// and how big a float is. Chosen.
const (
	boatFloatFore = 0.2
	boatFloatAft  = 0.42
	boatFloatIn   = 0.15
	boatFloatHigh = 0.8
	boatFloatSize = 0.12
)

// afterdeck is what is aft of the wheelhouse: the net drum, stacks of fish
// crates, buckets and a coil on the deck.
func (craft *boat) afterdeck(centre, long float64) {
	hull := craft.hull
	if !craft.detailed(boatClumps) {
		return
	}
	craft.drum(hull.z(boatDrumAt))
	stack := centre + long*spaceHalf + boatStackGap
	count := craft.design.count("crates", craft.from)
	high := min(count/boatCratePerStack+spaceOne, boatStackHigh)
	for index := range 2 {
		craft.crates(boatSide(index)*boatStackAside, hull.deckAt(stack)-
			boatDeckLift, stack, 2, high, boatDrawCrates+index*boatStackStream)
	}
	for index := range craft.design.count("buckets", craft.from) {
		along := stack + boatBucketRow*float64(index)
		craft.bucket(boatSide(index)*boatBucketAside, hull.deckAt(along)-
			boatDeckLift, along)
	}
}

// The afterdeck: how far aft of the stem the net drum stands, how far aft
// of the wheelhouse the crates begin, how many crates make a layer of a
// stack and the most layers a stack has, how far off the centreline the
// stacks and the buckets stand, the draws a stack takes and the row a
// bucket stands in. All chosen.
const (
	boatDrumAt        = 0.82
	boatStackGap      = 0.9
	boatCratePerStack = 3
	boatStackHigh     = 4
	boatStackAside    = 0.85
	boatStackStream   = 20
	boatBucketAside   = 0.5
	boatBucketRow     = 0.5
)

// drum is the net drum: a drum across the deck with a flange at each end, on
// two supports.
func (craft *boat) drum(along float64) {
	deck := craft.hull.deckAt(along) - boatDeckLift
	centre := deck + boatDrumSupport + boatNetDrum
	half := boatDrumWide * spaceHalf
	craft.rod(maths.V(-half, centre, along), maths.V(half, centre, along),
		boatNetDrum, boatNet)
	for index := range 2 {
		end := boatSide(index) * half
		craft.rod(maths.V(end, centre, along), maths.V(end+boatSide(index)*
			boatFlange, centre, along), boatNetDrum*boatFlaring, boatSteel)
		craft.block(end, deck, along, boatFlange, boatDrumSupport+boatNetDrum,
			boatDrumFoot, boatSteel)
	}
}

// The net drum: its width across the deck, its radius, its support's
// height, a flange's thickness, how much wider than the drum a flange is
// and the length of a support's foot. Chosen.
const (
	boatDrumWide    = 1.8
	boatNetDrum     = 0.35
	boatDrumSupport = 0.3
	boatFlange      = 0.06
	boatFlaring     = 1.6
	boatDrumFoot    = 0.5
)

// housegear is what is on the walls and roof of the wheelhouse: a
// lifebuoy on each side, and the liferaft in its canister on its cradle.
func (craft *boat) housegear(centre, long, wide, base, roof float64) {
	if !craft.detailed(boatClumps) {
		return
	}
	reach := wide*spaceHalf + boatLifebuoyOut
	for index := range craft.design.count("lifebuoys", craft.from) {
		craft.lifebuoy(boatSide(index)*reach, base+(roof-base)*boatBuoyHigh,
			centre-long*boatBuoyAt)
	}
	canister := maths.V(wide*boatRaftAside, roof+boatRaftRadius,
		centre+long*boatRaftAt)
	craft.rod(canister, canister.Add(maths.V(0, 0, boatRaftLong)),
		boatRaftRadius, boatHouse)
	for index := range 2 {
		craft.block(canister.X, roof, canister.Z+float64(index)*(boatRaftLong-
			boatCoaming), boatRaftRadius*2, boatRaftRadius, boatCoaming,
			boatSteel)
	}
}

// The wheelhouse's gear: how far a lifebuoy stands off the wall, how high
// up and how far forward of the middle as shares of the house, and the
// liferaft's place on the roof, its length and its radius. All chosen.
const (
	boatLifebuoyOut = 0.08
	boatBuoyHigh    = 0.62
	boatBuoyAt      = 0.1
	boatRaftAside   = 0.25
	boatRaftAt      = 0.1
	boatRaftLong    = 1.1
	boatRaftRadius  = 0.25
)

// rigging is the mast, forward of the wheelhouse, a crosstree, a boom to each
// side with the blocks and lines that work it, shrouds to the bulwark and stays
// down to the bow and the roof.
func (craft *boat) rigging(at, centre, long, roof float64) {
	hull := craft.hull
	along := hull.z(at - boatMastFore)
	deck := hull.deckAt(along) - boatDeckLift
	top := maths.V(0, deck+craft.design.pick("mast-height", craft.from), along)
	craft.mounts.top = top
	craft.rod(maths.V(0, deck, along), top, boatMastRadius, boatSteel)
	if !craft.detailed(boatClumps) {
		return
	}
	craft.booms(along, deck, top)
	bow := hull.z(boatStayBow)
	craft.rod(top, maths.V(0, hull.deckAt(bow), bow), boatStay, boatSteel)
	craft.rod(top, maths.V(0, roof, centre+long*spaceHalf), boatStay,
		boatSteel)
	if craft.detailed(boatQuick) {
		craft.shrouds(along, top.Y)
	}
}

// booms are the two booms, each from the mast's foot out and up, with a
// block at its tip, a topping lift to the mast and a line down to the deck.
func (craft *boat) booms(along, deck float64, top maths.Vec) {
	boom := craft.design.pick("boom-length", craft.from)
	for index := range 2 {
		turn := boatSide(index)
		foot := maths.V(0, deck+boatBoomFoot, along)
		tip := maths.V(turn*boom*boatBoomOut, deck+boatBoomFoot+boom*boatBoomUp,
			along-boom*boatBoomFore)
		craft.rod(foot, tip, boatBoomRadius, boatSteel)
		if !craft.detailed(boatQuick) {
			continue
		}
		craft.block(tip.X, tip.Y-boatBlock, tip.Z, boatBlock, boatBlock*2,
			boatBlock, boatRust)
		craft.rod(tip, maths.V(0, top.Y*boatLiftAt+deck*(spaceOne-boatLiftAt),
			along), boatStay, boatNet)
		craft.rod(tip.Sub(maths.V(0, boatBlock, 0)), maths.V(turn*
			boom*boatBoomOut*boatLineIn, deck, along+boatLineAft), boatStay, boatNet)
	}
}

// The booms: a block's size, how far up the mast a topping lift is made
// fast as a share of its height, how far in from the tip a boom's line
// lands as a share of its reach and how far aft. Chosen.
const (
	boatBlock   = 0.07
	boatLiftAt  = 0.85
	boatLineIn  = 0.6
	boatLineAft = 1.0
)

// shrouds are two a side from the mast's crosstree down to the bulwark,
// and the crosstree they spread from.
func (craft *boat) shrouds(along, mast float64) {
	hull := craft.hull
	level := hull.deckAt(along) + (mast-hull.deckAt(along))*boatCrosstreeAt
	craft.block(0, level, along, boatCrosstree, boatRodThin*2, boatRodThin*4,
		boatSteel)
	at := along/hull.length + spaceHalf
	for index := range 4 {
		turn := boatSide(index)
		span := boatSide(index/2) * boatShroudSpan
		craft.rod(maths.V(turn*boatCrosstree*spaceHalf, level, along),
			maths.V(turn*hull.skin(at, hull.sheer(at)), hull.sheer(at)+hull.rail,
				along+span), boatStay, boatSteel)
	}
}

// The crosstree's height up the mast as a share of it, its width and how
// far fore and aft of the mast a shroud lands. Chosen.
const (
	boatCrosstreeAt = 0.7
	boatCrosstree   = 1.4
	boatShroudSpan  = 0.4
)

// The mast, its booms and its stays: how far forward of the wheelhouse the
// mast stands, as a share of the length; its radius; how far out, up and
// forward a boom reaches as shares of its length, how high above the deck
// its foot is and its radius; where the forestay lands, as a share of the
// length from the bow; and the stays' radius. All chosen.
const (
	boatMastFore   = 0.22
	boatMastRadius = 0.09
	boatBoomOut    = 0.8
	boatBoomUp     = 0.45
	boatBoomFore   = 0.2
	boatBoomFoot   = 0.8
	boatBoomRadius = 0.05
	boatStayBow    = 0.04
	boatStay       = 0.012
)

// floods are the floodlights: a pair at the front of the wheelhouse roof
// looking forward and down, and, if the design has more, a pair on the
// mast looking aft and down. Each is a housing on an arm, with a lens, and a lamp.
func (craft *boat) floods(centre, long, wide, roof float64) {
	count := craft.design.count("floods", craft.from)
	for index := range count {
		turn := boatSide(index)
		at := maths.V(turn*wide*boatFloodAside, roof+boatFloodUp,
			centre-long*spaceHalf)
		toward := maths.V(turn*boatFloodOut, -boatFloodDown, -spaceOne)
		if index >= 2 {
			mast := craft.mounts.top
			at = maths.V(turn*boatFloodMast, mast.Y*boatFloodHeight, mast.Z)
			toward = maths.V(turn*boatFloodOut, -boatFloodDown, spaceOne)
		}
		craft.block(at.X, at.Y-boatFloodBox/2, at.Z, boatFloodBox,
			boatFloodBox, boatFloodBox, boatSteel)
		craft.rod(at.Add(maths.V(-turn*boatArm, -boatFloodBox, -toward.Z*
			boatFloodBox)), at, boatRodThin, boatSteel)
		craft.mounts.working = append(craft.mounts.working, boatWorking{
			lamp: "flood", lens: boatWhiteLens, toward: toward.Unit(),
			at: at.Add(toward.Unit().Scale(boatFloodBox))})
	}
}

// Where the floodlights stand and look: how far off the centreline a
// roof light is, as a share of the wheelhouse's width; how high over the
// roof; how far a mast light is off the mast and how high up it as a
// share of the mast's height; how far outward and how far down each
// looks; the housing's size in metres; and how far its arm reaches. All
// chosen.
const (
	boatFloodAside  = 0.4
	boatFloodUp     = 0.2
	boatFloodMast   = 0.2
	boatFloodHeight = 0.6
	boatFloodOut    = 0.35
	boatFloodDown   = 0.45
	boatFloodBox    = 0.22
	boatArm         = 0.3
)

// fishingLights decides where the side lights, the stern lamp and the
// top light stand: the side lights at the wheelhouse's front corners, the
// stern lamp on a post at the transom, the top light on the mast.
func (craft *boat) fishingLights(centre, long, wide, base, roof float64) {
	front := centre - long*spaceHalf
	reach := wide*spaceHalf + boatSideLamp + boatWallLamp
	level := base + boatSideHeight*(roof-base)
	craft.mounts.port = maths.V(-reach, level, front)
	craft.mounts.starboard = maths.V(reach, level, front)
	craft.sternPost()
	craft.mounts.power = true
}

// boatWallLamp is how much further off a wall than a canopy's a house's
// side lights stand, in metres. Chosen.
const boatWallLamp = 0.05
