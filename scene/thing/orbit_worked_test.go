package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// near fails the test when got is not within 1e-9 of want.
func near(t *testing.T, what string, got, want maths.Vec) {
	t.Helper()
	if !got.Near(want, 1e-9) {
		t.Errorf("%s is %v, want %v", what, got, want)
	}
}

// the angle is Phase and a turn every Period: worked by hand, a period of one
// nanosecond is a turn a nanosecond, so one nanosecond on is Phase + 2 pi; a
// quarter period is a quarter turn; no period, or one less than zero, never
// moves.
func TestAngleIsPhaseAndTurns(t *testing.T) {
	fast := Orbit{Period: 1, Phase: 0.5}
	if got, want := fast.Angle(time.Nanosecond), 0.5+2*math.Pi; math.Abs(got-want) > 1e-12 {
		t.Errorf("a nanosecond into a nanosecond orbit: %v, want %v", got, want)
	}
	slow := Orbit{Period: 4 * time.Second, Phase: 0.5}
	if got, want := slow.Angle(time.Second), 0.5+math.Pi/2; math.Abs(got-want) > 1e-12 {
		t.Errorf("a quarter of the way round: %v, want %v", got, want)
	}
	for _, p := range []time.Duration{0, -1, -time.Hour} {
		o := Orbit{Period: p, Phase: 0.5}
		if got := o.Angle(time.Hour); got != 0.5 {
			t.Errorf("a period of %v moved to %v", p, got)
		}
	}
}

// a quarter turn is where the right-hand rule says. about x the plane's
// first axis is z, and counterclockwise seen from +x, y goes to z, so
// z goes to -y.
//
// at a radius of 3: (0, 0, 3), then (0, -3, 0), then (0, 0, -3).
func TestAnOrbitAboutXGoesFromZToMinusY(t *testing.T) {
	o := Orbit{Radius: 3, Axis: maths.XAxis, Period: 4 * time.Second}
	near(t, "the start", o.At(0).Origin, maths.V(0, 0, 3))
	near(t, "a quarter on", o.At(time.Second).Origin, maths.V(0, -3, 0))
	near(t, "half on", o.At(2*time.Second).Origin, maths.V(0, 0, -3))
}

// a tilted axis, worked by hand: about (1, 1, 0), n = (1, 1, 0)/sqrt 2,
// which is 0.707 along x and under 0.9, so the first axis is x less its
// part along n: e = (1, -1, 0)/sqrt 2, and f = n cross e = (0, 0, -1).
//
// at a radius of 2 the start is (sqrt 2, -sqrt 2, 0), a quarter on is
// (0, 0, -2), an eighth on is 2 (cos 45 e + sin 45 f) = (1, -1, -sqrt 2).
func TestATiltedOrbitGoesRoundItsOwnPlane(t *testing.T) {
	o := Orbit{Radius: 2, Axis: maths.V(1, 1, 0), Period: 8 * time.Second}
	r2 := math.Sqrt2
	near(t, "the start", o.At(0).Origin, maths.V(r2, -r2, 0))
	near(t, "an eighth on", o.At(time.Second).Origin, maths.V(1, -1, -r2))
	near(t, "a quarter on", o.At(2*time.Second).Origin, maths.V(0, 0, -2))
	// the plane is square to the axis all the way round
	for k := range 8 {
		at := o.At(time.Duration(k) * time.Second).Origin
		if d := at.Dot(maths.V(1, 1, 0)); math.Abs(d) > 1e-9 {
			t.Errorf("at %d s the orbit left its plane: %v along its axis", k, d)
		}
	}
}

// the first axis is x less its part along the axis, unless the axis is
// past 0.9 along x, and so nearly along it, when it is z.
//
// an axis of (0.9, y, 0) with y = sqrt 0.19 is a unit vector just 0.9
// along x and not past it, so the first axis is x less 0.9 n, which is
// (0.19, -0.9 y, 0) over its length y: e = (y, -0.9, 0). then
// f = n cross e = (0, 0, -(0.81 + 0.19)) = (0, 0, -1).
func TestAnAxisJustUnderNineTenthsAlongXKeepsX(t *testing.T) {
	y := 0.43588989435406733
	axis := maths.V(0.9, y, 0)
	if axis.Unit().Dot(maths.XAxis) != 0.9 {
		t.Fatalf("the axis is %v along x, not 0.9 exactly", axis.Unit().Dot(maths.XAxis))
	}
	o := Orbit{Radius: 1, Axis: axis, Period: 4 * time.Second}
	near(t, "the start", o.At(0).Origin, maths.V(y, -0.9, 0))
	near(t, "a quarter on", o.At(time.Second).Origin, maths.V(0, 0, -1))
}

// a Phase that is not zero starts the orbit round the circle. about up, from +x
// toward -z, a quarter of the way in is -z: at a radius of 3 and a period of 8
// s, the start is (0, 0, -3), 2 s on is (-3, 0, 0) and 4 s on is (0, 0, 3).
func TestAnOrbitStartsAtItsPhase(t *testing.T) {
	o := Orbit{Radius: 3, Period: 8 * time.Second, Phase: math.Pi / 2}
	near(t, "the start", o.At(0).Origin, maths.V(0, 0, -3))
	near(t, "2 s on", o.At(2*time.Second).Origin, maths.V(-3, 0, 0))
	near(t, "4 s on", o.At(4*time.Second).Origin, maths.V(0, 0, 3))
	still := Orbit{Radius: 3, Phase: math.Pi / 2}
	near(t, "with no period", still.At(time.Hour).Origin, maths.V(0, 0, -3))
}

// a locked orbit turns the thing as far as it has gone round since it
// started, not as far as its angle: with a Phase of a quarter turn the
// thing has gone nowhere at the start, so it faces as Facing does, and a
// quarter turn on it has turned a quarter about up.
//
// worked by hand with Facing a quarter turn about x, which takes y to z
// and z to -y: at the start y goes to z, z to -y, x stays. a quarter
// on, a quarter about up follows that: y to z to x, x to -z, z to -y.
func TestALockedOrbitTurnsFromItsPhase(t *testing.T) {
	o := Orbit{Radius: 2, Period: 4 * time.Second, Phase: math.Pi / 2,
		Facing: maths.Rotation(maths.XAxis, math.Pi/2), Locked: true}
	start, quarter := o.At(0).Orientation, o.At(time.Second).Orientation
	near(t, "y at the start", start.Out(maths.YAxis), maths.ZAxis)
	near(t, "z at the start", start.Out(maths.ZAxis), maths.YAxis.Neg())
	near(t, "x at the start", start.Out(maths.XAxis), maths.XAxis)
	near(t, "y a quarter on", quarter.Out(maths.YAxis), maths.XAxis)
	near(t, "x a quarter on", quarter.Out(maths.XAxis), maths.ZAxis.Neg())
	near(t, "z a quarter on", quarter.Out(maths.ZAxis), maths.YAxis.Neg())
	// and it is where the phase puts it, still
	near(t, "the start", o.At(0).Origin, maths.V(0, 0, -2))
}

// a nested motion is its child inside its parent, worked by hand for a
// parent of scale 2 turned a quarter about y, standing at (1, 2, 3), and
// a child of scale 3 at (1, 0, 0) in it. a quarter about y takes x to
// -z and z to x.
//
// the child is at (1, 2, 3) + 2 * (0, 0, -1) = (1, 2, 1), at scale
// 2 * 3 = 6, turned as its parent is. its point (1, 0, 0) is at
// (1, 2, 1) + 6 * (0, 0, -1) = (1, 2, -5), and its (0, 1, 0) at
// (1, 2, 1) + 6 * (0, 1, 0) = (1, 8, 1).
func TestNestedIsTheChildInsideTheParent(t *testing.T) {
	parent := maths.Subspace{Origin: maths.V(1, 2, 3),
		Orientation: maths.Rotation(maths.Up, math.Pi/2), Scale: 2}
	child := maths.Subspace{Origin: maths.V(1, 0, 0), Scale: 3}
	n := Nested{Of: Still(parent), In: child}
	for _, at := range []time.Duration{0, time.Hour} {
		s := n.At(at)
		near(t, "the child's origin", s.Origin, maths.V(1, 2, 1))
		if s.Scale != 6 {
			t.Errorf("the child's scale is %v, want 6", s.Scale)
		}
		near(t, "the child's x", s.Orientation.Out(maths.XAxis), maths.V(0, 0, -1))
		near(t, "the child's z", s.Orientation.Out(maths.ZAxis), maths.XAxis)
		near(t, "the child's y", s.Orientation.Out(maths.YAxis), maths.YAxis)
		near(t, "(1, 0, 0) out", s.Out(maths.V(1, 0, 0)), maths.V(1, 2, -5))
		near(t, "(0, 1, 0) out", s.Out(maths.V(0, 1, 0)), maths.V(1, 8, 1))
		near(t, "(1, 2, -5) in", s.In(maths.V(1, 2, -5)), maths.V(1, 0, 0))
		near(t, "(1, 8, 1) in", s.In(maths.V(1, 8, 1)), maths.V(0, 1, 0))
	}
}

// a nested motion with no parent is its child, as it is.
func TestNestedWithNoParentIsTheChild(t *testing.T) {
	child := maths.Subspace{Origin: maths.V(1, 2, 3),
		Orientation: maths.Rotation(maths.XAxis, 1), Scale: 3}
	if got := (Nested{In: child}).At(time.Hour); got != child {
		t.Errorf("a nested motion with no parent is %v, want %v", got, child)
	}
}

// a growing subspace is a scale of 1e-12 before its birth and at it,
// and from then Rate a second: worked by hand, at a rate of 3 a
// quarter second on is 0.75, a second on at a rate of one is exactly
// one, and it grows in a parent of scale 2 at (1, 2, 3) to 2 * 6 = 12.
func TestGrowingScaleIsRateTimesAge(t *testing.T) {
	g := Growing{From: time.Second, Rate: 3}
	for _, c := range []struct {
		at   time.Duration
		want float64
	}{
		{0, 1e-12}, {time.Second, 1e-12}, {time.Second + 250*time.Millisecond, 0.75},
		{3 * time.Second, 6},
	} {
		if got := g.At(c.at).Scale; got != c.want {
			t.Errorf("at %v the scale is %v, want %v", c.at, got, c.want)
		}
	}
	one := Growing{From: time.Second, Rate: 1}
	if got := one.At(2 * time.Second).Scale; got != 1 {
		t.Errorf("a second on at a rate of one the scale is %v, want 1", got)
	}
	if got := (Growing{Rate: 3}).At(time.Second); got.Origin != (maths.Vec{}) ||
		got.Orientation != (maths.Orientation{}) {
		t.Errorf("with no parent it moved or turned: %v", got)
	}
	carried := Growing{Of: Still(maths.Subspace{Origin: maths.V(1, 2, 3), Scale: 2}),
		From: time.Second, Rate: 3}
	s := carried.At(3 * time.Second)
	if s.Scale != 12 || s.Origin != maths.V(1, 2, 3) {
		t.Errorf("carried by a parent it is at %v, scale %v, want (1, 2, 3), 12", s.Origin, s.Scale)
	}
}
