package film_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/mocks"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// ninety is half of ninety degrees, picked.
var ninety = data.Picked("film-ninety-degrees")

// anti-tests: no ray is zero or lost. The ray of a pinhole is a number in
// each part at every pixel. It is not the zero vector.
func TestNoRayOfAPinholeIsZeroOrLost(t *testing.T) {
	rapid.Check(t, rayIsAUnitAndNoNumberIsLost)
}

func rayIsAUnitAndNoNumberIsLost(t *rapid.T) {
	glass := pinhole(t)
	across, down := picture(t)
	x := rapid.IntRange(none, across-whole).Draw(t, "x")
	y := rapid.IntRange(none, down-whole).Draw(t, "y")
	got := glass.Ray(x, y, across, down, maths.Band{}).Dir
	if got == (maths.Vec{}) || math.IsNaN(got.X+got.Y+got.Z) {
		t.Fatalf("the ray at %d,%d is %v", x, y, got)
	}
}

// anti-tests: the left of a picture does not look right. Pixels in the
// left half look out to the left. Pixels in the right half do not.
func TestThePartsOfAPictureDoNotLookTheSameWay(t *testing.T) {
	rapid.Check(t, leftLooksLeftAndRightDoesNot)
}

func leftLooksLeftAndRightDoesNot(t *rapid.T) {
	glass := pinhole(t)
	half := rapid.IntRange(whole, pictureMost).Draw(t, "half")
	across, down := pair*half, whole
	x := rapid.IntRange(none, half-whole).Draw(t, "x")
	left := glass.Ray(x, none, across, down, maths.Band{}).Dir
	right := glass.Ray(across-whole-x, none, across, down, maths.Band{}).Dir
	if !(left.X < none) || !(right.X > none) {
		t.Fatalf("pixels %d and %d look %v and %v", x, across-whole-x, left,
			right)
	}
}

// anti-tests: the bottom row does not look up. It looks down, and the
// top row of the same picture does not.
func TestTheBottomRowDoesNotLookUp(t *testing.T) {
	rapid.Check(t, bottomRowLooksDown)
}

func bottomRowLooksDown(t *rapid.T) {
	glass := pinhole(t)
	size := rapid.IntRange(pair, pictureMost)
	across, down := size.Draw(t, "across"), size.Draw(t, "down")
	bottom := glass.Ray(none, down-whole, across, down, maths.Band{}).Dir
	top := glass.Ray(none, none, across, down, maths.Band{}).Dir
	if !(bottom.Y < none) || top.Y < bottom.Y {
		t.Fatalf("the bottom row looks %v and the top %v", bottom, top)
	}
}

// anti-tests: a pixel's shape is not left square. Pixels half as wide as
// they are tall step differently across a picture than square ones.
func TestAPixelsShapeIsNotLeftSquare(t *testing.T) {
	rapid.Check(t, shapeChangesTheStep)
}

func shapeChangesTheStep(t *rapid.T) {
	wide := rapid.Float64Range(wideLeast, wideMost).Draw(t, "wide")
	across := rapid.IntRange(third, pictureMost).Draw(t, "across")
	down := rapid.IntRange(third, pictureMost).Draw(t, "down")
	band := maths.Band{}
	square := film.NewPinhole(wide, whole).Ray(none, none, across, down,
		band)
	thin := film.NewPinhole(wide, half).Ray(none, none, across, down, band)
	if square.Dir.Near(thin.Dir, tight) {
		t.Fatalf("pixels of two shapes looked alike")
	}
}

// anti-tests: the middle of a picture does not look out at the edge of
// the field. Its ray is close to straight ahead.
func TestTheMiddleOfAPictureDoesNotLookOutAtTheEdge(t *testing.T) {
	rapid.Check(t, middleLooksAhead)
}

func middleLooksAhead(t *rapid.T) {
	glass := film.NewPinhole(rapid.Float64Range(wideLeast, wideMost).Draw(t,
		"wide"), none)
	middle := glass.Ray(edgeWidth/pair, none, edgeWidth+whole, whole,
		maths.Band{}).Dir
	if math.Abs(middle.X) > tight || !(middle.Z < none) {
		t.Fatalf("the middle of the picture looks %v", middle)
	}
}

// anti-tests: the default pinhole does not see ninety degrees. Its edge
// pixel looks out at less than forty five.
func TestTheDefaultPinholeDoesNotSeeNinetyDegrees(t *testing.T) {
	var glass film.Pinhole
	ray := glass.Ray(edgeWidth-whole, none, edgeWidth, whole, maths.Band{})
	out := math.Atan2(ray.Dir.X, -ray.Dir.Z) / maths.Degree
	if !(out < ninety) {
		t.Errorf("the rightmost pixel looks %v degrees out", out)
	}
}

// anti-tests: a field of half a radian is not the default. Fields below
// the half turn are kept, and the ray differs from the default's.
func TestAFieldBelowAHalfTurnIsNotTheDefault(t *testing.T) {
	rapid.Check(t, keptFieldIsNotDefault)
}

func keptFieldIsNotDefault(t *rapid.T) {
	wide := rapid.Float64Range(wideLeast, wideMost).Draw(t, "wide")
	var def film.Pinhole
	kept := film.NewPinhole(wide, none).Ray(none, none, pair, pair,
		maths.Band{})
	defaultWide := data.Picked("pinhole-wide-degrees") * maths.Degree
	if math.Abs(wide-defaultWide) > tight &&
		kept == def.Ray(none, none, pair, pair, maths.Band{}) {
		t.Fatalf("a field of %v read as the default", wide)
	}
}

// anti-tests: looking from an eye that is not at the origin does not leave
// the ray where the projection put it. The ray starts at the eye.
func TestLookFromAnEyeThatMovedDoesNotLeaveTheRayAtTheOrigin(t *testing.T) {
	rapid.Check(t, lookStartsAtTheEye)
}

func lookStartsAtTheEye(t *rapid.T) {
	standing := eye(t)
	if standing.Origin.Len() < whole {
		t.Skip("an eye near the origin")
	}
	got := film.Look(standing, film.Pinhole{}, none, none, whole, whole,
		maths.Band{})
	if got.At.Near(maths.Vec{}, tight) {
		t.Fatalf("looking from %v started at the origin", standing.Origin)
	}
}

// anti-tests: a nil projection is not a lens. It is the pinhole of sixty
// degrees, and a narrow lens looks another way at the edge.
func TestANilProjectionIsNotANarrowLens(t *testing.T) {
	rapid.Check(t, nilIsNotALens)
}

func nilIsNotALens(t *rapid.T) {
	standing := eye(t)
	frame := gauged(t, standardName)
	long := film.NewLens(longLensFocal, frame, none)
	across, down := picture(t)
	if across < pair {
		t.Skip("a picture with no side")
	}
	band := maths.Band{}
	plain := film.Look(standing, nil, none, none, across, down, band)
	narrow := film.Look(standing, long, none, none, across, down, band)
	if plain.Dir.Near(narrow.Dir, tight) {
		t.Fatalf("a nil projection looked as a long lens does")
	}
}

// anti-tests: a projection is asked only for the pixel it is looked
// through. Looks through a mock ask it once and no more.
func TestALookAsksTheProjectionOnlyOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	glass := mocks.NewMockProjection(ctrl)
	band := maths.Band{Nanometres: bandMost}
	glass.EXPECT().Ray(none, none, whole, whole, band).Return(maths.Ray{
		Dir: maths.V(none, none, -whole)}).Times(once)
	film.Look(maths.Subspace{}, glass, none, none, whole, whole, band)
}
