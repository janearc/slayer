package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// the four named eases are css's cubic beziers through (0, 0) and
// (1, 1). the curve at t is
//
//	x(t) = 3(1-t)^2 t x1 + 3(1-t) t^2 x2 + t^3
//	y(t) = 3(1-t)^2 t y1 + 3(1-t) t^2 y2 + t^3
//
// and the eased share at x is the y where the x is. t of a half has
// weights 0.375, 0.375 and 0.125; t of a quarter has 0.421875, 0.140625
// and 0.015625. each row is worked from css's published points.
func TestTheNamedEasesAreCssBeziers(t *testing.T) {
	for _, c := range []struct {
		name    string
		x, want float64
		working string
	}{
		// ease (0.25, 0.1, 0.25, 1)
		{"ease", 0.3125, 0.5375, "t=1/2: x .375*.25+.375*.25+.125, y .375*.1+.375*1+.125"},
		{"ease", 0.15625, 0.1984375, "t=1/4: x .421875*.25+.140625*.25+.015625, y .421875*.1+.140625*1+.015625"},
		{"", 0.3125, 0.5375, "no name is ease"},
		// ease-in (0.42, 0, 1, 1)
		{"ease-in", 0.6575, 0.5, "t=1/2: x .375*.42+.375*1+.125, y .375*0+.375*1+.125"},
		{"ease-in", 0.3334375, 0.15625, "t=1/4: x .421875*.42+.140625*1+.015625, y .140625+.015625"},
		// ease-out (0, 0, 0.58, 1)
		{"ease-out", 0.3425, 0.5, "t=1/2: x .375*0+.375*.58+.125, y .375*0+.375*1+.125"},
		{"ease-out", 0.0971875, 0.15625, "t=1/4: x .421875*0+.140625*.58+.015625, y .140625+.015625"},
		// ease-in-out (0.42, 0, 0.58, 1)
		{"ease-in-out", 0.5, 0.5, "t=1/2: x .375*.42+.375*.58+.125, y .375*0+.375*1+.125"},
		{"ease-in-out", 0.274375, 0.15625, "t=1/4: x .421875*.42+.140625*.58+.015625, y .140625+.015625"},
		// linear is the share itself; a name css does not have too
		{"linear", 0.3, 0.3, "linear"},
		{"linear", 0.9, 0.9, "linear"},
		{"swoop", 0.3, 0.3, "an unknown name is linear"},
		// the ends of every curve are the ends
		{"ease", 0, 0, "ends"},
		{"ease", 1, 1, "ends"},
		{"ease-in", 0, 0, "ends"},
		{"ease-in", 1, 1, "ends"},
		{"ease-out", 0, 0, "ends"},
		{"ease-out", 1, 1, "ends"},
		{"ease-in-out", 0, 0, "ends"},
		{"ease-in-out", 1, 1, "ends"},
	} {
		if got := Eased(c.name, c.x); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%q at %v is %v, want %v (%s)", c.name, c.x, got,
				c.want, c.working)
		}
	}
	for _, name := range Eases {
		if !IsEase(name) {
			t.Errorf("%s is named in Eases and is not an ease", name)
		}
	}
	for _, name := range []string{"", "swoop", "Ease", "ease-", "linearr"} {
		if IsEase(name) {
			t.Errorf("%q is an ease", name)
		}
	}
}

// every ease but linear gives a share below nought what it gives
// nought, since its curve is never below nought and the halvings take
// the same half at every step; and two what it gives one, which is one
// for every ease but ease-in, where it is the float below one. linear,
// and a name that is no ease, pass the share on. .mutants leans on
// these numbers for At's guards.
func TestEasesBelowNoughtAndPastOne(t *testing.T) {
	for _, name := range Eases {
		if name == "linear" {
			continue
		}
		atNought := Eased(name, 0)
		for _, x := range []float64{-1e-300, -0.5, -1, -5, -1e300} {
			if got := Eased(name, x); got != atNought {
				t.Errorf("%s at %v is %v, at nought %v", name, x, got,
					atNought)
			}
		}
		if got := Eased(name, 2); got != 1 {
			t.Errorf("%s at 2 is %v, not 1", name, got)
		}
		wantOne := 1.0
		if name == "ease-in" {
			wantOne = math.Nextafter(1, 0)
		}
		if got := Eased(name, 1); got != wantOne {
			t.Errorf("%s at 1 is %v, want %v", name, got, wantOne)
		}
	}
	for _, name := range []string{"linear", "swoop"} {
		for _, x := range []float64{-5, -1, 0, 1, 2} {
			if got := Eased(name, x); got != x {
				t.Errorf("%s at %v is %v", name, x, got)
			}
		}
	}
}

// the curve is solved for the time axis by forty halvings, and the
// share it gives is the curve at the middle of what is left.
//
// where the curve meets x exactly, at t of a half, the half that holds
// is the lower one, so what is left ends a half of 2 to the -40 below
// a half: t is a half less 2 to the -41.
//
// the share is then the curve's y there, which a first order step from
// the y at a half gives: y of a half less y' of a half times 2 to the
// -41, the next term being of the order of 2 to the -82. y' is worked
// from y(t) = 3(1-t)^2 t y1 + 3(1-t) t^2 y2 + t^3:
//
//	ease (y1 .1, y2 1):  y' = .3[(1-t)^2 - 2t(1-t)] + 3[2t(1-t) - t^2] + 3t^2,
//	                     at a half -.075 + .75 + .75 = 1.425, and y is .5375
//	ease-in-out (0, 1):  y' = 3[2t(1-t) - t^2] + 3t^2,
//	                     at a half .75 + .75 = 1.5, and y is .5
func TestTheHalvingsSettleBelowAnExactMeet(t *testing.T) {
	step := math.Ldexp(1, -41)
	for _, c := range []struct {
		name string
		x    float64
		want float64
	}{
		{"ease", 0.3125, 0.5375 - 1.425*step},
		{"ease-in-out", 0.5, 0.5 - 1.5*step},
	} {
		got := Eased(c.name, c.x)
		if math.Abs(got-c.want) > 1e-15 {
			t.Errorf("%s at %v is %.17g, want %.17g (a half of 2 to the "+
				"-40 below the meet)", c.name, c.x, got, c.want)
		}
	}
}

// along is a keyed run along x and up y: frames at nought, a half and
// one at 0 0 0, 10 0 0 and 10 20 0, standing at 1 2 3 before.
func along(hold bool) Keyed {
	return Keyed{Length: 10 * time.Second, Ease: "linear", Hold: hold,
		Base: maths.V(1, 2, 3), Frames: []Keyframe{{0, maths.V(0, 0, 0)},
			{0.5, maths.V(10, 0, 0)}, {1, maths.V(10, 20, 0)}}}
}

// a share of the run is blended between the two keyframes it falls
// between: a quarter of the run is half way from the first to the
// second, 5 0 0, and three quarters half way from the second to the
// third, 10 10 0.
func TestKeyframesBlendBetweenTheTwoAroundTheShare(t *testing.T) {
	for _, c := range []struct {
		at   time.Duration
		want maths.Vec
		what string
	}{
		{0, maths.V(0, 0, 0), "the start is the first frame"},
		{2500 * time.Millisecond, maths.V(5, 0, 0), "a quarter is half way to the middle frame"},
		{5 * time.Second, maths.V(10, 0, 0), "half is the middle frame"},
		{7500 * time.Millisecond, maths.V(10, 10, 0), "three quarters is half way to the last"},
		{10 * time.Second, maths.V(10, 20, 0), "the end is the last frame"},
		{time.Second, maths.V(2, 0, 0), "a tenth is a fifth of the way to the middle"},
		{9 * time.Second, maths.V(10, 16, 0), "nine tenths is four fifths of the way on"},
		{-time.Second, maths.V(0, 0, 0), "before the start is the first frame"},
	} {
		near(t, "held, "+c.what, along(true).At(c.at).Origin, c.want)
		near(t, "not held, "+c.what, along(false).At(c.at).Origin, c.want)
	}
	// past the end, a held run keeps the last frame and one not held
	// goes back to its base
	near(t, "an hour on, held", along(true).At(time.Hour).Origin,
		maths.V(10, 20, 0))
	near(t, "an hour on, not held", along(false).At(time.Hour).Origin,
		maths.V(1, 2, 3))
	near(t, "a moment past the end, not held",
		along(false).At(10*time.Second+time.Millisecond).Origin,
		maths.V(1, 2, 3))
}

// the eased share is blended once over the whole run: an ease-in-out
// push from 0 to 8 is at 4 where the curve is a half, at the middle of
// the run, and at 8 times 0.15625, 1.25, a share of 0.274375 in, where
// the curve (worked above) is 0.15625.
func TestAnEasedRunIsBlendedOnce(t *testing.T) {
	push := Keyed{Length: 100 * time.Second, Ease: "ease-in-out",
		Hold: true, Frames: []Keyframe{{0, maths.V(0, 0, 0)},
			{1, maths.V(8, 0, 0)}}}
	near(t, "the middle", push.At(50*time.Second).Origin, maths.V(4, 0, 0))
	near(t, "a share of 0.274375", push.At(27437500*time.Microsecond).Origin,
		maths.V(1.25, 0, 0))
}

// a run of no length, or of none, or with no frames, stands at its base;
// one of a single nanosecond is run, and ends on its last frame at that
// nanosecond.
func TestRunsOfNoLengthAndOfOneNanosecond(t *testing.T) {
	frames := []Keyframe{{0, maths.V(1, 0, 0)}, {1, maths.V(2, 0, 0)}}
	base := maths.V(9, 9, 9)
	for _, length := range []time.Duration{0, -time.Second} {
		run := Keyed{Length: length, Frames: frames, Base: base, Hold: true}
		for _, at := range []time.Duration{0, time.Second} {
			if got := run.At(at).Origin; got != base {
				t.Errorf("a run of %v at %v is at %v, want its base",
					length, at, got)
			}
		}
	}
	if got := (Keyed{Length: time.Second, Base: base}).At(0).Origin; got != base {
		t.Errorf("a run of no frames is at %v", got)
	}
	run := Keyed{Length: time.Nanosecond, Frames: frames, Base: base,
		Ease: "linear"}
	if got := run.At(time.Nanosecond).Origin; got != maths.V(2, 0, 0) {
		t.Errorf("a run of one nanosecond at its end is at %v", got)
	}
	if got := run.At(0).Origin; got != maths.V(1, 0, 0) {
		t.Errorf("a run of one nanosecond at its start is at %v", got)
	}
	if got := run.At(2 * time.Nanosecond).Origin; got != base {
		t.Errorf("a run of one nanosecond, two on, is at %v", got)
	}
}

// a first keyframe that is not at nought is held until its share; two
// frames at one share are each the place at that share, the later one
// winning at it and the earlier never showing, as css lets the later
// rule win.
func TestLateFirstFramesAndSharedShares(t *testing.T) {
	late := Keyed{Length: 10 * time.Second, Ease: "linear", Hold: true,
		Frames: []Keyframe{{0.2, maths.V(1, 0, 0)}, {1, maths.V(2, 0, 0)}}}
	for _, c := range []struct {
		at   time.Duration
		x    float64
		what string
	}{
		{0, 1, "before the first frame its place is held"},
		{time.Second, 1, "a tenth in, still held"},
		{2 * time.Second, 1, "at the first frame's own share it is that place"},
		{6 * time.Second, 1.5, "0.6 is half way from 0.2 to the end"},
		{10 * time.Second, 2, "the end"},
	} {
		near(t, c.what, late.At(c.at).Origin, maths.V(c.x, 0, 0))
	}
	same := Keyed{Length: 10 * time.Second, Ease: "linear", Hold: true,
		Frames: []Keyframe{{0, maths.V(1, 0, 0)}, {0, maths.V(2, 0, 0)},
			{1, maths.V(3, 0, 0)}}}
	near(t, "the first of two frames at nought, at nought",
		same.At(0).Origin, maths.V(1, 0, 0))
	near(t, "a quarter on, from the later of the two",
		same.At(2500*time.Millisecond).Origin, maths.V(2.25, 0, 0))
	mid := Keyed{Length: 10 * time.Second, Ease: "linear", Hold: true,
		Frames: []Keyframe{{0, maths.V(0, 0, 0)}, {0.5, maths.V(1, 0, 0)},
			{0.5, maths.V(2, 0, 0)}, {1, maths.V(3, 0, 0)}}}
	near(t, "before the shared share", mid.At(4*time.Second).Origin,
		maths.V(0.8, 0, 0))
	near(t, "at the shared share, the later frame",
		mid.At(5*time.Second).Origin, maths.V(2, 0, 0))
	near(t, "after the shared share", mid.At(6*time.Second).Origin,
		maths.V(2.2, 0, 0))
}
