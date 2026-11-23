package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The deck gear of a boat, every piece a real part: cleats and bollards,
// buckets, coils of rope, lifebuoys on brackets, ladders, railings, crates
// stacked, a net's floats, hatches with their coamings, an anchor in its
// hawse pipe with a windlass and its chain, and a rudder and a propeller
// under the stern. Each is a few boxes and rods; the repeats share a mesh.
// Every proportion is chosen.
const (
	boatRodThin    = 0.02  // chosen: metres a railing's rod is across, in radius
	boatCleatBase  = 0.04  // chosen: metres a cleat's base is tall
	boatCleatLong  = 0.12  // chosen: metres a cleat's horn runs each way from its post
	boatBollard    = 0.07  // chosen: metres a bollard is across, in radius
	boatBucketTall = 0.28  // chosen: metres a bucket is tall
	boatBucketWide = 0.13  // chosen: metres a bucket is across, in radius
	boatCoilRadius = 0.3   // chosen: metres a coil of rope is across, in radius
	boatCoilRope   = 0.025 // chosen: metres a rope is across, in radius
	boatBuoyRadius = 0.3   // chosen: metres a lifebuoy is across, in radius
	boatBuoyThick  = 0.055 // chosen: metres a lifebuoy's ring is across, in radius
	boatRungPitch  = 0.3   // chosen: metres between a ladder's rungs
	boatCrateWide  = 0.6   // chosen: metres a fish crate is wide
	boatCrateTall  = 0.3   // chosen: metres a fish crate is tall
	boatCrateLong  = 0.4   // chosen: metres a fish crate is long
)

// ring lays rods round a circle of a radius about a centre, in the plane of
// two directions, so many to the circle: a coil of rope or a lifebuoy.
func (craft *boat) ring(centre, across, up maths.Vec, radius float64,
	sides int, thick float64, material slot) {
	point := func(step int) maths.Vec {
		turn := fullTurn * float64(step) / float64(sides)
		return centre.Add(across.Scale(radius * math.Cos(turn))).
			Add(up.Scale(radius * math.Sin(turn)))
	}
	for step := range sides {
		craft.rod(point(step), point(step+spaceOne), thick, material)
	}
}

// round is how many rods a ring is made of, by the rank of the detail.
func (craft *boat) round() int {
	return [...]int{6, 6, 8, 12}[craft.rank]
}

// cleat is a cleat: a base on the deck and a horn across it, along the
// boat.
func (craft *boat) cleat(across, along float64) {
	deck := craft.hull.deckAt(along) - boatDeckLift
	craft.block(across, deck, along, 0.1, boatCleatBase, 2*boatCleatLong,
		boatSteel)
	horn := deck + boatCleatBase*1.5
	craft.rod(maths.V(across, horn, along-boatCleatLong),
		maths.V(across, horn, along+boatCleatLong), boatRodThin, boatSteel)
}

// cleats are as many cleats as the design says along each side of the
// deck, inside the bulwark or the rail, from the bow to the stern.
func (craft *boat) cleats() {
	if !craft.detailed(boatClumps) {
		return
	}
	count := craft.design.count("cleats", craft.from)
	for index := range count {
		along := craft.hull.length * (boatBowEnd - spaceHalf + (boatSternEnd-
			boatBowEnd)*(float64(index/2)+spaceHalf)/float64((count+1)/2))
		craft.cleat(boatSide(index)*(craft.hull.halfAt(along)-boatCleatSet),
			along)
	}
}

// boatCleatSet is how far in from the deck's edge a cleat stands. Chosen.
const boatCleatSet = 0.2

// bollard is a post on the deck with a cap.
func (craft *boat) bollard(across, along float64) {
	deck := craft.hull.deckAt(along) - boatDeckLift
	craft.rod(maths.V(across, deck, along), maths.V(across, deck+
		boatBollardHigh, along), boatBollard, boatSteel)
	craft.rod(maths.V(across, deck+boatBollardHigh, along), maths.V(across,
		deck+boatBollardHigh+boatBollardCap, along), boatBollard*boatCapWide,
		boatSteel)
}

// A bollard's height, its cap's depth and how much wider than the post
// the cap is. Chosen.
const (
	boatBollardHigh = 0.3
	boatBollardCap  = 0.04
	boatCapWide     = 1.4
)

// bucket is a bucket on the deck, a body and a rim.
func (craft *boat) bucket(across, up, along float64) {
	craft.rod(maths.V(across, up, along), maths.V(across, up+boatBucketTall,
		along), boatBucketWide, boatBucket)
	craft.rod(maths.V(across, up+boatBucketTall, along), maths.V(across,
		up+boatBucketTall+boatRodThin, along), boatBucketWide*boatCapWide,
		boatBucket)
}

// coil is a coil of rope lying flat: two rings of rope, one on the other.
func (craft *boat) coil(across, up, along float64) {
	for layer := range 2 {
		craft.ring(maths.V(across, up+boatCoilRope*float64(layer+spaceOne), along),
			maths.XAxis, maths.ZAxis, boatCoilRadius, craft.round(), boatCoilRope,
			boatNet)
	}
}

// lifebuoy is a ring of an orange stuff standing on a wall that faces out
// to a side, on two brackets, in the plane of the boat's length.
func (craft *boat) lifebuoy(across, up, along float64) {
	craft.ring(maths.V(across, up, along), maths.ZAxis, maths.YAxis,
		boatBuoyRadius, craft.round(), boatBuoyThick, boatLifebuoy)
	for index := range 2 {
		bracket := up + boatBuoyRadius*boatSide(index)
		craft.block(across, bracket-boatRodThin, along, boatBuoyThick,
			boatRodThin*2, boatBuoyThick*2, boatSteel)
	}
}

// ladder is a ladder up a wall that faces out to a side: two stiles and a
// rung at each pitch, from a height to another.
func (craft *boat) ladder(across, along, from, to, wide float64) {
	for index := range 2 {
		stile := along + boatSide(index)*wide*spaceHalf
		craft.rod(maths.V(across, from, stile), maths.V(across, to, stile),
			boatRodThin, boatSteel)
	}
	rungs := int((to - from) / boatRungPitch)
	for rung := 1; rung <= rungs; rung++ {
		craft.rod(maths.V(across, from+float64(rung)*boatRungPitch,
			along-wide*spaceHalf), maths.V(across, from+float64(rung)*
			boatRungPitch, along+wide*spaceHalf), boatRodThin, boatSteel)
	}
}

// railing is a railing along a run at one height: a post at every pitch
// and two rails between the posts, one at the top and one half way.
func (craft *boat) railing(from, to maths.Vec, height, pitch float64) {
	run := to.Sub(from)
	count := max(int(run.Len()/pitch), spaceOne)
	points := make([]maths.Vec, 0, count+spaceOne)
	for step := range count + spaceOne {
		points = append(points, from.Add(run.Scale(float64(step)/float64(count))))
	}
	craft.rails(points, height)
}

// post is a railing's post standing at a place: not again where one stands
// already, as at the corner of two runs, for two posts as one would have
// every edge in common and the mesh would refuse them.
func (craft *boat) post(at maths.Vec, height float64) {
	key := [3]int64{int64(math.Round(at.X * boatPostGrid)),
		int64(math.Round(at.Y * boatPostGrid)),
		int64(math.Round(at.Z * boatPostGrid))}
	if craft.posted[key] {
		return
	}
	if craft.posted == nil {
		craft.posted = map[[3]int64]bool{}
	}
	craft.posted[key] = true
	craft.rod(at.Sub(maths.V(0, boatDeckLift, 0)), at.Add(maths.V(0, height,
		0)), boatRodThin, boatSteel)
}

// boatPostGrid is how many places to the metre a post may stand at:
// posts a thousandth of a metre apart are one post. Chosen.
const boatPostGrid = 1000

// rails is a railing along a path of places: a post at each, and two rails
// from each to the next.
func (craft *boat) rails(points []maths.Vec, height float64) {
	for step, at := range points {
		craft.post(at, height)
		if step == 0 {
			continue
		}
		before := points[step-spaceOne]
		craft.rod(before.Add(maths.V(0, height, 0)), at.Add(maths.V(0,
			height, 0)), boatRodThin, boatSteel)
		craft.rod(before.Add(maths.V(0, height*spaceHalf, 0)), at.Add(
			maths.V(0, height*spaceHalf, 0)), boatRodThin*spaceHalf,
			boatSteel)
	}
}

// crates stack fish crates in a pile at a place: so many wide along the
// boat and so many high, a seed's few more or fewer on top.
func (craft *boat) crates(across, up, along float64, rows, high, stream int) {
	for row := range rows {
		for level := range high {
			if level > 0 && craft.from.draw(housing, stream+row*high+level,
				0) < boatCratesDropped {
				continue
			}
			craft.block(across, up+float64(level)*(boatCrateTall-boatSunk), along+
				float64(row)*boatCrateLong, boatCrateWide, boatCrateTall,
				boatCrateLong, boatCrate)
		}
	}
}

// boatCratesDropped is the share of the crates above the lowest layer a
// seed leaves out, so that a stack is not a block. Chosen.
const boatCratesDropped = 0.3

// hatch is a hatch in the deck: a coaming round it and a cover of planks
// laid across, with two handles.
func (craft *boat) hatch(along float64) {
	deck := craft.hull.deckAt(along) - boatDeckLift
	wide, long := boatHatch.X, boatHatch.Z
	for index := range 2 {
		craft.block(boatSide(index)*wide*spaceHalf, deck, along, boatCoaming,
			boatHatch.Y, long, boatSteel)
		craft.block(0, deck, along+boatSide(index)*long*spaceHalf, wide,
			boatHatch.Y, boatCoaming, boatSteel)
	}
	planks := 3
	for plank := range planks {
		craft.block(0, deck+boatHatch.Y, along+(float64(plank)-spaceOne)*long/
			float64(planks), wide, boatCoaming, long/float64(planks)-boatStrakeGap,
			boatDeck)
	}
	for index := range 2 {
		craft.rod(maths.V(boatSide(index)*wide*boatHandleAt, deck+boatHatch.Y+
			boatCoaming, along), maths.V(boatSide(index)*wide*boatHandleAt,
			deck+boatHatch.Y+boatCoaming*2, along), boatRodThin, boatSteel)
	}
}

// boatHatch is a hatch's coaming: its width, its height and its length.
// Chosen.
var boatHatch = maths.V(1.6, 0.35, 1.3)

// A coaming's thickness and where a hatch's handles stand, as a share of
// its width. Chosen.
const (
	boatCoaming  = 0.06
	boatHandleAt = 0.25
)

// screw is the rudder and the propeller under the stern: a shaft, a boss
// and four blades, and the rudder behind them. They stand under the water.
func (craft *boat) screw() {
	hull := craft.hull
	stern := hull.length * spaceHalf
	low := -hull.draught * boatScrewDepth
	craft.rod(maths.V(0, low, stern-boatShaft), maths.V(0, low, stern+boatBoss),
		boatShaftRadius, boatSteel)
	if !craft.detailed(boatClumps) {
		return
	}
	for blade := range boatBlades {
		turn := fullTurn * float64(blade) / boatBlades
		sin, cos := math.Sincos(turn)
		craft.assembly.add(boxShape(boatBladeWide, boatBladeLong, boatBladeThick),
			maths.NewSubspace(maths.V(-sin*boatBladeRoot, low+cos*boatBladeRoot,
				stern+boatBoss), maths.Orientation{
				X: maths.V(cos, sin, 0), Y: maths.V(-sin, cos, 0),
				Z: maths.ZAxis}, spaceOne), boatSteel)
	}
	craft.block(0, -hull.draught*boatRudderDepth, stern+boatRudderBack,
		boatRudderThick, hull.draught*boatRudderDepth+boatRudderUp,
		boatRudderLong, boatSteel)
}

// The screw and the rudder: how deep the shaft lies as a share of the
// draught, how far its ends stand inside and behind the transom, its
// radius, the number of blades and a blade's size, and the rudder's depth
// as a share of the draught, how far it stands behind the transom, how
// thick and long it is, and how far it rises above the waterline. All
// chosen.
const (
	boatScrewDepth  = 0.55
	boatShaft       = 1.2
	boatBoss        = 0.25
	boatShaftRadius = 0.05
	boatBlades      = 4
	boatBladeWide   = 0.12
	boatBladeLong   = 0.36
	boatBladeThick  = 0.02
	boatBladeRoot   = 0.03 // chosen: metres a blade's foot stands off the axis, so that no two blades share an edge
	boatRudderDepth = 0.9
	boatRudderBack  = 0.55
	boatRudderThick = 0.06
	boatRudderLong  = 0.5
	boatRudderUp    = 0.05
)
