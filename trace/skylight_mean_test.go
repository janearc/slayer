package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// The sky's light on a matte surface is weighed by the cosine to its
// normal: the surface's colour times the irradiance over pi. Under a sky
// of radiance a + b e, e the elevation, a floor's is
//
//	(1/pi) integral of (a + b e) cos(t) over the upper half
//	  = a + 2 b integral of (pi/2 - t) cos(t) sin(t) dt, t from 0 to pi/2
//	  = a + b pi / 4,
//
// and the sixteen looks give exactly that: the look at share s of the
// disc is at elevation asin(sqrt(1 - s)), and its pair at share 1 - s is
// at pi/2 less that, so each pair's mean is pi/4. With a = 1, b = 0.5 and
// a colour of a half that is 0.5 (1 + pi/8), 0.6963.
//
// Looks spread evenly over the half sphere and not by the cosine would
// give a + b (pi/2 - 1), which makes 0.6427.
func TestTheSkyIsGatheredByTheCosineToTheNormal(t *testing.T) {
	scene := skyOver(rising{floor: 1, slope: 0.5})
	got := scene.skylight(maths.Vec{}, maths.Up, maths.Grey(0.5))
	lightIs(t, "a floor under a rising sky", got,
		maths.Grey(0.5*(1+math.Pi/8)), 1e-9)
}

// Anti: the cosine is to the surface's own normal, not to straight up.
// A ceiling under the same sky looks at its lower half, where every
// elevation is the floor's turned over, and takes a - b pi / 4: with the
// same numbers 0.5 (1 - pi/8), 0.3037.
func TestACeilingTakesTheLowerHalfOfTheSky(t *testing.T) {
	scene := skyOver(rising{floor: 1, slope: 0.5})
	got := scene.skylight(maths.Vec{}, maths.Up.Neg(), maths.Grey(0.5))
	lightIs(t, "a ceiling under a rising sky", got,
		maths.Grey(0.5*(1-math.Pi/8)), 1e-9)
}

// wallResidue is the most the sixteen looks' mean elevation can be off
// nought for a wall, in radians, whichever way the wall's own frame is
// turned about its normal: the mean of asin(sqrt(s) sin(turn)) over the
// looks, s the look's share of the disc and turn its golden-angle turn
// plus any one angle. TestTheWallResidueIsUnderItsBound sweeps that
// angle and holds the mean's largest size under this, rounded up.
const wallResidue = 0.05

// wallSweepStep is the step of the angle the wall's frame is turned
// by in the sweep of the residue: a hundredth of a radian.
const wallSweepStep = 0.01

// worstWallMean is the largest size, over the angles the wall's frame
// can be turned by, of the mean elevation of the sixteen looks above a
// wall's horizon, when each look's turn is turnEach more than the one
// before. Its elevation is asin(sqrt(s) sin(turn)) for s its share of
// the disc, as the frame's other way is straight up.
func worstWallMean(turnEach float64) float64 {
	worst := 0.0
	for angle := 0.0; angle < 2*math.Pi; angle += wallSweepStep {
		mean := 0.0
		for look := range skyLooks {
			share := (float64(look) + 0.5) / skyLooks
			turn := turnEach*float64(look) + angle
			mean += math.Asin(math.Sqrt(share) * math.Sin(turn))
		}
		worst = max(worst, math.Abs(mean/skyLooks))
	}
	return worst
}

// The wall's residue, the mean elevation its sixteen looks leave, is
// under wallResidue at every angle of the frame about the wall's normal,
// swept at every hundredth of a radian, and is not nothing: it is over
// four fifths of the bound, so the bound is not slack.
func TestTheWallResidueIsUnderItsBound(t *testing.T) {
	worst := worstWallMean(goldenTurn)
	if !(worst <= wallResidue && worst >= 0.8*wallResidue) {
		t.Errorf("the wall's largest mean elevation is %v, want "+
			"between %v and %v", worst, 0.8*wallResidue, wallResidue)
	}
}

// Anti: looks that were not turned from one another, all at one turn,
// leave a mean elevation far over the bound, which the same sweep
// reports.
func TestLooksAllAtOneTurnLeaveAResidueOverTheBound(t *testing.T) {
	if worst := worstWallMean(0); worst <= wallResidue {
		t.Errorf("looks at one turn leave %v, under the bound %v",
			worst, wallResidue)
	}
}

// A wall sees as much sky above the horizon as below it, each height as
// squarely as its opposite, so under a sky that brightens with height
// the rise cancels and the wall takes the sky's light at the horizon:
// a, and with a = 1, b = 0.5 and a colour of a half, 0.5. The sixteen
// looks leave of the rise at most wallResidue of b.
//
// A frame built round straight up and not round the wall's normal would
// give the floor's 0.6963, and round straight down the ceiling's 0.3037.
func TestAWallTakesTheSkyAtTheHorizon(t *testing.T) {
	scene := skyOver(rising{floor: 1, slope: 0.5})
	for _, normal := range []maths.Vec{maths.V(1, 0, 0), maths.V(0, 0, -1),
		maths.V(0.6, 0, 0.8)} {
		got := scene.skylight(maths.Vec{}, normal, maths.Grey(0.5))
		lightIs(t, "a wall under a rising sky", got, maths.Grey(0.5),
			0.5*0.5*wallResidue)
	}
}

// Anti: under a sky of one radiance everywhere the way a surface faces
// changes nothing: floor, wall and ceiling each give back their colour
// times it.
func TestUnderAnEvenSkyEveryFacingTakesTheSame(t *testing.T) {
	scene := skyOver(even(skyBlue))
	for _, normal := range []maths.Vec{maths.Up, maths.Up.Neg(),
		maths.V(1, 0, 0), maths.V(0.6, 0, 0.8)} {
		got := scene.skylight(maths.Vec{}, normal, maths.Grey(0.5))
		lightIs(t, "under an even sky", got, skyBlue.Scale(0.5), 1e-12)
	}
}

// squareFrame is whether two ways and a normal are each a unit long and
// at right angles to one another, to twelve places.
func squareFrame(first, second, normal maths.Vec) bool {
	const near = 1e-12
	return within(first.Len(), 1, near) && within(second.Len(), 1, near) &&
		within(first.Dot(second), 0, near) &&
		within(first.Dot(normal), 0, near) &&
		within(second.Dot(normal), 0, near)
}

// Round any unit normal, across gives two ways a unit long, square to
// the normal and to each other: the frame the sky's looks are laid out
// in. As a property over the whole sphere, and at the two ends of z and
// along each axis, where a frame built by a cross product gives out.
func TestAcrossIsASquareFrameRoundAnyNormal(t *testing.T) {
	for _, normal := range []maths.Vec{maths.V(0, 0, 1), maths.V(0, 0, -1),
		maths.V(1, 0, 0), maths.V(-1, 0, 0), maths.V(0, 1, 0),
		maths.V(0, -1, 0)} {
		if first, second := across(normal); !squareFrame(first, second,
			normal) {
			t.Errorf("round %v the frame is %v and %v", normal, first,
				second)
		}
	}
	rapid.Check(t, func(draw *rapid.T) {
		up := rapid.Float64Range(-1, 1).Draw(draw, "up")
		turn := rapid.Float64Range(0, 2*math.Pi).Draw(draw, "turn")
		out := math.Sqrt(1 - up*up)
		normal := maths.V(out*math.Cos(turn), out*math.Sin(turn), up)
		if first, second := across(normal); !squareFrame(first, second,
			normal) {
			draw.Fatalf("round %v the frame is %v and %v", normal, first,
				second)
		}
	})
}

// Anti: the frame is square only round a normal a unit long. Round one
// twice as long the two ways are not, and the check says so.
func TestAcrossRoundALongNormalIsNoSquareFrame(t *testing.T) {
	normal := maths.V(0.6, 0, 0.8).Scale(2)
	if first, second := across(normal); squareFrame(first, second, normal) {
		t.Errorf("round a normal two long the frame %v and %v passed "+
			"for square", first, second)
	}
}
