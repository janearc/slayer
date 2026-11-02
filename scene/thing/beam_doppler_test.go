package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the boost is delta cubed, with delta = 1 / (gamma (1 - beta cos)).
// worked by hand for a jet at 0.6 c, where gamma = 1 / sqrt(1 - 0.36)
// = 1.25:
//
//	at the eye:      delta = 1 / (1.25 * 0.4) = 2,       boost 8
//	away from it:    delta = 1 / (1.25 * 1.6) = 0.5,     boost 0.125
//	across the way:  delta = 1 / 1.25 = 0.8,             boost 0.512
//	at cos 0.8:      delta = 1 / (1.25 * 0.52) = 1/0.65, boost 1/0.274625
func TestBeamBoostIsDeltaCubed(t *testing.T) {
	b := Beam{Speed: 0.6}
	up := maths.V(0, 1, 0)
	for _, c := range []struct {
		name   string
		toward maths.Vec
		at     maths.Vec
		want   float64
	}{
		{"at the eye", maths.Up, up, 8},
		{"at the eye, not one unit long", maths.V(0, 3, 0), up, 8},
		{"away from it", maths.Up.Neg(), up, 0.125},
		{"across the way", maths.XAxis, up, 0.512},
		{"at cos 0.8", maths.V(0.6, 0.8, 0), up, 1 / 0.274625},
		{"the lower jet at the eye", maths.Up.Neg(), maths.V(0, -1, 0), 8},
		{"the lower jet away from it", maths.Up, maths.V(0, -1, 0), 0.125},
	} {
		if got := b.Boost(c.toward, c.at); math.Abs(got-c.want) > 1e-12*c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// a beam at light's speed or past it, or at no speed or a negative one,
// boosts nothing: one, whichever way it is seen from. past light's speed
// gamma is no number, and at it, endless.
func TestABeamOutOfRangeBoostsNothing(t *testing.T) {
	for _, speed := range []float64{0, -0.5, 1, 1.5, 2, 100} {
		b := Beam{Speed: speed}
		for _, toward := range []maths.Vec{maths.Up, maths.Up.Neg(), maths.XAxis} {
			if got := b.Boost(toward, maths.V(0, 1, 0)); got != 1 {
				t.Errorf("a beam at %v seen along %v boosts %v, want 1", speed, toward, got)
			}
		}
		round := Beam{Speed: speed, Round: true, At: 1}
		if got := round.Boost(maths.V(0, 0, -1), maths.V(1, 0, 0)); got != 1 {
			t.Errorf("a round beam at %v boosts %v, want 1", speed, got)
		}
	}
}

// a beam of no speed boosts nothing even where the slowing outside At has no
// number: At is so large that At over the radius, at a point a ten-billionth
// out, is endless, and zero of endless is no number.
func TestABeamOfNoSpeedBoostsNothingAnywhere(t *testing.T) {
	b := Beam{Speed: 0, Round: true, At: 1e300}
	if got := b.Boost(maths.Up, maths.V(1e-10, 0, 0)); got != 1 {
		t.Errorf("a beam of no speed boosts %v, want 1", got)
	}
}

// round, the gas on the axis is still, so light from it is not boosted,
// whichever way it goes, down to a direction of no number.
func TestStillGasOnTheAxisIsNotBoosted(t *testing.T) {
	b := Beam{Speed: 0.6, Round: true, At: 1}
	for _, toward := range []maths.Vec{maths.Up, maths.XAxis, {}, maths.V(math.NaN(), 0, 0)} {
		for _, at := range []maths.Vec{{}, maths.V(0, 5, 0), maths.V(0, -2, 0)} {
			if got := b.Boost(toward, at); got != 1 {
				t.Errorf("on the axis at %v seen along %v the boost is %v, want 1", at, toward, got)
			}
		}
	}
}

// a round beam boosts by its flow where the gas is: worked by hand for
// a speed of 0.8 with At = 9, 16 out, where the flow is
// 0.8 * sqrt(9/16) = 0.6, and so gamma 1.25 and the same deltas as a
// jet at 0.6 c, with the flow along -z at +x.
func TestARoundBeamBoostsByItsFlowThere(t *testing.T) {
	b := Beam{Speed: 0.8, Round: true, At: 9}
	at := maths.V(16, 3, 0)
	for _, c := range []struct {
		toward maths.Vec
		want   float64
	}{
		{maths.V(0, 0, -1), 8}, {maths.V(0, 0, 1), 0.125}, {maths.Up, 0.512},
	} {
		if got := b.Boost(c.toward, at); math.Abs(got-c.want) > 1e-12*c.want {
			t.Errorf("seen along %v the boost is %v, want %v", c.toward, got, c.want)
		}
	}
}

// the flow of a beam along its axis is away from the origin on each
// side, on the equator as well, and along an axis of its own, as long
// as one unit.
func TestAJetFlowsAwayFromItsOrigin(t *testing.T) {
	jet := Beam{Speed: 0.6}
	for _, c := range []struct {
		at, want maths.Vec
	}{
		{maths.V(0, 0.5, 0), maths.Up}, {maths.V(1, 0, 0), maths.Up},
		{maths.V(0, 2, 0), maths.Up}, {maths.V(0, -0.5, 0), maths.Up.Neg()},
		{maths.V(0, 0, 0), maths.Up},
	} {
		if dir, speed := jet.Flow(c.at); dir != c.want || speed != 0.6 {
			t.Errorf("at %v the jet flows %v at %v, want %v at 0.6", c.at, dir, speed, c.want)
		}
	}
	tipped := Beam{Speed: 0.6, Axis: maths.V(3, 0, 0)}
	if dir, _ := tipped.Flow(maths.V(2, 0, 0)); dir != maths.XAxis {
		t.Errorf("an axis of (3, 0, 0) flows %v at +x, want one unit of x", dir)
	}
	if dir, _ := tipped.Flow(maths.V(-2, 0, 0)); dir != maths.XAxis.Neg() {
		t.Errorf("an axis of (3, 0, 0) flows %v at -x", dir)
	}
}

// a round flow goes round the axis, counterclockwise seen from its end,
// by the part of the point square to the axis.
//
// at (1, 2, 0) about up that is (1, 0, 0), and up cross x is -z. about
// x, the point (5, 0, 2) is (0, 0, 2) square to it, and x cross z is -y.
func TestARoundFlowGoesRoundTheAxis(t *testing.T) {
	up := Beam{Speed: 0.4, Round: true}
	if dir, speed := up.Flow(maths.V(1, 2, 0)); dir != maths.V(0, 0, -1) || speed != 0.4 {
		t.Errorf("about up at (1, 2, 0) the flow is %v at %v, want -z at 0.4", dir, speed)
	}
	if dir, _ := up.Flow(maths.V(1, -7, 0)); dir != maths.V(0, 0, -1) {
		t.Errorf("about up at (1, -7, 0) the flow is %v, want -z", dir)
	}
	x := Beam{Speed: 0.4, Round: true, Axis: maths.XAxis}
	if dir, _ := x.Flow(maths.V(5, 0, 2)); dir != maths.V(0, -1, 0) {
		t.Errorf("about x at (5, 0, 2) the flow is %v, want -y", dir)
	}
}

// on the axis the gas is still: a flow of no way and no speed.
func TestTheFlowOnTheAxisIsStill(t *testing.T) {
	b := Beam{Speed: 0.4, Round: true, At: 1}
	for _, at := range []maths.Vec{{}, maths.V(0, 5, 0), maths.V(0, -1, 0)} {
		if dir, speed := b.Flow(at); dir != (maths.Vec{}) || speed != 0 {
			t.Errorf("on the axis at %v the flow is %v at %v", at, dir, speed)
		}
	}
}

// a round flow slows outside At by the square root, never passes 0.99
// of light's speed, inside At as well, and with no At does not slow.
//
// worked by hand: 0.8 * sqrt(9/16) = 0.6; 0.9 * sqrt(4/1) = 1.8, which
// is held to 0.99; 0.5 * sqrt(1/0.0625) = 2, held to 0.99; and with no
// At, 0.4 anywhere.
func TestARoundFlowSlowsByTheSquareRootAndIsHeld(t *testing.T) {
	for _, c := range []struct {
		b    Beam
		at   maths.Vec
		want float64
	}{
		{Beam{Speed: 0.8, Round: true, At: 9}, maths.V(16, 0, 0), 0.6},
		{Beam{Speed: 0.9, Round: true, At: 4}, maths.V(1, 0, 0), 0.99},
		{Beam{Speed: 0.5, Round: true, At: 1}, maths.V(0.25, 0, 0), 0.99},
		{Beam{Speed: 0.4, Round: true}, maths.V(4, 0, 0), 0.4},
		{Beam{Speed: 0.4, Round: true}, maths.V(0.01, 0, 0), 0.4},
	} {
		if _, got := c.b.Flow(c.at); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("%+v at %v flows at %v, want %v", c.b, c.at, got, c.want)
		}
	}
}
