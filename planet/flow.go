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
// marking how much more spin brings it in full. all three are picked; at
// a tenth, turbulence crossed marked nearly everywhere.
const (
	lifting = 2.0
	marked  = 0.3
	marking = 0.3
)

// fading and walled are the latitudes, in degrees, past which a flow
// fades into the plain bands, and where it is gone: a flow's grid stops
// at its walls, and read past them its last row fans out to the pole.
// picked, inside the weather's walls at 80 degrees.
const (
	fading = 70.0
	walled = 78.0
)

// flowed is the latitude in degrees whose air a way of the face shows,
// how its air lifts there, and the marks its spin leaves, read from the
// flow. a cyclone, spinning the planet's way for its hemisphere, sinks
// and clears; an anticyclone lifts its air and caps it, as on jupiter.
func (face *Face) flowed(up maths.Vec) (latitude, lift float64, made marks) {
	latitude = latitudeOf(up, face.ways.pole)
	longitude := math.Atan2(up.Dot(face.ways.east), up.Dot(face.ways.front))
	if longitude < 0 {
		longitude += 2 * math.Pi
	}
	origin, spin, age := face.flow.At(latitude*maths.Degree, longitude)
	gone := math.Max(0, math.Min(1, (math.Abs(latitude)-fading)/
		(walled-fading)))
	origin += gone * (latitude*maths.Degree - origin)
	spin *= 1 - gone
	made.age = age
	cyclonic := spin
	if latitude < 0 {
		cyclonic = -spin
	}
	lift = math.Max(-1, math.Min(1, -cyclonic*lifting))
	made.capped = math.Max(0, math.Min(1, (-cyclonic-marked)/marking))
	made.cleared = math.Max(0, math.Min(1, (cyclonic-marked)/marking))
	return origin / maths.Degree, lift, made
}
