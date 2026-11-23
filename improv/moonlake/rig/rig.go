// Package rig is where moonlake's light show stands and where it points:
// the laser heads and the figures they throw, the strobes, the
// searchlights and their moves, the backlights and the ship's beam.
//
// It was lightshow's own geometry, moved here unchanged so that lightshow,
// which lights the show, and the untraced films, which draw it on other
// cameras, use the same functions. Every place is metres east, north and
// up from the show's eye, as lightshow has always had them.
package rig

import "math"

// Point is a place or a way from the show's eye: metres east, north and
// up.
type Point [3]float64

// Add is the point moved by another.
func (at Point) Add(by Point) Point {
	return Point{at[0] + by[0], at[1] + by[1], at[2] + by[2]}
}

// Scale is the point so many times over.
func (at Point) Scale(times float64) Point {
	return Point{at[0] * times, at[1] * times, at[2] * times}
}

// dot is the dot product of two points.
func (at Point) dot(other Point) float64 {
	return at[0]*other[0] + at[1]*other[1] + at[2]*other[2]
}

// unit is the point's way, a metre long.
func (at Point) unit() Point { return at.Scale(1 / math.Sqrt(at.dot(at))) }

// cross is the cross product of two points.
func (at Point) cross(other Point) Point {
	return Point{at[1]*other[2] - at[2]*other[1],
		at[2]*other[0] - at[0]*other[2], at[0]*other[1] - at[1]*other[0]}
}

// Toward is the unit way to a compass azimuth and an altitude, degrees.
func Toward(azimuth, altitude float64) Point {
	round, up := azimuth*math.Pi/halfTurn, altitude*math.Pi/halfTurn
	return Point{math.Sin(round) * math.Cos(up),
		math.Cos(round) * math.Cos(up), math.Sin(up)}
}

// bearing is the compass azimuth from one point to another, in degrees.
func bearing(from, to Point) float64 {
	return math.Atan2(to[0]-from[0], to[1]-from[1]) * halfTurn / math.Pi
}

// Layout is the show's geometry for an eye so many metres over the lake
// and towers whose top decks stand so many metres over their ground
// (nought: the heads stand where they stood before the towers).
type Layout struct {
	eyeOverLake, deck float64
}

// NewLayout is the show's geometry for an eye over the lake and a deck
// height, as lightshow's MOONLAKE_EYE and RIG_DECK say them.
func NewLayout(eyeOverLake, deck float64) Layout {
	return Layout{eyeOverLake: eyeOverLake, deck: deck}
}

// Water is the lake's height from the eye: below it.
func (show Layout) Water() float64 { return 0 - show.eyeOverLake }

// farTree is the far tree from the eye: 1727.5 m east, 617.5 m south,
// and as far up as its ground at 1195.5 m stands over the eye.
func (show Layout) farTree() Point {
	return Point{farTreeEast, farTreeNorth,
		farTreeGround - (lakeHeight + show.eyeOverLake)}
}

// Head is a laser head: where it stands, the heading it aims at and the
// degrees its fan is lifted from the rim heads' tilt, which way its
// mirrors swing, which half of a split fan it lights, and its bank.
type Head struct {
	Origin            Point
	Aim, Lift, Mirror float64
	Flip, Bank        int
}

// Heads is the four laser heads: two on the far rim, 120 m either side
// of the far tree, aimed back at the eye; two on the shores, aimed
// across the lake at each other and lifted 7 degrees.
func (show Layout) Heads() []Head {
	far := show.farTree()
	// the differences from the eye are taken as constants, exactly, as
	// lightshow always had them, so its picture is unchanged to the bit
	left := show.shore(shoreLeftEast-eyeEast, shoreLeftNorth-eyeNorth)
	right := show.shore(shoreRightEast-eyeEast, shoreRightNorth-eyeNorth)
	return []Head{
		{show.onDeck(far.Add(Toward(azimuth-quarter, 0).Scale(rimApart)).
			Add(Point{0, 0, rimHeadUp}), 0), azimuth + halfTurn, 0, 1, 0, 0},
		{show.onDeck(far.Add(Toward(azimuth+quarter, 0).Scale(rimApart)).
			Add(Point{0, 0, rimHeadUp}), 1), azimuth + halfTurn, 0, -1, 1, 1},
		{show.onDeck(left, 2), bearing(left, right), shoreLift, 1, 1, 0},
		{show.onDeck(right, 3), bearing(right, left), shoreLift, -1, 0, 1},
	}
}

// shore is a shore installation so far east and north of the eye, 8 m
// over the water.
func (show Layout) shore(east, north float64) Point {
	return Point{east, north, show.Water() + shoreHeadUp}
}

// onDeck is a head lifted onto its tower's top deck, when the towers
// stand, and where it was when they do not.
func (show Layout) onDeck(head Point, tower int) Point {
	if show.deck <= 0 {
		return head
	}
	head[2] = rigGround[tower] - (lakeHeight + show.eyeOverLake) + show.deck
	return head
}

// Strobes is the two strobes: 160 m behind the far tree, 20 m up,
// white; and 220 m behind and 260 m to the left, 30 m up, a magenta gel.
func (show Layout) Strobes() [2]Point {
	far := show.farTree()
	return [2]Point{
		far.Add(Toward(azimuth, 0).Scale(whiteBehind)).Add(
			Point{0, 0, whiteUp}),
		far.Add(Toward(azimuth, 0).Scale(magentaBehind)).Add(
			Toward(azimuth-quarter, 0).Scale(magentaLeft)).Add(
			Point{0, 0, magentaUp}),
	}
}

// Searchlight is searchlight k of so many behind the rim: 200 m past the
// far tree, 10 m up, and 140 m apart either side of it, alternating.
func (show Layout) Searchlight(k int) Point {
	out, rank := searchSide(k)
	behind := show.farTree().Add(Toward(azimuth, 0).Scale(searchBehind)).
		Add(Point{0, 0, searchUp})
	return behind.Add(Toward(azimuth+out*quarter, 0).Scale(
		searchApart * (rank + 1)))
}

// searchSide is which side searchlight k stands, minus one for the left,
// and its rank out from the middle.
func searchSide(k int) (float64, float64) {
	return float64(2*(k%2) - 1), float64(k / 2)
}

// SearchWay is the way searchlight k of count points in a move, a phase
// of its sweep and a share of the move done, as lightshow aims it:
//
//   - sweep, and chase: tilted 30 to 55 degrees up and swung outward,
//     the further out the further, each on the phase and its own offset
//   - fan: all opening outward and down together and closing upward
//     again, once in the move
//   - viewer: all turned on the eye, a little apart
func (show Layout) SearchWay(move string, k, count int, phase,
	movePhase float64) Point {
	out, rank := searchSide(k)
	turn := 2 * math.Pi * phase
	offset := float64(k) * 2 * math.Pi / float64(count)
	opening := half - half*math.Cos(2*math.Pi*movePhase)
	switch move {
	case "fan":
		return Toward(azimuth+out*(fanFirst+(rank+1)*fanStep*opening),
			fanTop-fanDrop*opening)
	case "viewer":
		at := show.Searchlight(k).Scale(-1).unit()
		side := at.cross(Point{0, 0, 1}).unit()
		return at.Add(side.Scale(out * (rank + half) * viewerApart)).unit()
	}
	return Toward(azimuth+out*(sweepOut+sweepRank*rank+sweepSwing*
		math.Sin(turn+offset)), sweepTilt+sweepNod*math.Sin(2*turn+offset))
}

// Running is which searchlight of count a chase has bright a share of
// the way through its move: it runs across them twice.
func Running(count int, movePhase float64) int {
	return int(math.Floor(movePhase*2*float64(count))) % count
}

// Backlights is the two backlights behind the close spruces, from the
// eye: the left white, the right through a magenta gel.
func (show Layout) Backlights() [2]Point {
	up := backlightGround - (lakeHeight + show.eyeOverLake)
	return [2]Point{{backLeftEast, backLeftNorth, up},
		{backRightEast, backRightNorth, up}}
}

// Ship is the ship's beam: its apex so many metres over the water and
// where it lands, so many metres out along the view, and its radius
// there.
func (show Layout) Ship(out, apex float64) (top, foot Point) {
	axis := Toward(azimuth, 0).Scale(out)
	return axis.Add(Point{0, 0, show.Water() + apex}),
		axis.Add(Point{0, 0, show.Water()})
}

// BeamEnd is where a laser beam from a place along a way stops: where it
// meets the water, or reach metres out.
func (show Layout) BeamEnd(origin, way Point, reach float64) Point {
	if way[2] < 0 {
		down := (origin[2] - show.Water()) / (0 - way[2])
		if down < reach {
			return origin.Add(way.Scale(down))
		}
	}
	return origin.Add(way.Scale(reach))
}

// StrobeFires is whether a strobe fires in a frame of lightshow's
// twelve a second (frame is the moment's seconds times twelve, rounded):
// a cue of one fires it, nought holds it dark, and below nought leaves
// it to its own luck, firing when its lattice number for the frame is
// under strobeLuck. The white strobe is seed 1, the magenta 2.
func StrobeFires(seed uint64, forced, frame int) bool {
	if forced >= 0 {
		return forced > 0
	}
	return Lattice(int64(frame), 0, seed) < strobeLuck
}

// StrobeFrame is the frame of lightshow's strobe clock a moment falls
// in.
func StrobeFrame(seconds float64) int {
	return int(math.Round(seconds * strobeRate))
}

// Lattice is a number from nought to one for a lattice point and a
// seed, the same every time: a hash, mixed twice.
func Lattice(x, y int64, seed uint64) float64 {
	h := uint64(x)*latticeX ^ uint64(y)*latticeY ^ seed*latticeSeed
	h ^= h >> latticeShift
	h *= latticeMix
	h ^= h >> latticeShift
	return float64(h>>latticeDrop) / float64(uint64(1)<<latticeBits)
}
