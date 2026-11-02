package globe

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/janearc/slayer/maths"
)

// earth is a globe of the earth's mean radius, 6371 km.
const earth = 6371000.0

// mustGlobe is a globe of a radius, or the test fails.
func mustGlobe(t *testing.T, radius float64) Globe {
	t.Helper()
	made, err := New(radius)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// below is a way a turn of so many radians below level, looking along
// x from over the globe's top.
func below(radians float64) maths.Vec {
	return maths.V(math.Cos(radians), -math.Sin(radians), 0)
}

// from 100 km up the horizon lies arccos(r / (r + h)) below level: a
// ray a millionth of a radian under it meets the globe, and one a
// millionth over it does not.
func TestTheHorizonDipsAsGeometrySays(t *testing.T) {
	ball := mustGlobe(t, earth)
	height := 100000.0
	eye := maths.V(0, earth+height, 0)
	dip := math.Acos(earth / (earth + height))
	if _, met := ball.Hit(maths.Ray{At: eye, Dir: below(dip + 1e-6)},
		0, math.Inf(1)); !met {
		t.Errorf("a ray under the horizon misses")
	}
	if _, met := ball.Hit(maths.Ray{At: eye, Dir: below(dip - 1e-6)},
		0, math.Inf(1)); met {
		t.Errorf("a ray over the horizon meets the globe")
	}
}

// straight down from twelve metres the ground is twelve metres off.
func TestStraightDownIsTheHeight(t *testing.T) {
	ball := mustGlobe(t, earth)
	along, met := ball.Hit(maths.Ray{At: maths.V(0, earth+12, 0),
		Dir: maths.V(0, -1, 0)}, 0, math.Inf(1))
	if !met || math.Abs(along-12) > 1e-6 {
		t.Errorf("straight down from 12 m the ground is %v m off", along)
	}
}

// exact is where a ray from a height, a turn below level, meets a
// globe, worked in 200-bit arithmetic: the nearer root of t^2 + 2bt +
// c, with b the ray's start along its way and c its start's distance
// squared less the radius squared.
func exact(radius, height, radians float64) float64 {
	precise := func(value float64) *big.Float {
		return new(big.Float).SetPrec(200).SetFloat64(value)
	}
	start := precise(radius + height)
	along := new(big.Float).Mul(start, precise(-math.Sin(radians)))
	squared := new(big.Float).Mul(start, start)
	gap := new(big.Float).Sub(squared, new(big.Float).Mul(precise(radius),
		precise(radius)))
	inside := new(big.Float).Sub(new(big.Float).Mul(along, along), gap)
	root := new(big.Float).Sqrt(inside)
	near := new(big.Float).Sub(new(big.Float).Neg(along), root)
	answer, _ := near.Float64()
	return answer
}

// grazing from eye height the globe is met where exact arithmetic
// says, to a millimetre.
func TestAGrazingRayMeetsWhereExactArithmeticSays(t *testing.T) {
	ball := mustGlobe(t, earth)
	for _, radians := range []float64{1e-2, 2e-3, 1e-3} {
		want := exact(earth, 1.7, radians)
		got, met := ball.Hit(maths.Ray{At: maths.V(0, earth+1.7, 0),
			Dir: below(radians)}, 0, math.Inf(1))
		if !met || math.Abs(got-want) > 1e-3 {
			t.Errorf("%g rad under level the ground is %.6f m off; "+
				"exactly, %.6f", radians, got, want)
		}
	}
}

// the globe's way out is away from its middle, and its distance is
// how far above it a point is.
func TestTheGlobeFacesOutAndMeasuresHeight(t *testing.T) {
	ball := mustGlobe(t, earth)
	at := maths.V(3e6, 4e6, earth)
	if !ball.Normal(at).Near(at.Unit(), 1e-15) {
		t.Errorf("at %v the globe faces %v", at, ball.Normal(at))
	}
	if got := ball.Distance(maths.V(0, earth+250, 0)); math.Abs(
		got-250) > 1e-6 {
		t.Errorf("250 m up is %g m from the globe", got)
	}
}

// a globe with no size, or one that is no number, is none.
func TestAGlobeWithNoSizeIsRefused(t *testing.T) {
	for _, radius := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := New(radius); !errors.Is(err, ErrGlobe) {
			t.Errorf("a globe of %g is made: %v", radius, err)
		}
	}
}
