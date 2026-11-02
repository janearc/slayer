package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// V keeps its parts in order, and so does RGB.
func TestPartsKeepTheirPlaces(test *testing.T) {
	first := maths.V(picked(test, "half"), picked(test, "doubling"), picked(test, "within"))
	if first.X != picked(test, "half") || first.Y != picked(test, "doubling") || first.Z != picked(test, "within") {
		test.Errorf("V swapped its parts: %v", first)
	}
	lamp := maths.RGB(picked(test, "half"), picked(test, "doubling"), picked(test, "within"))
	if lamp.R != picked(test, "half") || lamp.G != picked(test, "doubling") || lamp.B != picked(test, "within") {
		test.Errorf("RGB swapped its parts: %v", lamp)
	}
}

// adding is not subtracting, and a vector is not its own negative.
func TestAddingIsNotSubtracting(test *testing.T) {
	first, second := vector(test, "a"), vector(test, "b")
	if first.Add(second) == first.Sub(second) {
		test.Errorf("add and sub agree on %v and %v", first, second)
	}
	if first.Neg() == first {
		test.Errorf("%v is its own negative", first)
	}
}

// scaling by one leaves a vector alone, and by zero leaves nothing.
func TestScalingByOneIsNothingAndByNoughtIsEverything(test *testing.T) {
	first := vector(test, "a")
	exactly(test, "by one", first.Scale(unity), first)
	exactly(test, "by nought", first.Scale(nought), vector(test, "origin"))
}

// the axes are at right angles, not along one another.
func TestTheAxesAreAtRightAngles(test *testing.T) {
	if maths.XAxis.Dot(maths.YAxis) != nought || maths.YAxis.Dot(maths.ZAxis) != nought {
		test.Error("the axes are not at right angles")
	}
	if maths.XAxis.Dot(maths.XAxis) != unity {
		test.Error("an axis is not one long")
	}
}

// the cross product runs one way: swapped, it is the opposite.
func TestCrossProductOfSwappedVectorsIsOpposite(test *testing.T) {
	first, second := vector(test, "a"), vector(test, "b")
	if first.Cross(second) == second.Cross(first) {
		test.Error("the cross product does not mind the order")
	}
}

// Lengths are never negative. Each vector is as long as its negative.
func TestALengthIsNeverNegative(test *testing.T) {
	first := vector(test, "a")
	if first.Len() < nought || first.Len() != first.Neg().Len() {
		test.Errorf("%v is %v long and its negative %v", first, first.Len(), first.Neg().Len())
	}
}

// Turned inside out, a box is met only by rays that cross it slantwise.
// Rays straight down through it do not meet it.
func TestABoxTurnedInsideOutIsNotMetStraightOn(test *testing.T) {
	inverted := maths.Box{Min: vector(test, "box max"), Max: vector(test, "box min")}
	ray := maths.Ray{At: vector(test, "high above"), Dir: vector(test, "down")}
	if _, _, ok := inverted.Hit(ray, nought, plusInf()); ok {
		test.Error("a ray straight down met a box turned inside out")
	}
}

// Starting inside, a ray enters at near and not before it.
func TestARayFromInsideEntersAtNear(test *testing.T) {
	ray := maths.Ray{At: vector(test, "origin"), Dir: vector(test, "down")}
	near := picked(test, "half")
	enter, _, ok := unitBox(test).Hit(ray, near, plusInf())
	if !ok || enter != near {
		test.Errorf("entered at %v, want %v", enter, near)
	}
}

// the box of no edge is not a box of any size: a ray from far away still
// meets it.
func TestTheBoxOfNoEdgeIsNotAUnitBox(test *testing.T) {
	ray := maths.Ray{At: vector(test, "beyond everything"), Dir: vector(test, "down")}
	if _, _, ok := maths.Everywhere().Hit(ray, nought, plusInf()); !ok {
		test.Error("a ray from far away missed a box with no edge")
	}
	if _, _, ok := unitBox(test).Hit(ray, nought, plusInf()); ok {
		test.Error("a ray from far away met a box of one unit")
	}
}

// Quarter turns are turns. They are not the identity and they do not go
// the left-handed way.
func TestAQuarterTurnIsNotNoTurnAndNotLeftHanded(test *testing.T) {
	turn := turned(test, "up from z", quarter)
	if turn == maths.Identity {
		test.Error("a quarter turn is no turn")
	}
	if turn.Out(maths.XAxis).Near(maths.YAxis.Neg(), within(test)) {
		test.Error("a quarter turn about z went the left-handed way")
	}
}

// the order of two turns matters.
func TestTurnsDoNotCommute(test *testing.T) {
	aboutZ, aboutX := turned(test, "up from z", quarter), turned(test, "one on x", quarter)
	first, second := aboutZ.Then(aboutX), aboutX.Then(aboutZ)
	if first.X.Near(second.X, within(test)) && first.Y.Near(second.Y, within(test)) {
		test.Error("two turns about different axes commute")
	}
}

// in is not out: for a turn it is the way back.
func TestInIsNotOut(test *testing.T) {
	turn, direction := turned(test, "up from z", quarter), vector(test, "one on x")
	if turn.In(direction).Near(turn.Out(direction), within(test)) {
		test.Error("in and out agree on a quarter turn")
	}
}

// looking at the target from the eye is a turn, not the identity.
func TestLookingElsewhereIsATurn(test *testing.T) {
	if maths.LookAt(vector(test, "eye"), vector(test, "target")) == maths.Identity {
		test.Error("looking at the target is no turn")
	}
}

// standing on something other than up is a turn.
func TestStandingOnSomethingElseIsATurn(test *testing.T) {
	if maths.Upright(vector(test, "upright up")) == maths.Identity {
		test.Error("standing on a slanted up is no turn")
	}
}

// Any subspace with an origin moves a point.
func TestAnOriginMovesAPoint(test *testing.T) {
	moved := maths.Subspace{Origin: vector(test, "a")}
	point := vector(test, "a point")
	if moved.Out(point) == point || moved.In(point) == point {
		test.Error("a subspace with an origin left a point where it was")
	}
}

// Larger subspaces make distances longer going out, not shorter.
func TestADoubleSpaceMakesDistancesLongerGoingOut(test *testing.T) {
	doubled := maths.Subspace{Scale: picked(test, "doubling")}
	point := vector(test, "a")
	if doubled.Out(point).Len() <= point.Len() || doubled.In(point).Len() >= point.Len() {
		test.Error("a double space shrank going out or grew coming in")
	}
}

// Taken into a subspace, a ray keeps its direction one unit long.
func TestARayTakenInKeepsItsDirectionOneLong(test *testing.T) {
	doubled := maths.Subspace{Scale: picked(test, "doubling"),
		Orientation: turned(test, "up from z", quarter)}
	ray := maths.Ray{At: vector(test, "a"), Dir: maths.XAxis}
	if got := doubled.RayIn(ray).Dir.Len(); !closeTo(test, got, unity) {
		test.Errorf("the direction is %v long inside", got)
	}
}

// Subspaces face their minus z, not their z.
func TestASubspaceDoesNotFacePlusZ(test *testing.T) {
	facing := maths.Subspace{}.Forward()
	if facing == maths.ZAxis {
		test.Error("a subspace faces plus z")
	}
}

// Steps with no width are zero up to the start and one past it.
func TestAStepOfNoWidthIsStillAStep(test *testing.T) {
	start := picked(test, "half")
	if maths.Smooth(start, start, start) != nought || maths.Smooth(start, start, start+start) != unity {
		test.Error("a step of no width is not a step")
	}
}

// Green weighs most and blue least. Red is between.
func TestTheEyeWeighsGreenMostAndBlueLeast(test *testing.T) {
	red := maths.RGB(unity, nought, nought).Luminance()
	green := maths.RGB(nought, unity, nought).Luminance()
	blue := maths.RGB(nought, nought, unity).Luminance()
	if !(green > red && red > blue) {
		test.Errorf("weights: red %v, green %v, blue %v", red, green, blue)
	}
}

// the weights add to one: white looks as bright as its number.
func TestWhiteLooksAsBrightAsItsNumber(test *testing.T) {
	if got := maths.Grey(unity).Luminance(); !closeTo(test, got, unity) {
		test.Errorf("white looks %v", got)
	}
}

// adding no light is no change, and multiplying by white is none.
func TestNoLightAddsNothingAndWhiteChangesNothing(test *testing.T) {
	first := light(test, "a")
	if first.Add(maths.Grey(nought)) != first || first.Times(maths.Grey(unity)) != first {
		test.Error("nothing added or white multiplied changed the light")
	}
	if first.Add(first) == first.Times(first) {
		test.Error("adding and multiplying agree")
	}
}

// About about one axis is not About about another.
func TestAboutDependsOnTheAxis(test *testing.T) {
	angle := picked(test, "turn-radians")
	if maths.About(maths.XAxis, angle) == maths.About(maths.ZAxis, angle) {
		test.Error("turns about different axes are the same")
	}
}

// no distance is near enough to a point that is not there: a negative
// within takes in nothing, not even the point itself.
func TestNothingIsNearWithinLessThanNought(test *testing.T) {
	first := vector(test, "a")
	if first.Near(first, -picked(test, "half")) {
		test.Error("a point is near itself within less than nought")
	}
}

// Degrees are not radians. Half turns are pi.
func TestADegreeIsASmallPartOfARadian(test *testing.T) {
	if maths.Degree >= unity {
		test.Errorf("a degree is %v radians", maths.Degree)
	}
	if got := quarter * maths.Degree * picked(test, "doubling"); !closeTo(test, got, math.Pi) {
		test.Errorf("a half turn is %v radians", got)
	}
}

// Looking along its own up, a view falls back to another up. It does not
// fall back to nothing.
func TestAViewAlongItsOwnUpFallsBack(test *testing.T) {
	look := maths.LookAtUp(vector(test, "origin"), vector(test, "down"), vector(test, "down"))
	for _, axis := range []maths.Vec{look.X, look.Y, look.Z} {
		if !closeTo(test, axis.Len(), unity) {
			test.Errorf("a view along its own up gave %v", look)
		}
	}
}
