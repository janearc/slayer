package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The numbers the tests of a height field met by the tracer are held to, each
// a choice of the test.
const (
	// seaAcross is how wide the level sea is, in its own units.
	seaAcross = 4096.0
	// seaAbove is how high above the sea the rays start, in the world.
	seaAbove = 2.0
	// seaBack is how far south of the sea's middle the rays start, in the
	// world.
	seaBack = 1900.0
	// seaShare is how far a hit may be from the true distance, as a share of
	// it, for rounding.
	seaShare = 1e-9
)

// seaDegrees are the angles below level the tests draw at.
var seaDegrees = []float64{5, 2, 1, 0.1}

// levelSea is a placed level height field, with the placement the world gave
// it: where it stands and how large its units are.
func levelSea(t *testing.T, origin maths.Vec, scale float64) placedThing {
	t.Helper()
	field, err := shape.NewHeightField(make([]float64, 4), 2, seaAcross)
	if err != nil {
		t.Fatal(err)
	}
	return placedThing{thing: &thing.Thing{Shape: field, Material: water()},
		placement: placement{at: maths.NewSubspace(origin, maths.Identity,
			scale), scale: scale}}
}

// glance is a ray in the world from a start, toward positive z, the given
// degrees below level.
func glance(at maths.Vec, degrees float64) maths.Ray {
	radians := degrees * math.Pi / 180
	return maths.Ray{At: at, Dir: maths.V(0, -math.Sin(radians),
		math.Cos(radians))}
}

// A placed height field is met by the tracer at the distance geometry says,
// at five, two, one and a tenth of a degree, wherever it is placed and
// however large its units, where marching its distance runs out of steps.
func TestTheTracerMeetsAPlacedHeightFieldAtGrazingAngles(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		origin := maths.V(10, 3, -5)
		sea := levelSea(t, origin, scale)
		for _, degrees := range seaDegrees {
			start := origin.Add(maths.V(0, seaAbove, -seaBack*scale))
			want := seaAbove / math.Sin(degrees*math.Pi/180)
			got, hit := sea.meet(glance(start, degrees), 0, nowhere)
			if !hit || math.Abs(got-want) > seaShare*want {
				t.Errorf("scale %g, %g degrees: met at %g, %t; want %g",
					scale, degrees, got, hit, want)
			}
		}
	}
}

// The anti-test of the one above: a ray that climbs, a ray cut short of the
// sea, and a ray over a sea too narrow to hold its hit, are not met.
func TestTheTracerDoesNotMeetAPlacedHeightFieldItDoesNotReach(t *testing.T) {
	origin := maths.V(10, 3, -5)
	sea := levelSea(t, origin, 1)
	for _, degrees := range seaDegrees {
		start := origin.Add(maths.V(0, seaAbove, -seaBack))
		want := seaAbove / math.Sin(degrees*math.Pi/180)
		if _, hit := sea.meet(glance(start, -degrees), 0, nowhere); hit {
			t.Errorf("%g degrees up met the sea", degrees)
		}
		if _, hit := sea.meet(glance(start, degrees), 0,
			want*(1-1e3*seaShare)); hit {
			t.Errorf("%g degrees met the sea short of it", degrees)
		}
		near := origin.Add(maths.V(0, seaAbove, -seaAcross/4))
		if _, hit := sea.meet(glance(near, 0.01), 0, nowhere); hit {
			t.Errorf("a ray reaching the sea past its edge met it")
		}
	}
}

// A ray lifted off the sea by the tracer's own lift leaves it: going away it
// is not met, from either side, and aimed back it is met at the lift over the
// sine of its angle.
func TestTheTracerDoesNotMeetTheHeightFieldARayLeaves(t *testing.T) {
	origin := maths.V(10, 3, -5)
	sea := levelSea(t, origin, 1)
	away := lift * seaAcross
	for _, degrees := range []float64{45, 5, 1} {
		for _, side := range []float64{1, -1} {
			start := origin.Add(maths.V(0, side*away, 0))
			if _, hit := sea.meet(glance(start, -side*degrees), 0,
				nowhere); hit {
				t.Errorf("%g degrees leaving from side %g met the sea", degrees,
					side)
			}
			want := away / math.Sin(degrees*math.Pi/180)
			got, hit := sea.meet(glance(start, side*degrees), 0, nowhere)
			if !hit || math.Abs(got-want) > seaShare*want {
				t.Errorf("%g degrees aimed back from side %g met at %g, %t; "+
					"want %g", degrees, side, got, hit, want)
			}
		}
	}
}
