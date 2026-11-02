package maths_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// Boxes are not met by the rays hits.csv says miss. One is beside the
// box. One is along a face outside. One is behind. One is too short.
// Every one with something in it that is no number misses too.
func TestABoxIsNotMetWhereARayDoesNotCrossIt(test *testing.T) {
	for _, row := range read(test, hitsFile) {
		met, err := strconv.ParseBool(row["ok"])
		if err != nil {
			test.Fatal(err)
		}
		if met {
			continue
		}
		if enter, left, ok := hit(test, row); ok {
			test.Errorf("%s: met, from %v to %v", row["name"], enter, left)
		}
	}
}

// Inside the box, a ray along no axis stays inside only for a near and a
// far that are in order.
func TestAStillRayMeetsNothingBetweenAFarBeforeANear(test *testing.T) {
	still := maths.Ray{At: vector(test, "origin"), Dir: vector(test, "origin")}
	box := unitBox(test)
	if _, _, ok := box.Hit(still, picked(test, "coordinate-most"), nought); ok {
		test.Error("a still ray met the box between a near past its far")
	}
}

// Nothing with no direction is a turn. That covers a turn about nothing,
// an up of nothing and looking at where you stand.
func TestNoDirectionIsNoTurn(test *testing.T) {
	nothing, place := vector(test, "origin"), vector(test, "a")
	turn := picked(test, "turn-radians")
	if got := maths.Rotation(nothing, turn); got != maths.Identity {
		test.Errorf("a turn about nothing is %v", got)
	}
	if got := maths.Upright(nothing); got != maths.Identity {
		test.Errorf("standing on nothing is %v", got)
	}
	if got := maths.LookAt(place, place); got != maths.Identity {
		test.Errorf("looking at yourself is %v", got)
	}
}

// The up of nothing is the world's.
func TestAnUpOfNothingIsTheWorlds(test *testing.T) {
	from, to := vector(test, "eye"), vector(test, "target")
	got := maths.LookAtUp(from, to, vector(test, "origin"))
	if want := maths.LookAt(from, to); got != want {
		test.Errorf("an up of nothing is %v, not the world's %v", got, want)
	}
}

// What Len measures as zero has no direction, as nothing has none. That holds
// for exact zero and for the vectors too short for Len to measure. Vectors Len
// can measure still have one.
func TestTooShortToMeasureIsNoDirection(test *testing.T) {
	eye, target := vector(test, "eye"), vector(test, "target")
	turn, origin := picked(test, "turn-radians"), vector(test, "origin")
	for _, name := range []string{"origin", "a hair", "short"} {
		none := vector(test, name)
		if got := maths.Rotation(none, turn); got != maths.Identity {
			test.Errorf("%s: a turn about it is %v", name, got)
		}
		if got := maths.Upright(none); got != maths.Identity {
			test.Errorf("%s: standing on it is %v", name, got)
		}
		if got, want := maths.LookAtUp(eye, target, none), maths.LookAt(eye, target); got != want {
			test.Errorf("%s: an up of it is %v, not the world's %v", name, got, want)
		}
		if got := maths.LookAtUp(none, origin, maths.Up); got != maths.Identity {
			test.Errorf("%s: looking at a place that near is %v", name, got)
		}
	}
	barely := vector(test, "barely measured")
	if got := maths.Rotation(barely, turn); got == maths.Identity {
		test.Error("a turn about a vector Len just measures is no turn")
	}
	if got := maths.LookAtUp(barely, origin, maths.Up); got == maths.Identity {
		test.Error("looking from a place Len just measures is no turn")
	}
	long := vector(test, "measured in full")
	nearTurn(test, "standing on a measurable x", maths.Upright(long),
		orientation(test, "standing on x"))
	if got, world := maths.LookAtUp(eye, target, long), maths.LookAt(eye, target); got == world {
		test.Error("an up Len can measure was read as the world's")
	}
}

// Looking from a place to itself is no turn. That holds for infinity, where the
// two subtract to no number. Len of no number is not zero, so only the test for
// equal places gives the identity here.
func TestLookingFromInfinityToItselfIsNoTurn(test *testing.T) {
	place := vector(test, "infinite on x")
	if got := maths.LookAtUp(place, place, maths.Up); got != maths.Identity {
		test.Errorf("looking from %v to itself is %v", place, got)
	}
	if got := maths.LookAtUp(place, vector(test, "origin"), maths.Up); got == maths.Identity {
		test.Error("looking from infinity to the origin is no turn")
	}
}

// the zero orientation is no turn, not a collapse of every direction to
// nothing.
func TestTheZeroOrientationIsNotACollapse(test *testing.T) {
	direction := vector(test, "a")
	var zero maths.Orientation
	exactly(test, "out", zero.Out(direction), direction)
	exactly(test, "in", zero.In(direction), direction)
	exactly(test, "then", zero.Then(zero).Out(direction), direction)
}

// Subspaces that say no scale are one unit. They do not divide by zero.
func TestASubspaceOfNoScaleIsOneUnit(test *testing.T) {
	empty := maths.Subspace{Origin: vector(test, "origin")}
	point := vector(test, "a point")
	exactly(test, "out", empty.Out(point), point)
	exactly(test, "in", empty.In(point), point)
	if got := empty.Within(empty).Scale; got != unity {
		test.Errorf("an empty space in an empty space is scaled %v", got)
	}
}

// Turning by no number gives no numbers. So does turning about one. The
// fault shows where it is used and is not hidden as no turn.
func TestATurnByNoNumberIsNoNumbers(test *testing.T) {
	up, nan := maths.Up, math.NaN()
	for name, turn := range map[string]maths.Orientation{
		"by NaN":         maths.Rotation(up, nan),
		"by infinity":    maths.Rotation(up, plusInf()),
		"about NaN":      maths.Rotation(vector(test, "nan axis"), unity),
		"about infinity": maths.Rotation(vector(test, "infinite axis"), unity),
	} {
		if !math.IsNaN(turn.X.X) {
			test.Errorf("a turn %s is %v", name, turn)
		}
	}
}

// past what a float can square a length is endless and a direction is zero;
// these are the limits Len says.
func TestPastAFloatALengthIsEndlessAndADirectionIsNought(test *testing.T) {
	long := vector(test, "long")
	if got := long.Len(); !math.IsInf(got, unity) {
		test.Errorf("%v is %v long", long, got)
	}
	exactly(test, "direction", long.Unit(), vector(test, "origin"))
}

// Len at the small end gives the bits recorded in shorts.csv.
// The wants were measured with Len itself, then checked against the
// figures in Len's doc. The test holds the code to those bits.
func TestLenAtTheSmallEndIsAsMeasured(test *testing.T) {
	for _, row := range read(test, shortsFile) {
		x, want := float(test, row, "x"), float(test, row, "want")
		if got := maths.V(x, nought, nought).Len(); got != want {
			test.Errorf("%s: Len of %v is %v, want %v", row["name"], x, got, want)
		}
	}
}

// below what a float can square a length is zero and a direction is the vector
// as it was.
func TestBelowAFloatALengthIsNoughtAndADirectionIsUnchanged(test *testing.T) {
	short := vector(test, "short")
	if got := short.Len(); got != nought {
		test.Errorf("%v is %v long", short, got)
	}
	exactly(test, "direction", short.Unit(), short)
}

// nothing points anywhere.
func TestNothingPointsNowhere(test *testing.T) {
	nothing := vector(test, "origin")
	exactly(test, "direction", nothing.Unit(), nothing)
}

// two points a tenth apart are not within a little less than a tenth.
func TestNearIsNotMoreThanWithin(test *testing.T) {
	origin, along := vector(test, "origin"), vector(test, "a tenth along x")
	if origin.Near(along, picked(test, "near-not-enough")) {
		test.Error("two points a tenth apart are near a little under a tenth")
	}
}

// Two vectors along one line have no cross product direction to give.
func TestParallelVectorsHaveNoCrossProduct(test *testing.T) {
	first := vector(test, "a")
	doubled := first.Scale(picked(test, "doubling"))
	exactly(test, "cross", first.Cross(doubled), vector(test, "origin"))
}

// the axes are not changed by working with them: the vectors Add and
// Scale make are new ones.
func TestWorkingWithAnAxisLeavesItAlone(test *testing.T) {
	_ = maths.XAxis.Add(maths.YAxis).Scale(picked(test, "doubling")).Neg()
	exactly(test, "x", maths.XAxis, vector(test, "one on x"))
	exactly(test, "y", maths.YAxis, vector(test, "one on y"))
	exactly(test, "z", maths.ZAxis, vector(test, "up from z"))
}

// The step does not go past zero or one anywhere. Given a start above its end
// it is a plain step at the start, not a fall.
func TestSmoothNeverLeavesTheUnit(test *testing.T) {
	rapid.Check(test, smoothStaysIn)
}

func smoothStaysIn(test *rapid.T) {
	most := picked(test, "coordinate-most")
	start := rapid.Float64Range(-most, most).Draw(test, "start")
	end := rapid.Float64Range(-most, most).Draw(test, "end")
	x := rapid.Float64Range(-most, most).Draw(test, "x")
	if got := maths.Smooth(start, end, x); got < nought || got > unity {
		test.Fatalf("Smooth(%v, %v, %v) is %v", start, end, x, got)
	}
	if start >= end {
		want := float64(unity)
		if x <= start {
			want = nought
		}
		if got := maths.Smooth(start, end, x); got != want {
			test.Fatalf("a step at %v given the end %v is %v at %v", start, end, got, x)
		}
	}
}

// light has no ceiling: white scaled by a thousand twice is not clipped.
func TestLightHasNoCeiling(test *testing.T) {
	big := picked(test, "coordinate-most")
	got := maths.Grey(big).Scale(big)
	if got.R != big*big || got.G != big*big || got.B != big*big {
		test.Errorf("light was clipped to %v", got)
	}
}

// light scaled by zero is none, and so is light put through nothing.
func TestScalingLightByNothingLeavesNone(test *testing.T) {
	got := light(test, "a").Scale(nought)
	if got != maths.Grey(nought) {
		test.Errorf("light scaled by nought is %v", got)
	}
	if got := light(test, "a").Times(maths.Grey(nought)); got != maths.Grey(nought) {
		test.Errorf("light through nothing is %v", got)
	}
}

// looking straight along the only up given, a camera turns neither to
// nothing nor to numbers that are not: its axes are each one long and at
// right angles to the others.
func TestLookingStraightAlongAnUpIsStillAnOrientation(test *testing.T) {
	look := maths.LookAtUp(vector(test, "overhead"), vector(test, "origin"),
		vector(test, "up from z"))
	for _, axis := range []maths.Vec{look.X, look.Y, look.Z} {
		if !closeTo(test, axis.Len(), unity) {
			test.Errorf("an axis of %v is %v long", look, axis.Len())
		}
	}
	if !closeTo(test, look.X.Dot(look.Y), nought) || !closeTo(test, look.Y.Dot(look.Z), nought) ||
		!closeTo(test, look.X.Dot(look.Z), nought) {
		test.Errorf("the axes of %v are not at right angles", look)
	}
}

// Turning about nothing leaves every direction where it was.
func TestATurnAboutNothingMovesNothing(test *testing.T) {
	direction := vector(test, "a")
	turn := maths.Rotation(vector(test, "origin"), picked(test, "turn-radians"))
	exactly(test, "out", turn.Out(direction), direction)
}

// Rays are at their start after no distance, whichever way they point.
func TestARayAtNoDistanceIsAtItsStart(test *testing.T) {
	for _, way := range []string{"down", "a", "slanted way"} {
		ray := maths.Ray{At: vector(test, "b"), Dir: vector(test, way)}
		exactly(test, "point along "+way, ray.Point(nought), ray.At)
	}
}

// The direction of a vector with an infinite part is no number in that part and
// zero in the others. The infinite axis is infinite on x only, so x is no
// number and y and z are zero.
func TestAnInfiniteDirectionIsNoNumber(test *testing.T) {
	got := vector(test, "infinite axis").Unit()
	if !math.IsNaN(got.X) || got.Y != nought || got.Z != nought {
		test.Errorf("the direction of an infinite vector is %v, want no number, 0, 0", got)
	}
}

// Boxes with a corner that is no number are met by nothing.
func TestABoxWithNoNumberForACornerMeetsNothing(test *testing.T) {
	box := maths.Box{Min: vector(test, "nan corner"), Max: vector(test, "box max")}
	ray := maths.Ray{At: vector(test, "high above"), Dir: vector(test, "down")}
	if _, _, ok := box.Hit(ray, nought, plusInf()); ok {
		test.Error("a box with a corner of no number was met")
	}
}

// Scales that are no number are not read as one.
func TestASubspaceOfNoNumberForAScaleGivesNoNumbers(test *testing.T) {
	space := maths.Subspace{Scale: math.NaN()}
	got := space.Out(vector(test, "a"))
	if !math.IsNaN(got.X) {
		test.Errorf("a scale of no number gave %v", got)
	}
}

// Steps of no number are no number.
func TestSmoothOfNoNumberIsNoNumber(test *testing.T) {
	got := maths.Smooth(nought, unity, math.NaN())
	if !math.IsNaN(got) {
		test.Errorf("the step at no number is %v", got)
	}
}

// Light of no number does not look like any amount.
func TestLightOfNoNumberHasNoLuminance(test *testing.T) {
	got := maths.RGB(math.NaN(), nought, nought).Luminance()
	if !math.IsNaN(got) {
		test.Errorf("light of no number looks %v", got)
	}
}

// Standing on no number is not standing on nothing.
func TestUprightOnNoNumberIsNotNoTurn(test *testing.T) {
	got := maths.Upright(vector(test, "nan axis"))
	if !math.IsNaN(got.Y.X) || !math.IsNaN(got.Y.Y) || !math.IsNaN(got.Y.Z) {
		test.Errorf("standing on no number gave y %v, want no number", got.Y)
	}
}

// Facing no number is not facing where you stand.
func TestLookingAtNoNumberIsNotNoTurn(test *testing.T) {
	got := maths.LookAt(vector(test, "origin"), vector(test, "nan axis"))
	if !math.IsNaN(got.Z.X) || !math.IsNaN(got.Z.Y) || !math.IsNaN(got.Z.Z) {
		test.Errorf("looking at no number gave back %v, want no number", got.Z)
	}
}
