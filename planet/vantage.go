package planet

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// a vantage is where a ring is seen from: an eye, a point in the ring's
// own space, its plane across x and z and its pole up y, and how wide
// one ray from it is, in radians. the zero vantage is no eye, and every
// grain is drawn.
type vantage struct {
	eye maths.Vec
	ray float64
}

// mostStretch is the most a ray's footprint is drawn out across a ring
// seen nearly edge on, so a ray grazing the plane still sees its largest
// clumps: picked.
const mostStretch = 100.0

// finest is the finest a ray from the eye tells apart at a point of the ring's
// plane, in metres, across the orbit, the way the ring's clumps are narrow: the
// ray's width at that distance, stretched as the plane slants away from it
// along the reach. zero with no eye.
func (from vantage) finest(x, z float64) float64 {
	point := maths.V(x, 0, z)
	toward := point.Sub(from.eye)
	distance, reach := toward.Len(), point.Len()
	if from.ray == 0 || distance == 0 || reach == 0 {
		return 0
	}
	slant := point.Scale(1 / reach).Dot(toward.Scale(1 / distance))
	squash := math.Max(1/mostStretch, math.Sqrt(math.Max(0, 1-slant*slant)))
	return from.ray * distance / squash
}

// Seen is the body with its rings seen from an eye, a point in their own
// space, through rays so many radians wide: their grains finer than a
// ray tells apart are drawn as the mean of what it covers, as a pixel
// takes them.
func (body Body) Seen(eye maths.Vec, ray float64) (Body, error) {
	if !(ray >= 0) || math.IsInf(ray, 0) || !finite(eye) {
		return body, fmt.Errorf("rings seen from %v through rays %g wide",
			eye, ray)
	}
	body.rings = slices.Clone(body.rings)
	for i := range body.rings {
		body.rings[i].from = vantage{eye: eye, ray: ray}
	}
	return body, nil
}

// finite is whether each of a point's coordinates is a number and not
// endless.
func finite(p maths.Vec) bool {
	for _, each := range []float64{p.X, p.Y, p.Z} {
		if math.IsNaN(each) || math.IsInf(each, 0) {
			return false
		}
	}
	return true
}
