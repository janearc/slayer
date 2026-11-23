package planet

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Storms are how a planet's bands are torn and mixed: modelled, not a
// simulation of its air. a point of the face looks up its column at a
// latitude the flow has moved: pushed by turbulence, swirled by spots,
// and blended with the next zone across a band's edge.
type Storms struct {
	seed               uint64
	turbulence, eddies float64
	stretch, edge      float64
	folds              int
	spots              []Spot
	shear, every       float64
}

// Jetted is the storms under jets that run east and west by turns, band
// by band: the air at a latitude carried so many degrees of longitude
// along, its way changing every so many degrees of latitude, as a
// planet turning fast drives its jets against each other.
//
// what the spots wind and the folds draw is smeared along them. modelled: a
// sine across the latitudes, not the jets' measured speeds.
func (storms Storms) Jetted(shear, every float64) (Storms, error) {
	if !(shear >= nothing && shear <= fullTurnDegrees) ||
		!(every > nothing && every <= halfTurnDegrees) {
		return storms, fmt.Errorf("jets carrying %g degrees, turning "+
			"every %g", shear, every)
	}
	storms.shear, storms.every = shear, every
	return storms, nil
}

// jet is how many degrees of longitude the jets carry the air at a
// latitude in degrees.
func (storms Storms) jet(latitude float64) float64 {
	if storms.shear == nothing {
		return nothing
	}
	return storms.shear * math.Sin(circle*latitude/storms.every)
}

// sheared is where the air at a way was before the jets carried it: its
// longitude moved back by the jet at its latitude; but air a vortex
// holds rides whole, by the jet at the vortex's own latitude, so the
// vortex keeps its shape and only the air about it is smeared.
func (storms Storms) sheared(up maths.Vec, ways frame,
	marked marks) maths.Vec {
	if storms.shear == nothing {
		return up
	}
	latitude := latitudeOf(up, ways.pole)
	longitude := math.Atan2(up.Dot(ways.east), up.Dot(ways.front)) /
		maths.Degree
	carried := (whole-marked.held)*storms.jet(latitude) +
		marked.held*storms.jet(marked.ridden)
	return ways.toward(latitude, longitude-carried)
}

// calmest is how turbulent the air is at a jet's core, where it runs fastest
// and smoothest, against one in the lanes where the jets turn about and rub
// past each other; and lanes, how sharply the turbulence gathers into those
// lanes. both are in picks.yaml.
var (
	calmest = pick("jet-core-turbulence")
	lanes   = pick("jet-lane-sharpness")
)

// stilled is how much of the air's turbulence a vortex stills where it
// holds its air, from picks.yaml.
var stilled = pick("vortex-stilling")

// rough is how turbulent the air is at a latitude in degrees, zero to one: one
// everywhere with no jets; with them, most where their shear is, the jets'
// slope across the latitudes, and calmest at their cores.
func (storms Storms) rough(latitude float64) float64 {
	if storms.shear == nothing {
		return whole
	}
	slope := math.Abs(math.Cos(circle * latitude / storms.every))
	return calmest + (whole-calmest)*math.Pow(slope, lanes)
}

// mostOctaves is the most layers of turbulence, each half the size and
// half the strength of the one before, drawn where no pixel says how
// fine to stop, and the most drawn anywhere; its value is in picks.yaml.
var mostOctaves = pickCount("turbulence-most-octaves")

// reach is how far turbulence of strength one moves a latitude, in
// degrees, from picks.yaml.
var reach = pick("turbulence-reach")

// fold is how far each fold carries the air along its flow, in eddies;
// carrying, how many octaves the flow's stream function holds; and steps,
// how many small steps each fold takes along it. all three are in
// picks.yaml.
//
// one long jump through a random field tears neighbouring pixels apart
// into noise, as four eddies with every octave, three times, did; small
// steps along a smooth flow draw the air out into filaments instead.
var (
	fold     = pick("fold-reach")
	carrying = pickCount("fold-stream-octaves")
	steps    = pickCount("fold-steps")
)

// averaging is how many eddies a pixel must be drawn across by the folds
// before half of what they move is kept and half taken as its mean, as
// the pixel's light would average filaments it cannot tell apart; its
// value is in picks.yaml.
var averaging = pick("fold-averaging")

// nudge is how far apart, in eddies, the stream function is read either
// side of a point to find its slope, far below an eddy; its value is in
// picks.yaml.
var nudge = pick("stream-nudge")

// mostFolds is the most times the flow may fold back on itself, so a set
// cannot ask a cost without end; its value is in picks.yaml.
var mostFolds = pickCount("fold-most")

// foldSeedStride is how far apart the seeds of successive folds are, and
// liftSeedOffset how far the seed of the lift is from the turbulence's, so
// none of them repeat another's noise; both are in picks.yaml.
var (
	foldSeedStride = pickCount("fold-seed-stride")
	liftSeedOffset = uint64(pickCount("lift-seed-offset"))
)

// leastApart is the least distance, in eddies, the second point a fold is
// read at stands from the first, so the stretch it reads is never of two
// points on one another; its value is in picks.yaml.
var leastApart = pick("fold-least-apart")

// frameFloor is how short the way toward the eye may be, over the eye's
// distance, once the pole's part of it is taken out, before the frame takes
// a front of its own; its value is in picks.yaml.
var frameFloor = pick("frame-front-floor")

// NewStorms are storms from a seed, with a turbulence from zero, none, to one,
// latitudes moved by up to about twelve degrees, and eddies, how many fit
// around the planet.
//
// a stretch is how many times longer east to west than north to south the jets
// pull them; an edge, how many degrees either side of a band's edge its two
// columns blend; and folds, how many times the flow is folded back through
// itself, zero for none. eddies, stretch and edge may not be endless.
func NewStorms(seed uint64, turbulence, eddies, stretch, edge float64,
	folds int, spots []Spot) (Storms, error) {
	if !(turbulence >= nothing && turbulence <= whole) ||
		!(eddies > nothing) || !(stretch >= whole) || !(edge >= nothing) ||
		endless(eddies) || endless(stretch) || endless(edge) ||
		folds < nothing || folds > mostFolds {
		return Storms{}, fmt.Errorf("storms of turbulence %g, eddies %g, "+
			"stretch %g, edge %g, folds %d", turbulence, eddies, stretch,
			edge, folds)
	}
	return Storms{seed: seed, turbulence: turbulence, eddies: eddies,
		stretch: stretch, edge: edge, folds: folds, spots: spots}, nil
}

// a frame is a planet's own ways: its pole, the way toward the eye at longitude
// zero, and east of it.
type frame struct{ pole, front, east maths.Vec }

// frameOf is the frame for a pole and an eye in the body's space.
func frameOf(pole, eye maths.Vec) frame {
	along := eye.Dot(pole)
	front := eye.Sub(pole.Scale(along))
	if front.Len() < frameFloor*eye.Len() || eye.Len() == nothing {
		front = maths.Upright(pole).X
	}
	front = front.Unit()
	return frame{pole: pole, front: front, east: pole.Cross(front).Unit()}
}

// toward is the unit way at a latitude and longitude in degrees.
func (ways frame) toward(latitude, longitude float64) maths.Vec {
	up, round := latitude*maths.Degree, longitude*maths.Degree
	along := ways.front.Scale(math.Cos(round))
	across := ways.east.Scale(math.Sin(round))
	level := along.Add(across)
	flat := level.Scale(math.Cos(up))
	raised := ways.pole.Scale(math.Sin(up))
	return flat.Add(raised)
}

// flowAt is a way as a point of the flow's space: on the sphere itself,
// so no seam runs down a longitude, an eddy to a unit, and finer along
// the pole so eddies stretch east to west.
func (storms Storms) flowAt(up maths.Vec, ways frame) maths.Vec {
	size := storms.eddies / circle
	return maths.V(size*up.Dot(ways.front),
		size*storms.stretch*up.Dot(ways.pole), size*up.Dot(ways.east))
}

// layered is gradient noise in layers at a point of the flow's space,
// from a seed: a layer for each halving of size, each half as strong,
// down to the finest a pixel there tells apart, in radians of the sphere.
//
// a layer between twice the finest and the finest is drawn in part, so
// none pops in. with no finest, every layer is drawn.
func (storms Storms) layered(seed uint64, point maths.Vec,
	finest float64) float64 {
	return storms.layers(seed, point, finest, mostOctaves)
}

// layers is layered noise of no more than so many octaves.
func (storms Storms) layers(seed uint64, point maths.Vec, finest float64,
	most int) float64 {
	var sum float64
	var scale, strength float64 = whole, whole
	width := circle / (storms.eddies * storms.stretch)
	for i := nothing; i < most; i++ {
		var seen float64 = whole
		if finest > nothing {
			seen = math.Max(nothing, math.Min(whole, width/finest-whole))
		}
		if seen == nothing {
			break
		}
		sum += seen * strength * noise.Gradient(seed+uint64(i),
			scale*point.X, scale*point.Y, scale*point.Z)
		scale, strength, width = scale*double, strength/double, width/double
	}
	return sum
}

// across is how much a stream function changes from one side of a point to
// the other, along a step either way from it.
func (storms Storms) across(seed uint64, point, step maths.Vec,
	finest float64) float64 {
	forward := storms.layers(seed, point.Add(step), finest, carrying)
	backward := storms.layers(seed, point.Sub(step), finest, carrying)
	return forward - backward
}

// flow is the way the air moves at a point of the flow's space, under a stream
// function from a seed: across the slope of the function and square to the way
// up, so it runs along the function's contours, keeps to the sphere, and
// neither gathers nor spreads, as bridson's curl noise does.
func (storms Storms) flow(seed uint64, point maths.Vec,
	finest float64) maths.Vec {
	stepX := maths.V(nudge, nothing, nothing)
	stepY := maths.V(nothing, nudge, nothing)
	stepZ := maths.V(nothing, nothing, nudge)
	slope := maths.V(storms.across(seed, point, stepX, finest),
		storms.across(seed, point, stepY, finest),
		storms.across(seed, point, stepZ, finest))
	return point.Unit().Cross(slope.Scale(whole / (double * nudge)))
}

// folded is where the air at a point of the flow came from: carried back
// through each fold in small steps along a flow of its own, so it is drawn out
// into filaments and wound into spirals. each fold's flow is another, so the
// air is mixed as a turbulence that changes does.
//
// it is also how many times over the folds stretched it, read from a second
// point a pixel's width beside it, carried alike. each step goes so far as the
// air's roughness there says, zero to one. modelled.
func (storms Storms) folded(point maths.Vec, finest, rough float64) (
	maths.Vec, float64) {
	apart := math.Max(finest*storms.eddies/circle, leastApart)
	side := maths.V(nothing, whole, nothing).Cross(point)
	if side.Len() == nothing {
		side = maths.V(whole, nothing, nothing)
	}
	at, beside := point, point.Add(side.Unit().Scale(apart))
	stride := fold * rough / float64(steps)
	for level := adjacent; level <= storms.folds; level++ {
		seed := storms.seed + uint64(foldSeedStride*level)
		for range steps {
			at = at.Add(storms.flow(seed, at, finest).Scale(stride))
			beside = beside.Add(storms.flow(seed, beside, finest).Scale(
				stride))
		}
	}
	return at, beside.Sub(at).Len() / apart
}

// moved is the latitude, in degrees, whose air a point of the face shows, how
// its air lifts there, from minus one, sinking, to one, rising, and the marks
// its vortices leave.
//
// its way is swirled by the spots, carried back along the jets, carried through
// the folds, then pushed by the turbulence there; the lift is read from the
// same folded flow, so it follows the filaments the folds draw.
//
// how rough the air is goes by the lanes, and is calm where a vortex holds it.
// finest is the finest a pixel there tells apart, in radians, zero for every
// layer.
func (storms Storms) moved(up maths.Vec, ways frame, whirls []vortex,
	finest float64) (latitude, lift float64, marked marks) {
	turned, marked := swirled(up, whirls)
	turned = storms.sheared(turned, ways, marked)
	latitude = latitudeOf(turned, ways.pole)
	rough := storms.rough(latitude) * (whole - stilled*marked.held)
	at, stretch := storms.folded(storms.flowAt(turned, ways), finest, rough)
	drawn := finest * math.Max(whole, stretch)
	across := drawn * storms.eddies / circle / averaging
	kept := rough / (whole + across*across)
	pushed := storms.turbulence * reach * kept * storms.layered(storms.seed,
		at, drawn)
	raised := storms.layers(storms.seed+liftSeedOffset, at, drawn, carrying)
	lift = kept * math.Max(-whole, math.Min(whole, raised))
	return latitude + pushed, lift, marked
}
