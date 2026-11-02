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
	if !(shear >= 0 && shear <= 360) || !(every > 0 && every <= 180) {
		return storms, fmt.Errorf("jets carrying %g degrees, turning "+
			"every %g", shear, every)
	}
	storms.shear, storms.every = shear, every
	return storms, nil
}

// jet is how many degrees of longitude the jets carry the air at a
// latitude in degrees.
func (storms Storms) jet(latitude float64) float64 {
	if storms.shear == 0 {
		return 0
	}
	return storms.shear * math.Sin(2*math.Pi*latitude/storms.every)
}

// sheared is where the air at a way was before the jets carried it: its
// longitude moved back by the jet at its latitude; but air a vortex
// holds rides whole, by the jet at the vortex's own latitude, so the
// vortex keeps its shape and only the air about it is smeared.
func (storms Storms) sheared(up maths.Vec, ways frame,
	marked marks) maths.Vec {
	if storms.shear == 0 {
		return up
	}
	latitude := latitudeOf(up, ways.pole)
	longitude := math.Atan2(up.Dot(ways.east), up.Dot(ways.front)) /
		maths.Degree
	carried := (1-marked.held)*storms.jet(latitude) +
		marked.held*storms.jet(marked.ridden)
	return ways.toward(latitude, longitude-carried)
}

// calmest is how turbulent the air is at a jet's core, where it runs fastest
// and smoothest, against one in the lanes where the jets turn about and rub
// past each other; and lanes, how sharply the turbulence gathers into those
// lanes: picked, after juno's pictures, laminar belts beside folded lanes.
const (
	calmest = 0.15
	lanes   = 4
)

// stilled is how much of the air's turbulence a vortex stills where it
// holds its air: picked, so its middle keeps the soft spiral grain
// juno's ovals show.
const stilled = 0.7

// rough is how turbulent the air is at a latitude in degrees, zero to one: one
// everywhere with no jets; with them, most where their shear is, the jets'
// slope across the latitudes, and calmest at their cores.
func (storms Storms) rough(latitude float64) float64 {
	if storms.shear == 0 {
		return 1
	}
	slope := math.Abs(math.Cos(2 * math.Pi * latitude / storms.every))
	return calmest + (1-calmest)*math.Pow(slope, lanes)
}

// mostOctaves is the most layers of turbulence, each half the size and
// half the strength of the one before, drawn where no pixel says how
// fine to stop, and the most drawn anywhere: picked.
const mostOctaves = 16

// reach is how far turbulence of strength one moves a latitude, in
// degrees: picked.
const reach = 12.0

// fold is how far each fold carries the air along its flow, in eddies;
// carrying, how many octaves the flow's stream function holds; and steps,
// how many small steps each fold takes along it: picked.
//
// one long jump through a random field tears neighbouring pixels apart
// into noise, as four eddies with every octave, three times, did; small
// steps along a smooth flow draw the air out into filaments instead.
const (
	fold     = 0.5
	carrying = 3
	steps    = 8
)

// averaging is how many eddies a pixel must be drawn across by the folds
// before half of what they move is kept and half taken as its mean, as
// the pixel's light would average filaments it cannot tell apart:
// picked.
const averaging = 2.0

// nudge is how far apart, in eddies, the stream function is read either
// side of a point to find its slope: picked, far below an eddy.
const nudge = 1e-3

// mostFolds is the most times the flow may fold back on itself: picked,
// so a set cannot ask a cost without end.
const mostFolds = 8

// NewStorms are storms from a seed, with a turbulence from zero, none, to one,
// latitudes moved by up to about twelve degrees, and eddies, how many fit
// around the planet.
//
// a stretch is how many times longer east to west than north to south the jets
// pull them; an edge, how many degrees either side of a band's edge its two
// columns blend; and folds, how many times the flow is folded back through
// itself, zero for none.
func NewStorms(seed uint64, turbulence, eddies, stretch, edge float64,
	folds int, spots []Spot) (Storms, error) {
	if !(turbulence >= 0 && turbulence <= 1) || !(eddies > 0) ||
		!(stretch >= 1) || !(edge >= 0) || folds < 0 || folds > mostFolds {
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
	front := eye.Sub(pole.Scale(eye.Dot(pole)))
	if front.Len() < 1e-9*eye.Len() || eye.Len() == 0 {
		front = maths.Upright(pole).X
	}
	front = front.Unit()
	return frame{pole: pole, front: front, east: pole.Cross(front).Unit()}
}

// toward is the unit way at a latitude and longitude in degrees.
func (ways frame) toward(latitude, longitude float64) maths.Vec {
	up, round := latitude*maths.Degree, longitude*maths.Degree
	level := ways.front.Scale(math.Cos(round)).Add(
		ways.east.Scale(math.Sin(round)))
	return level.Scale(math.Cos(up)).Add(ways.pole.Scale(math.Sin(up)))
}

// flowAt is a way as a point of the flow's space: on the sphere itself,
// so no seam runs down a longitude, an eddy to a unit, and finer along
// the pole so eddies stretch east to west.
func (storms Storms) flowAt(up maths.Vec, ways frame) maths.Vec {
	size := storms.eddies / (2 * math.Pi)
	return maths.V(size*up.Dot(ways.front),
		size*storms.stretch*up.Dot(ways.pole), size*up.Dot(ways.east))
}

// layered is gradient noise in layers at a point of the flow's space,
// from a seed: a layer for each halving of size, each half as strong,
// down to the finest a pixel there tells apart, in radians of the sphere.
//
// a layer between twice the finest and the finest is drawn in part, so
// none pops in. with no finest, every layer is drawn.
func (storms Storms) layered(seed uint64, p maths.Vec,
	finest float64) float64 {
	return storms.layers(seed, p, finest, mostOctaves)
}

// layers is layered noise of no more than so many octaves.
func (storms Storms) layers(seed uint64, p maths.Vec, finest float64,
	most int) float64 {
	sum, scale, strength := 0.0, 1.0, 1.0
	width := 2 * math.Pi / (storms.eddies * storms.stretch)
	for i := 0; i < most; i++ {
		seen := 1.0
		if finest > 0 {
			seen = math.Max(0, math.Min(1, width/finest-1))
		}
		if seen == 0 {
			break
		}
		sum += seen * strength * noise.Gradient(seed+uint64(i),
			scale*p.X, scale*p.Y, scale*p.Z)
		scale, strength, width = scale*2, strength/2, width/2
	}
	return sum
}

// flow is the way the air moves at a point of the flow's space, under a stream
// function from a seed: across the slope of the function and square to the way
// up, so it runs along the function's contours, keeps to the sphere, and
// neither gathers nor spreads, as bridson's curl noise does.
func (storms Storms) flow(seed uint64, p maths.Vec, finest float64) maths.Vec {
	stream := func(q maths.Vec) float64 {
		return storms.layers(seed, q, finest, carrying)
	}
	slope := maths.V(
		stream(p.Add(maths.V(nudge, 0, 0)))-stream(p.Sub(maths.V(nudge, 0, 0))),
		stream(p.Add(maths.V(0, nudge, 0)))-stream(p.Sub(maths.V(0, nudge, 0))),
		stream(p.Add(maths.V(0, 0, nudge)))-stream(p.Sub(maths.V(0, 0, nudge))))
	return p.Unit().Cross(slope.Scale(1 / (2 * nudge)))
}

// folded is where the air at a point of the flow came from: carried back
// through each fold in small steps along a flow of its own, so it is drawn out
// into filaments and wound into spirals. each fold's flow is another, so the
// air is mixed as a turbulence that changes does.
//
// it is also how many times over the folds stretched it, read from a second
// point a pixel's width beside it, carried alike. each step goes so far as the
// air's roughness there says, zero to one. modelled.
func (storms Storms) folded(p maths.Vec, finest, rough float64) (
	maths.Vec, float64) {
	apart := math.Max(finest*storms.eddies/(2*math.Pi), 1e-6)
	side := maths.V(0, 1, 0).Cross(p)
	if side.Len() == 0 {
		side = maths.V(1, 0, 0)
	}
	at, beside := p, p.Add(side.Unit().Scale(apart))
	stride := fold * rough / steps
	for level := 1; level <= storms.folds; level++ {
		seed := storms.seed + uint64(1000*level)
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
	rough := storms.rough(latitude) * (1 - stilled*marked.held)
	at, stretch := storms.folded(storms.flowAt(turned, ways), finest, rough)
	drawn := finest * math.Max(1, stretch)
	across := drawn * storms.eddies / (2 * math.Pi) / averaging
	kept := rough / (1 + across*across)
	pushed := storms.turbulence * reach * kept * storms.layered(storms.seed,
		at, drawn)
	lift = kept * math.Max(-1, math.Min(1, storms.layers(storms.seed+300,
		at, drawn, carrying)))
	return latitude + pushed, lift, marked
}
