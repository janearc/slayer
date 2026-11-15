package section

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// A placed lamp is one that says where it stands and the most light it
// sends any point a distance from there, whichever way the point lies.
// Only such a lamp's change can be bounded over a whole volume: a lamp
// asked at a few points says nothing of a narrow beam between them, nor
// of itself where it stands inside the volume.
type placed interface {
	Stands() maths.Vec
	AtMost(distance float64) float64
}

// lights is whether a steady volume, by its box as placed, is lit at the
// second moment as it was at the first, to within the drift allowed: the
// lamps that changed bring it no more than the drift's share of what the
// steady lamps bring, and nothing that changed stands where it could
// shadow the volume from a steady lamp. With no drift allowed, no.
//
// What a changed lamp brings is the most it sends the point of the box
// nearest it. A changed lamp that does not say where it stands, or that
// stands in the box, brings more than any drift allows.
func (change Change) lights(box maths.Box, where maths.Subspace) bool {
	centre, radius := ballOf(box, where)
	if !positive(change.drift) || unbounded(radius) {
		return false
	}
	var changed, steady float64
	for _, lamp := range change.unlit {
		changed += mostOn(lamp, box, where)
	}
	for _, lamp := range change.lit {
		steady += least(lamp, corners(box, where))
		if change.shadowed(lamp, centre, radius) {
			return false
		}
	}
	return changed <= change.drift*steady
}

// mostOn is the most light a lamp sends any point of a placed box: what
// it says it sends at most at the distance of the box's nearest point,
// and more than any number for a lamp that does not say or is in the box.
func mostOn(lamp thing.Lamp, box maths.Box, where maths.Subspace) float64 {
	known, says := lamp.(placed)
	if !says {
		return math.Inf(one)
	}
	local := where.In(known.Stands())
	near := where.Out(maths.V(
		math.Max(box.Min.X, math.Min(box.Max.X, local.X)),
		math.Max(box.Min.Y, math.Min(box.Max.Y, local.Y)),
		math.Max(box.Min.Z, math.Min(box.Max.Z, local.Z))))
	far := near.Sub(known.Stands()).Len()
	if !positive(far) {
		return math.Inf(one)
	}
	return known.AtMost(far)
}

// corners are the eight corners of a box as placed, and its middle.
func corners(box maths.Box, placed maths.Subspace) []maths.Vec {
	sum := box.Min.Add(box.Max)
	points := []maths.Vec{placed.Out(sum.Scale(half))}
	for at := range cornerCount {
		right, above, behind := has(at, xBit), has(at, yBit), has(at, zBit)
		corner := maths.V(pick(right, box.Min.X, box.Max.X),
			pick(above, box.Min.Y, box.Max.Y),
			pick(behind, box.Min.Z, box.Max.Z))
		points = append(points, placed.Out(corner))
	}
	return points
}

// cornerCount is defined as how many corners a box has: two ends on
// each of three axes, two times two times two.
const cornerCount = two * two * two

// The bits of a corner's number that say which end of each axis it is
// at, each defined as the place value of its axis in binary: one, two and
// four.
const (
	xBit = one
	yBit = two
	zBit = two * two
)

// has is whether a corner's number has a bit set.
func has(corner, bit int) bool { return corner&bit != nothing }

// positive is whether an amount is above none.
func positive(amount float64) bool { return amount > nothing }

// clamp is an amount held between a low and a high.
func clamp(low, high, amount float64) float64 {
	return math.Max(low, math.Min(high, amount))
}

// plus is defined as the sign that math.Inf takes for positive infinity.
const plus = one

// pick is the second of two numbers where a thing holds, and the first
// where it does not.
func pick(holds bool, first, second float64) float64 {
	if holds {
		return second
	}
	return first
}

// brightness is how bright a lamp's light is at a point, on a surface
// that faces it: the largest of the light's three.
func brightness(lamp thing.Lamp, point maths.Vec) float64 {
	toward, _, _ := lamp.Light(point, nowhere)
	_, _, light := lamp.Light(point, toward)
	return math.Max(light.R, math.Max(light.G, light.B))
}

// least is the dimmest a lamp is at any of some points.
func least(lamp thing.Lamp, points []maths.Vec) float64 {
	found := math.Inf(plus)
	for _, point := range points {
		found = math.Min(found, brightness(lamp, point))
	}
	return found
}

// shadowed is whether anything that changed stands where it could shadow
// a ball from a steady lamp: within the ball's width of the line from the
// ball's middle to the lamp, on the lamp's side. A lamp that gives no one
// direction casts no shadow the tracer tests.
func (change Change) shadowed(lamp thing.Lamp, centre maths.Vec,
	radius float64) bool {
	toward, far, _ := lamp.Light(centre, nowhere)
	if toward == nowhere {
		return false
	}
	for _, mover := range change.movers {
		if unbounded(mover.radius) {
			return true
		}
		off := mover.centre.Sub(centre)
		along := clamp(nothing, far, off.Dot(toward))
		drop := toward.Scale(along)
		if off.Sub(drop).Len() <= radius+mover.radius {
			return true
		}
	}
	return false
}
