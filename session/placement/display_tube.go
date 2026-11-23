package placement

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// These bound how a pick ray finds the wall of a tube.
const (
	// tubeMarchSteps is the most steps a ray takes toward a tube's wall
	// before it is called a miss. A ray that grazes the wall needs many.
	tubeMarchSteps = 256
	// tubeReachShare is how near the wall a step must come, as a share of
	// the wall's thickness, for the ray to be said to touch it.
	tubeReachShare = 0.01
)

// tubeHitter answers a ray for a tube. A tube has no exact hit, so the ray
// is marched to the wall by the tube's own distance, as the tracer does.
// It exists so that a tube, which the display can draw, can also be picked.
type tubeHitter struct{ tube shape.Tube }

// Hit marches the ray from near toward the wall, a step at a time as far as
// the wall is, and gives the distance at which it comes within reach of it.
// It gives far and false when the ray passes the wall, outruns far or takes
// all its steps without arriving.
func (marched tubeHitter) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	length := ray.Dir.Len()
	if length == 0 {
		return far, false
	}
	reach := marched.tube.Thick * tubeReachShare
	distance := math.Max(near, 0)
	for range tubeMarchSteps {
		if distance > far {
			return far, false
		}
		gap := math.Abs(marched.tube.Distance(ray.Point(distance)))
		if gap <= reach {
			return distance, true
		}
		distance += gap / length
	}
	return far, false
}

// displayHitter is how a placed shape is picked: by its own exact hit when
// it has one, and by a marched hit for a tube.
func displayHitter(solid shape.Shape) (shape.Hitter, bool) {
	if tube, isTube := solid.(shape.Tube); isTube {
		return tubeHitter{tube: tube}, true
	}
	hitter, ok := solid.(shape.Hitter)
	return hitter, ok
}
