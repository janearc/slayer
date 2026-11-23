package planet

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The numbers the package counts and measures in that are true by
// definition. The ones that count a share or an amount, nothing, whole,
// double, circle and rightAngle, are in laid.go beside what they count.
const (
	// half is defined as half of a whole: one half.
	half = 0.5
	// adjacent is defined as the distance between neighbours in a list or
	// a row of cells: one.
	adjacent = 1
	// absent is defined as the place of a column that is not there: minus
	// one, which no list has.
	absent = -1
	// axes is defined as how many axes space has: three.
	axes = 3
	// poleLatitude is defined as the latitude of a pole, in degrees.
	poleLatitude = 90.0
	// halfTurnDegrees is defined as the degrees in half a turn.
	halfTurnDegrees = 180.0
	// fullTurnDegrees is defined as the degrees in a full turn.
	fullTurnDegrees = 360.0
	// sphere is defined as the steradians in a sphere: four pi.
	sphere = 4 * math.Pi
)

// noLight is the light of nothing: no red, no green and no blue.
var noLight maths.Light
