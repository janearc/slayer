package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The tables the worked orbit tests read, beside numbers.yaml.
const (
	orbitWorkedPoints  = "testdata/orbit-worked-points.csv"
	orbitWorkedIdle    = "testdata/orbit-worked-idle-periods.csv"
	orbitWorkedGrowing = "testdata/orbit-worked-growing.csv"
)

// orbitWorkedPoint is the point called name in the points table.
func orbitWorkedPoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := readSheet(t, orbitWorkedPoints).row(t, orbitWorkedPoints, name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// orbitWorkedSeconds is the picked number called name, as whole seconds.
func orbitWorkedSeconds(t *testing.T, name string) time.Duration {
	t.Helper()
	return time.Duration(pickedCount(t, name)) * time.Second
}

// near fails the test when got is not within the picked tolerance of want.
func near(t *testing.T, what string, got, want maths.Vec) {
	t.Helper()
	if !got.Near(want, picked(t, "orbit-worked-tolerance")) {
		t.Errorf("%s is %v, want %v", what, got, want)
	}
}

// the angle is Phase and a turn every Period: worked by hand, a period of one
// nanosecond is a turn a nanosecond, so one nanosecond on is Phase + 2 pi; a
// quarter period is a quarter turn; no period, or one less than zero, never
// moves.
func TestAngleIsPhaseAndTurns(t *testing.T) {
	phase := picked(t, "orbit-worked-phase")
	tolerance := picked(t, "orbit-worked-angle-tolerance")
	fast := Orbit{Period: time.Nanosecond, Phase: phase}
	got, want := fast.Angle(time.Nanosecond), phase+pair*math.Pi
	if !(math.Abs(got-want) <= tolerance) {
		t.Errorf("a nanosecond into a nanosecond orbit: %v, want %v", got, want)
	}
	period := orbitWorkedSeconds(t, "orbit-worked-short-period")
	slow := Orbit{Period: period, Phase: phase}
	got, want = slow.Angle(time.Second), phase+math.Pi/pair
	if !(math.Abs(got-want) <= tolerance) {
		t.Errorf("a quarter of the way round: %v, want %v", got, want)
	}
	for _, row := range readSheet(t, orbitWorkedIdle) {
		idle := time.Duration(cellCount(t, row, "nanoseconds"))
		orbit := Orbit{Period: idle, Phase: phase}
		if got := orbit.Angle(time.Hour); got != phase {
			t.Errorf("a period of %v moved to %v", idle, got)
		}
	}
}

// a quarter turn is where the right-hand rule says. about x the plane's
// first axis is z, and counterclockwise seen from +x, y goes to z, so
// z goes to -y.
//
// at a radius of 3: (0, 0, 3), then (0, -3, 0), then (0, 0, -3).
func TestAnOrbitAboutXGoesFromZToMinusY(t *testing.T) {
	radius := picked(t, "orbit-worked-wide-radius")
	period := orbitWorkedSeconds(t, "orbit-worked-short-period")
	orbit := Orbit{Radius: radius, Axis: maths.XAxis, Period: period}
	start := maths.V(nought, nought, radius)
	quarter := maths.V(nought, -radius, nought)
	half := maths.V(nought, nought, -radius)
	near(t, "the start", orbit.At(nought).Origin, start)
	near(t, "a quarter on", orbit.At(time.Second).Origin, quarter)
	near(t, "half on", orbit.At(pair*time.Second).Origin, half)
}

// a tilted axis, worked by hand: about (1, 1, 0), n = (1, 1, 0)/sqrt 2,
// which is 0.707 along x and under 0.9, so the first axis is x less its
// part along n: e = (1, -1, 0)/sqrt 2, and f = n cross e = (0, 0, -1).
//
// at a radius of 2 the start is (sqrt 2, -sqrt 2, 0), a quarter on is
// (0, 0, -2), an eighth on is 2 (cos 45 e + sin 45 f) = (1, -1, -sqrt 2).
func TestATiltedOrbitGoesRoundItsOwnPlane(t *testing.T) {
	radius := picked(t, "orbit-worked-small-radius")
	axis := maths.V(unity, unity, nought)
	period := orbitWorkedSeconds(t, "orbit-worked-long-period")
	orbit := Orbit{Radius: radius, Axis: axis, Period: period}
	rootTwo := math.Sqrt2
	start := maths.V(rootTwo, -rootTwo, nought)
	eighth := maths.V(unity, -unity, -rootTwo)
	quarter := maths.V(nought, nought, -radius)
	near(t, "the start", orbit.At(nought).Origin, start)
	near(t, "an eighth on", orbit.At(time.Second).Origin, eighth)
	near(t, "a quarter on", orbit.At(pair*time.Second).Origin, quarter)
	// the plane is square to the axis all the way round
	tolerance := picked(t, "orbit-worked-tolerance")
	for k := range pickedCount(t, "orbit-worked-tilt-steps") {
		at := orbit.At(time.Duration(k) * time.Second).Origin
		if along := at.Dot(axis); !(math.Abs(along) <= tolerance) {
			t.Errorf("at %d s the orbit left its plane: %v along its axis",
				k, along)
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
	y := picked(t, "orbit-worked-axis-y")
	along := picked(t, "orbit-worked-axis-along-x")
	axis := maths.V(along, y, nought)
	if axis.Unit().Dot(maths.XAxis) != along {
		t.Fatalf("the axis is %v along x, not 0.9 exactly",
			axis.Unit().Dot(maths.XAxis))
	}
	period := orbitWorkedSeconds(t, "orbit-worked-short-period")
	orbit := Orbit{Radius: unity, Axis: axis, Period: period}
	near(t, "the start", orbit.At(nought).Origin, maths.V(y, -along, nought))
	quarter := maths.V(nought, nought, -unity)
	near(t, "a quarter on", orbit.At(time.Second).Origin, quarter)
}

// a Phase that is not zero starts the orbit round the circle. about up, from +x
// toward -z, a quarter of the way in is -z: at a radius of 3 and a period of 8
// s, the start is (0, 0, -3), 2 s on is (-3, 0, 0) and 4 s on is (0, 0, 3).
func TestAnOrbitStartsAtItsPhase(t *testing.T) {
	radius := picked(t, "orbit-worked-wide-radius")
	period := orbitWorkedSeconds(t, "orbit-worked-long-period")
	orbit := Orbit{Radius: radius, Period: period, Phase: math.Pi / pair}
	start := maths.V(nought, nought, -radius)
	later := maths.V(-radius, nought, nought)
	half := maths.V(nought, nought, radius)
	near(t, "the start", orbit.At(nought).Origin, start)
	near(t, "2 s on", orbit.At(pair*time.Second).Origin, later)
	near(t, "4 s on", orbit.At(period/pair).Origin, half)
	still := Orbit{Radius: radius, Phase: math.Pi / pair}
	near(t, "with no period", still.At(time.Hour).Origin, start)
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
	radius := picked(t, "orbit-worked-small-radius")
	period := orbitWorkedSeconds(t, "orbit-worked-short-period")
	orbit := Orbit{Radius: radius, Period: period, Phase: math.Pi / pair,
		Facing: maths.Rotation(maths.XAxis, math.Pi/pair), Locked: true}
	start := orbit.At(nought).Orientation
	quarter := orbit.At(time.Second).Orientation
	near(t, "y at the start", start.Out(maths.YAxis), maths.ZAxis)
	near(t, "z at the start", start.Out(maths.ZAxis), maths.YAxis.Neg())
	near(t, "x at the start", start.Out(maths.XAxis), maths.XAxis)
	near(t, "y a quarter on", quarter.Out(maths.YAxis), maths.XAxis)
	near(t, "x a quarter on", quarter.Out(maths.XAxis), maths.ZAxis.Neg())
	near(t, "z a quarter on", quarter.Out(maths.ZAxis), maths.YAxis.Neg())
	// and it is where the phase puts it, still
	near(t, "the start", orbit.At(nought).Origin,
		maths.V(nought, nought, -radius))
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
	turn := maths.Rotation(maths.Up, math.Pi/pair)
	parentScale := picked(t, "orbit-worked-parent-scale")
	place := orbitWorkedPoint(t, "place")
	parent := maths.NewSubspace(place, turn, parentScale)
	unturned := maths.Local().Orientation
	childScale := picked(t, "orbit-worked-child-scale")
	child := maths.NewSubspace(maths.XAxis, unturned, childScale)
	nested := Nested{Of: Still(parent), In: child}
	for _, at := range []time.Duration{nought, time.Hour} {
		orbitWorkedCheckNested(t, nested.At(at))
	}
}

// orbitWorkedCheckNested checks the worked child of the nested test: where
// it stands, how big it is, how it is turned, and the points it takes out
// and back in.
func orbitWorkedCheckNested(t *testing.T, space maths.Subspace) {
	t.Helper()
	near(t, "the child's origin", space.Origin,
		orbitWorkedPoint(t, "child-origin"))
	if want := picked(t, "orbit-worked-nested-scale"); space.Scale != want {
		t.Errorf("the child's scale is %v, want %v", space.Scale, want)
	}
	turned := space.Orientation
	near(t, "the child's x", turned.Out(maths.XAxis),
		orbitWorkedPoint(t, "child-x"))
	near(t, "the child's z", turned.Out(maths.ZAxis), maths.XAxis)
	near(t, "the child's y", turned.Out(maths.YAxis), maths.YAxis)
	sideways := orbitWorkedPoint(t, "child-x-out")
	raised := orbitWorkedPoint(t, "child-y-out")
	near(t, "(1, 0, 0) out", space.Out(maths.XAxis), sideways)
	near(t, "(0, 1, 0) out", space.Out(maths.YAxis), raised)
	near(t, "(1, 2, -5) in", space.In(sideways), maths.XAxis)
	near(t, "(1, 8, 1) in", space.In(raised), maths.YAxis)
}

// a nested motion with no parent is its child, as it is.
func TestNestedWithNoParentIsTheChild(t *testing.T) {
	turn := maths.Rotation(maths.XAxis, unity)
	scale := picked(t, "orbit-worked-child-scale")
	child := maths.NewSubspace(orbitWorkedPoint(t, "place"), turn, scale)
	if got := (Nested{In: child}).At(time.Hour); got != child {
		t.Errorf("a nested motion with no parent is %v, want %v", got, child)
	}
}

// a growing subspace is a scale of 1e-12 before its birth and at it,
// and from then Rate a second: worked by hand, at a rate of 3 a
// quarter second on is 0.75, a second on at a rate of one is exactly
// one, and it grows in a parent of scale 2 at (1, 2, 3) to 2 * 6 = 12.
func TestGrowingScaleIsRateTimesAge(t *testing.T) {
	rate := picked(t, "orbit-worked-growth-rate")
	growing := Growing{From: time.Second, Rate: rate}
	for _, row := range readSheet(t, orbitWorkedGrowing) {
		at := time.Duration(cellCount(t, row, "milliseconds")) *
			time.Millisecond
		want := cell(t, row, "scale")
		if got := growing.At(at).Scale; got != want {
			t.Errorf("at %v the scale is %v, want %v", at, got, want)
		}
	}
	one := Growing{From: time.Second, Rate: unity}
	if got := one.At(pair * time.Second).Scale; got != unity {
		t.Errorf("a second on at a rate of one the scale is %v, want 1", got)
	}
	orbitWorkedGrowingParents(t, rate)
}

// orbitWorkedGrowingParents checks a growing subspace with no parent, which
// neither moves nor turns, and one carried by a parent.
func orbitWorkedGrowingParents(t *testing.T, rate float64) {
	t.Helper()
	zero := maths.Local()
	got := (Growing{Rate: rate}).At(time.Second)
	if got.Origin != zero.Origin || got.Orientation != zero.Orientation {
		t.Errorf("with no parent it moved or turned: %v", got)
	}
	place := orbitWorkedPoint(t, "place")
	scale := picked(t, "orbit-worked-parent-scale")
	parent := maths.NewSubspace(place, zero.Orientation, scale)
	carried := Growing{Of: Still(parent), From: time.Second, Rate: rate}
	space := carried.At(orbitWorkedSeconds(t, "orbit-worked-carried-seconds"))
	want := picked(t, "orbit-worked-carried-scale")
	if space.Scale != want || space.Origin != place {
		t.Errorf("carried by a parent it is at %v, scale %v, "+
			"want (1, 2, 3), 12", space.Origin, space.Scale)
	}
}
