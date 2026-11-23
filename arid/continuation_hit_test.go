package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The numbers the flat-mean hit tests are held to, each a choice of the test.
const (
	// hitStart is how far from the middle the grazing rays start, on the side
	// they come from, inside the core's square.
	hitStart = -3.9
	// hitAbove is how far over the surface at its start a grazing ray starts.
	hitAbove = 0.1
	// hitSteps is how many places before a hit the ray is checked to be over
	// the surface.
	hitSteps = 200
	// hitOn is how far off the surface the point of a hit may be.
	hitOn = 1e-9
)

// hitDegrees are the angles below level the grazing rays are drawn at.
var hitDegrees = []float64{5, 2, 1, 0.5}

// climbDegrees are the angles above level the climbing rays are drawn at,
// each steeper than the slope of the core they leave.
var climbDegrees = []float64{5, 3}

// grazingAcross is a ray along x from over the surface at its start, the
// given degrees below level, and the height of the surface there.
func grazingAcross(water *FlatMeanContinuation, degrees float64) maths.Ray {
	height, _, _ := water.sample(hitStart, 0)
	radians := degrees * math.Pi / 180
	return maths.Ray{At: maths.V(hitStart, height+hitAbove, 0),
		Dir: maths.V(math.Cos(radians), -math.Sin(radians), 0)}
}

// A ray at a shallow angle over a continued core meets it, on the surface and
// with the ray over the surface until it does, at five degrees down to half a
// degree.
func TestFlatMeanContinuationHitMeetsGrazingRays(t *testing.T) {
	water, err := NewFlatMeanContinuation(continuationCore(t, false), 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, degrees := range hitDegrees {
		ray := grazingAcross(water, degrees)
		along, hit := water.Hit(ray, 0, math.MaxFloat64)
		if !hit {
			t.Errorf("%g degrees was not met", degrees)
			continue
		}
		at := ray.Point(along)
		height, _, _ := water.sample(at.X, at.Z)
		if math.Abs(at.Y-height) > hitOn {
			t.Errorf("%g degrees met %g off the surface", degrees, at.Y-height)
		}
		for step := range hitSteps {
			before := ray.Point(along * float64(step) / hitSteps)
			floor, _, _ := water.sample(before.X, before.Z)
			if before.Y < floor {
				t.Errorf("%g degrees was under the surface before its hit",
					degrees)
				break
			}
		}
	}
}

// The anti-test of the one above: a ray that climbs, a ray whose hit would be
// on the mean plane past the collar, and a ray from beyond the square, are not
// met, since the surface ends at its square.
func TestFlatMeanContinuationHitDoesNotMeetWhatIsNotThere(t *testing.T) {
	water, err := NewFlatMeanContinuation(continuationCore(t, false), 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, degrees := range climbDegrees {
		if _, hit := water.Hit(grazingAcross(water, -degrees), 0,
			math.MaxFloat64); hit {
			t.Errorf("%g degrees climbing met the surface", degrees)
		}
	}
	past := maths.Ray{At: maths.V(water.outerHalf-0.5, 1, 0),
		Dir: maths.V(math.Cos(math.Pi/180), -math.Sin(math.Pi/180), 0)}
	if along, hit := water.Hit(past, 0, math.MaxFloat64); hit {
		t.Errorf("a ray to the plane past the collar met it at %g", along)
	}
	outside := maths.Ray{At: maths.V(water.outerHalf+1, 1, 0),
		Dir: maths.V(0, -1, 0)}
	if along, hit := water.Hit(outside, 0, math.MaxFloat64); hit {
		t.Errorf("a ray from beyond the square met it at %g", along)
	}
}
