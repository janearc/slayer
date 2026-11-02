package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// coordinate is a number a vector part is drawn from.
func coordinate(test reporter) *rapid.Generator[float64] {
	most := picked(test, "coordinate-most")
	return rapid.Float64Range(-most, most)
}

// anyVec is a vector of drawn parts.
func anyVec(test *rapid.T, label string) maths.Vec {
	part := coordinate(test)
	return maths.V(part.Draw(test, label+" x"), part.Draw(test, label+" y"),
		part.Draw(test, label+" z"))
}

// aDirection is a vector long enough to point somewhere.
func aDirection(test *rapid.T, label string) maths.Vec {
	for {
		got := anyVec(test, label)
		if got.Len() > picked(test, "length-least") {
			return got
		}
		label += " again"
	}
}

// anAngle is an angle in radians, a little over a whole turn each way.
func anAngle(test *rapid.T, label string) float64 {
	most := picked(test, "angle-most")
	return rapid.Float64Range(-most, most).Draw(test, label)
}

// aScale is a scale greater than zero, from the reciprocal of the most up to
// the most.
func aScale(test *rapid.T, label string) float64 {
	most := picked(test, "scale-most")
	return rapid.Float64Range(unity/most, most).Draw(test, label)
}

// closeTo is whether two numbers are within the relative tolerance of
// each other, as a share of the larger of them and of one.
func closeTo(test reporter, got, want float64) bool {
	larger := math.Max(abs(got), abs(want))
	size := math.Max(unity, larger)
	return abs(got-want) <= picked(test, "relative")*size
}

// vecClose is whether two vectors are close part by part.
func vecClose(test reporter, got, want maths.Vec) bool {
	return closeTo(test, got.X, want.X) && closeTo(test, got.Y, want.Y) &&
		closeTo(test, got.Z, want.Z)
}

func TestAdditionCommutesAndSubtractionUndoesIt(test *testing.T) {
	rapid.Check(test, additionCommutes)
}

func additionCommutes(test *rapid.T) {
	first, second := anyVec(test, "a"), anyVec(test, "b")
	if first.Add(second) != second.Add(first) {
		test.Fatalf("%v plus %v differs from %v plus %v", first, second, second, first)
	}
	if got := first.Add(second).Sub(second); !vecClose(test, got, first) {
		test.Fatalf("%v plus %v less %v is %v", first, second, second, got)
	}
	if first.Neg().Neg() != first {
		test.Fatalf("minus minus %v is %v", first, first.Neg().Neg())
	}
}

func TestADotProductCommutes(test *testing.T) {
	rapid.Check(test, dotCommutes)
}

func dotCommutes(test *rapid.T) {
	first, second := anyVec(test, "a"), anyVec(test, "b")
	if first.Dot(second) != second.Dot(first) {
		test.Fatalf("%v dot %v is not %v dot %v", first, second, second, first)
	}
}

func TestACrossProductIsAtRightAnglesToBoth(test *testing.T) {
	rapid.Check(test, crossIsSquare)
}

func crossIsSquare(test *rapid.T) {
	first, second := anyVec(test, "a"), anyVec(test, "b")
	cross := first.Cross(second)
	if !vecClose(test, cross, second.Cross(first).Neg()) {
		test.Fatalf("%v cross %v is not minus the other way", first, second)
	}
	size := first.Len() * second.Len() * math.Max(first.Len(), second.Len())
	if abs(cross.Dot(first)) > picked(test, "relative")*math.Max(unity, size) ||
		abs(cross.Dot(second)) > picked(test, "relative")*math.Max(unity, size) {
		test.Fatalf("%v cross %v is %v, not at right angles", first, second, cross)
	}
}

func TestALengthScalesAndADirectionIsOneLong(test *testing.T) {
	rapid.Check(test, lengthScales)
}

func lengthScales(test *rapid.T) {
	first, by := aDirection(test, "a"), aScale(test, "scale")
	if got, want := first.Scale(by).Len(), by*first.Len(); !closeTo(test, got, want) {
		test.Fatalf("%v scaled by %v is %v long, want %v", first, by, got, want)
	}
	if got := first.Unit().Len(); !closeTo(test, got, unity) {
		test.Fatalf("the direction of %v is %v long", first, got)
	}
	if first.Len() < nought {
		test.Fatalf("%v is %v long", first, first.Len())
	}
}

func TestNearIsSymmetricAndReflexive(test *testing.T) {
	rapid.Check(test, nearIsSymmetric)
}

func nearIsSymmetric(test *rapid.T) {
	first, second := anyVec(test, "a"), anyVec(test, "b")
	within := rapid.Float64Range(nought, picked(test, "coordinate-most")).Draw(test, "within")
	if first.Near(second, within) != second.Near(first, within) {
		test.Fatalf("%v near %v within %v depends on the order", first, second, within)
	}
	if !first.Near(first, nought) {
		test.Fatalf("%v is not near itself", first)
	}
}

func TestATurnKeepsLengthsAndAxesSquare(test *testing.T) {
	rapid.Check(test, turnIsSquare)
}

func turnIsSquare(test *rapid.T) {
	axis, angle := aDirection(test, "axis"), anAngle(test, "angle")
	turn := maths.Rotation(axis, angle)
	for _, got := range []maths.Vec{turn.X, turn.Y, turn.Z} {
		if !closeTo(test, got.Len(), unity) {
			test.Fatalf("a turn about %v by %v makes an axis %v long", axis, angle, got.Len())
		}
	}
	if !closeTo(test, turn.X.Dot(turn.Y), nought) || !closeTo(test, turn.Y.Dot(turn.Z), nought) {
		test.Fatalf("a turn about %v by %v is not square: %v", axis, angle, turn)
	}
	if !vecClose(test, turn.X.Cross(turn.Y), turn.Z) {
		test.Fatalf("a turn about %v by %v is left-handed: %v", axis, angle, turn)
	}
	if !vecClose(test, turn.Out(axis), axis) {
		test.Fatalf("a turn about %v moves the axis to %v", axis, turn.Out(axis))
	}
}

func TestInUndoesOutForAnyTurn(test *testing.T) {
	rapid.Check(test, inUndoesOut)
}

func inUndoesOut(test *rapid.T) {
	turn := maths.Rotation(aDirection(test, "axis"), anAngle(test, "angle"))
	direction := anyVec(test, "direction")
	if got := turn.In(turn.Out(direction)); !vecClose(test, got, direction) {
		test.Fatalf("in of out of %v is %v", direction, got)
	}
}

func TestThenIsOneTurnAfterTheOther(test *testing.T) {
	rapid.Check(test, thenIsComposition)
}

func thenIsComposition(test *rapid.T) {
	first := maths.Rotation(aDirection(test, "first axis"), anAngle(test, "first"))
	second := maths.Rotation(aDirection(test, "second axis"), anAngle(test, "second"))
	direction := anyVec(test, "direction")
	got := first.Then(second).Out(direction)
	if want := first.Out(second.Out(direction)); !vecClose(test, got, want) {
		test.Fatalf("then gives %v, one after the other gives %v", got, want)
	}
}

func TestUprightStandsOnAnyUp(test *testing.T) {
	rapid.Check(test, uprightIsSquare)
}

func uprightIsSquare(test *rapid.T) {
	up := aDirection(test, "up")
	got := maths.Upright(up)
	if !vecClose(test, got.Y, up.Unit()) {
		test.Fatalf("upright on %v has y %v", up, got.Y)
	}
	if !closeTo(test, got.X.Len(), unity) || !closeTo(test, got.X.Dot(got.Y), nought) {
		test.Fatalf("upright on %v is not square: %v", up, got)
	}
	if !vecClose(test, got.X.Cross(got.Y), got.Z) {
		test.Fatalf("upright on %v is left-handed: %v", up, got)
	}
}

func TestLookAtFacesTheTargetFromAnywhere(test *testing.T) {
	rapid.Check(test, lookAtFaces)
}

func lookAtFaces(test *rapid.T) {
	from, to := anyVec(test, "from"), anyVec(test, "to")
	if from.Sub(to).Len() < picked(test, "length-least") {
		test.Skip("the two places are too near to face")
	}
	look := maths.LookAt(from, to)
	back := from.Sub(to).Unit()
	if !vecClose(test, look.Z, back) {
		test.Fatalf("looking from %v to %v, back is %v, not %v", from, to, look.Z, back)
	}
	if !closeTo(test, look.X.Len(), unity) || !closeTo(test, look.Y.Len(), unity) ||
		!closeTo(test, look.X.Dot(look.Y), nought) {
		test.Fatalf("looking from %v to %v is not square: %v", from, to, look)
	}
}

func TestInUndoesOutForAnySubspace(test *testing.T) {
	rapid.Check(test, subspaceInUndoesOut)
}

func aSubspace(test *rapid.T) maths.Subspace {
	return maths.Subspace{Origin: anyVec(test, "origin"),
		Orientation: maths.Rotation(aDirection(test, "axis"), anAngle(test, "angle")),
		Scale:       aScale(test, "scale")}
}

func subspaceInUndoesOut(test *rapid.T) {
	space, point := aSubspace(test), anyVec(test, "point")
	if got := space.In(space.Out(point)); !vecClose(test, got, point) {
		test.Fatalf("in of out of %v is %v", point, got)
	}
}

func TestWithinIsOutOfEachInTurn(test *testing.T) {
	rapid.Check(test, withinIsComposition)
}

func withinIsComposition(test *rapid.T) {
	parent, child, point := aSubspace(test), aSubspace(test), anyVec(test, "point")
	got := child.Within(parent).Out(point)
	if want := parent.Out(child.Out(point)); !vecClose(test, got, want) {
		test.Fatalf("within gives %v, out of each gives %v", got, want)
	}
}

func TestADistanceInsideIsScaleTimesItOutside(test *testing.T) {
	rapid.Check(test, rayDistancesScale)
}

func rayDistancesScale(test *rapid.T) {
	space := aSubspace(test)
	ray := maths.Ray{At: anyVec(test, "at"), Dir: aDirection(test, "way").Unit()}
	along := rapid.Float64Range(nought, picked(test, "distance-most")).Draw(test, "along")
	in := space.RayIn(ray)
	got := space.Out(in.Point(along))
	if want := ray.Point(along * space.Scale); !vecClose(test, got, want) {
		test.Fatalf("a distance %v inside is %v, want %v", along, got, want)
	}
}

func TestAHitLiesInsideTheBoxAndBetweenNearAndFar(test *testing.T) {
	rapid.Check(test, hitIsInside)
}

// The box of no edge is met by every ray drawn, from any start and way.
func TestTheBoxOfNoEdgeCatchesEveryDrawnRay(test *testing.T) {
	rapid.Check(test, func(test *rapid.T) {
		ray := maths.Ray{At: anyVec(test, "at"), Dir: aDirection(test, "way").Unit()}
		if _, _, ok := maths.Everywhere().Hit(ray, nought, math.Inf(unity)); !ok {
			test.Fatalf("a ray from %v along %v missed the box of no edge", ray.At, ray.Dir)
		}
	})
}

// Rays aimed at a point inside a box meet it. The start is outside the
// box or inside it, by the draw; both must meet.
// The box is widened a unit each side, so rounding cannot put the point out.
// The point is reached between enter and leave.
func TestARayAimedAtAPointInsideMeetsTheBox(test *testing.T) {
	rapid.Check(test, func(test *rapid.T) {
		first, second := anyVec(test, "a"), anyVec(test, "b")
		low := maths.V(math.Min(first.X, second.X), math.Min(first.Y, second.Y),
			math.Min(first.Z, second.Z))
		high := maths.V(math.Max(first.X, second.X), math.Max(first.Y, second.Y),
			math.Max(first.Z, second.Z))
		box := maths.Box{Min: low.Sub(maths.V(unity, unity, unity)),
			Max: high.Add(maths.V(unity, unity, unity))}
		share := rapid.Float64Range(nought, unity)
		inside := maths.V(
			low.X+share.Draw(test, "x share")*(high.X-low.X),
			low.Y+share.Draw(test, "y share")*(high.Y-low.Y),
			low.Z+share.Draw(test, "z share")*(high.Z-low.Z))
		way := aDirection(test, "way").Unit()
		away := rapid.Float64Range(nought, picked(test, "distance-most")).Draw(test, "away")
		ray := maths.Ray{At: inside.Sub(way.Scale(away)), Dir: way}
		span := rapid.Float64Range(nought, picked(test, "distance-most")).Draw(test, "span")
		enter, leave, ok := box.Hit(ray, nought, away+span)
		if !ok {
			test.Fatalf("a ray from %v aimed at %v inside %v met nothing", ray.At, inside, box)
		}
		slack := picked(test, "relative") * picked(test, "coordinate-most") * picked(test, "distance-most")
		if enter > away+slack || leave < away-slack {
			test.Fatalf("the point is at %v, not between %v and %v", away, enter, leave)
		}
	})
}

func hitIsInside(test *rapid.T) {
	first, second := anyVec(test, "a"), anyVec(test, "b")
	box := maths.Box{
		Min: maths.V(math.Min(first.X, second.X), math.Min(first.Y, second.Y), math.Min(first.Z, second.Z)),
		Max: maths.V(math.Max(first.X, second.X), math.Max(first.Y, second.Y), math.Max(first.Z, second.Z)),
	}
	ray := maths.Ray{At: anyVec(test, "at"), Dir: aDirection(test, "way").Unit()}
	near := rapid.Float64Range(nought, picked(test, "distance-most")).Draw(test, "near")
	far := near + rapid.Float64Range(nought, picked(test, "coordinate-most")).Draw(test, "span")
	enter, leave, ok := box.Hit(ray, near, far)
	if !ok {
		return
	}
	if enter < near || leave > far || enter > leave {
		test.Fatalf("a hit from %v to %v is not inside %v to %v", enter, leave, near, far)
	}
	middle := ray.Point((enter + leave) * picked(test, "half"))
	slack := picked(test, "relative") * picked(test, "coordinate-most") * picked(test, "distance-most")
	if !inBox(box, middle, slack) {
		test.Fatalf("the middle of a hit, %v, is outside %v", middle, box)
	}
}

// inBox is whether a point is in a box, or no further out than slack.
func inBox(box maths.Box, point maths.Vec, slack float64) bool {
	return point.X >= box.Min.X-slack && point.X <= box.Max.X+slack &&
		point.Y >= box.Min.Y-slack && point.Y <= box.Max.Y+slack &&
		point.Z >= box.Min.Z-slack && point.Z <= box.Max.Z+slack
}

func TestSmoothStaysInTheUnitAndNeverFalls(test *testing.T) {
	rapid.Check(test, smoothRises)
}

func smoothRises(test *rapid.T) {
	start := rapid.Float64Range(-picked(test, "coordinate-most"), picked(test, "coordinate-most")).Draw(test, "start")
	width := rapid.Float64Range(picked(test, "length-least"), picked(test, "coordinate-most")).Draw(test, "width")
	end := start + width
	x := rapid.Float64Range(start-width, end+width).Draw(test, "x")
	later := x + rapid.Float64Range(nought, picked(test, "coordinate-most")).Draw(test, "later")
	got, then := maths.Smooth(start, end, x), maths.Smooth(start, end, later)
	if got < nought || got > unity || then < got {
		test.Fatalf("Smooth(%v, %v) is %v at %v and %v at %v", start, end, got, x, then, later)
	}
	mirrored := maths.Smooth(start, end, start+end-x)
	if !closeTo(test, got+mirrored, unity) {
		test.Fatalf("Smooth(%v, %v) at %v and its mirror sum to %v", start, end, x, got+mirrored)
	}
}

func TestLightsAddAndMultiplyInAnyOrder(test *testing.T) {
	rapid.Check(test, lightsCommute)
}

func aLight(test *rapid.T, label string) maths.Light {
	part := coordinate(test)
	return maths.RGB(part.Draw(test, label+" r"), part.Draw(test, label+" g"), part.Draw(test, label+" b"))
}

func lightsCommute(test *rapid.T) {
	first, second, by := aLight(test, "a"), aLight(test, "b"), aScale(test, "scale")
	if first.Add(second) != second.Add(first) || first.Times(second) != second.Times(first) {
		test.Fatalf("%v and %v do not commute", first, second)
	}
	if got, want := first.Add(second).Scale(by), first.Scale(by).Add(second.Scale(by)); !lightClose(test, got, want) {
		test.Fatalf("scaling does not distribute: %v and %v", got, want)
	}
}

// lightClose is whether two lights are close part by part.
func lightClose(test reporter, got, want maths.Light) bool {
	return closeTo(test, got.R, want.R) && closeTo(test, got.G, want.G) && closeTo(test, got.B, want.B)
}

func TestLuminanceIsLinearAndAGreyKeepsItsAmount(test *testing.T) {
	rapid.Check(test, luminanceIsLinear)
}

func luminanceIsLinear(test *rapid.T) {
	first, second, amount := aLight(test, "a"), aLight(test, "b"), coordinate(test).Draw(test, "amount")
	if got, want := first.Add(second).Luminance(), first.Luminance()+second.Luminance(); !closeTo(test, got, want) {
		test.Fatalf("luminance of a sum is %v, sum of luminances %v", got, want)
	}
	if got := maths.Grey(amount).Luminance(); !closeTo(test, got, amount) {
		test.Fatalf("a grey of %v looks %v", amount, got)
	}
}
