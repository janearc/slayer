package planet

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// A Flow is weather a face reads its air from, as a fluid made it.
//
// at a latitude and longitude in radians, longitude zero toward the eye, it
// gives the latitude its air came from, in radians; its vorticity over twice
// the planet's spin; and how many seconds the air has cooked at the top.
type Flow interface {
	At(latitude, longitude float64) (origin, spin, age float64)
}

// Flowing is the body with its bands' air read from a flow, and its
// vortices' marks from the flow's own spin, in place of its storms'
// folds and spots.
func (body Body) Flowing(flow Flow) Body {
	body.flow = flow
	return body
}

// lifting is how far a vortex's spin, against twice the planet's, lifts
// or sinks its air, from minus one to one, per unit of spin.
//
// marked is the spin past which a vortex shows a cap or a clearing, and
// marking how much more spin brings it in full. all three are in
// picks.yaml.
var (
	lifting = pick("flow-lift")
	marked  = pick("flow-mark-start")
	marking = pick("flow-mark-span")
)

// fading and walled are the latitudes, in degrees, past which a flow
// fades into the plain bands, and where it is gone: a flow's grid stops
// at its walls, and read past them its last row fans out to the pole.
var (
	fading = pick("flow-fade-start")
	walled = pick("flow-wall")
)

// flowed is the latitude in degrees whose air a way of the face shows,
// how its air lifts there, and the marks its spin leaves, read from the
// flow. a cyclone, spinning the planet's way for its hemisphere, sinks
// and clears; an anticyclone lifts its air and caps it, as on a gas giant.
func (face *Face) flowed(up maths.Vec) (latitude, lift float64, made marks) {
	latitude = latitudeOf(up, face.ways.pole)
	longitude := math.Atan2(up.Dot(face.ways.east), up.Dot(face.ways.front))
	if longitude < nothing {
		longitude += circle
	}
	origin, spin, age := face.flow.At(latitude*maths.Degree, longitude)
	into := (math.Abs(latitude) - fading) / (walled - fading)
	gone := math.Max(nothing, math.Min(whole, into))
	origin += gone * (latitude*maths.Degree - origin)
	spin *= whole - gone
	made.age = age
	cyclonic := spin
	if latitude < nothing {
		cyclonic = -spin
	}
	lifted := -cyclonic * lifting
	lift = math.Max(-whole, math.Min(whole, lifted))
	made.capped = math.Max(nothing, math.Min(whole,
		(-cyclonic-marked)/marking))
	made.cleared = math.Max(nothing, math.Min(whole,
		(cyclonic-marked)/marking))
	return origin / maths.Degree, lift, made
}
