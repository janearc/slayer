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

// what the projection tests count in, each defined
const (
	pair  = 2   // defined: a count of two
	once  = 1   // defined: a call that happens exactly once
	third = 3   // defined: a count of three
	half  = 0.5 // defined: half a share
	ahead = -1  // defined: the sensor's space looks down its minus z
)

// the tables the projection tests are run over
const emptyFile = "testdata/pictures-empty.csv"

// what the projection tests choose, each picked in picked-numbers.yml
var (
	wideLeast    = data.Picked("film-wide-least")
	wideMost     = data.Picked("film-wide-most")
	pixelLeast   = data.Picked("film-pixel-least")
	pixelMost    = data.Picked("film-pixel-most")
	pictureMost  = data.Whole("film-picture-most")
	edgeWidth    = data.Whole("film-edge-width")
	eyeMost      = data.Picked("film-eye-most")
	focalLeast   = data.Picked("film-focal-least")
	focalMost    = data.Picked("film-focal-most")
	hubbleSeen   = data.Picked("film-hubble-degrees")
	hubbleWithin = data.Picked("film-hubble-within")
	bandMost     = data.Picked("film-band-most")
	tight        = data.Picked("tolerance-tight")
	loose        = data.Picked("tolerance-loose")
)

// picture draws the size of a picture, in pixels across and down.
func picture(t *rapid.T) (across, down int) {
	size := rapid.IntRange(whole, pictureMost)
	return size.Draw(t, "across"), size.Draw(t, "down")
}

// pinhole draws a pinhole, with a field and a pixel's shape.
func pinhole(t *rapid.T) film.Pinhole {
	wide := rapid.Float64Range(wideLeast, wideMost).Draw(t, "wide")
	shape := rapid.Float64Range(pixelLeast, pixelMost).Draw(t, "pixel")
	return film.NewPinhole(wide, shape)
}

// Every ray a pinhole makes is one unit long, across pictures of many
// shapes and fields of many widths.
func TestEveryRayOfAPinholeIsOneUnitLong(t *testing.T) {
	rapid.Check(t, rayIsOneUnitLong)
}

func rayIsOneUnitLong(t *rapid.T) {
	glass := pinhole(t)
	across, down := picture(t)
	x := rapid.IntRange(none, across-whole).Draw(t, "x")
	y := rapid.IntRange(none, down-whole).Draw(t, "y")
	got := glass.Ray(x, y, across, down, maths.Band{})
	if length := got.Dir.Len(); math.Abs(length-whole) > tight {
		t.Fatalf("the ray at %d, %d of %d by %d is %v long", x, y,
			across, down, length)
	}
}

// The two halves of a picture mirror each other. Pixels across from each
// other look out at the same angle on either side. Pixels above and below
// each other look up and down alike.
func TestAPinholesPictureMirrorsItself(t *testing.T) {
	rapid.Check(t, pictureMirrors)
}

func pictureMirrors(t *rapid.T) {
	glass := pinhole(t)
	across, down := picture(t)
	x := rapid.IntRange(none, across-whole).Draw(t, "x")
	y := rapid.IntRange(none, down-whole).Draw(t, "y")
	band := maths.Band{}
	here := glass.Ray(x, y, across, down, band).Dir
	over := glass.Ray(across-whole-x, y, across, down, band).Dir
	under := glass.Ray(x, down-whole-y, across, down, band).Dir
	if math.Abs(here.X+over.X) > tight || math.Abs(here.Y-over.Y) >
		tight || math.Abs(here.Y+under.Y) > tight ||
		math.Abs(here.X-under.X) > tight {
		t.Fatalf("%v at %d, %d is mirrored by %v and %v", here, x, y, over,
			under)
	}
}

// The top row looks up, and the left column looks left. Row zero is the top of
// the picture, and up in space is up the picture.
func TestTheTopRowLooksUpAndTheLeftColumnLooksLeft(t *testing.T) {
	rapid.Check(t, cornerLooksUpAndLeft)
}

func cornerLooksUpAndLeft(t *rapid.T) {
	glass := pinhole(t)
	size := rapid.IntRange(pair, pictureMost)
	across, down := size.Draw(t, "across"), size.Draw(t, "down")
	corner := glass.Ray(none, none, across, down, maths.Band{}).Dir
	if !(corner.Y > none && corner.X < none && corner.Z < none) {
		t.Fatalf("the top left of %d by %d looks along %v", across, down,
			corner)
	}
}

// The step between neighbouring columns over the step between
// neighbouring rows is the pixel's own width over its height. Otherwise
// circles are not round.
func TestAPixelsShapeKeepsCirclesRound(t *testing.T) {
	rapid.Check(t, shapeKeepsCirclesRound)
}

// slopes are how far right and up a pixel looks for each unit it looks
// ahead.
func slopes(glass film.Pinhole, across, down, x, y int) (right,
	up float64) {
	ray := glass.Ray(x, y, across, down, maths.Band{})
	return ray.Dir.X / -ray.Dir.Z, ray.Dir.Y / -ray.Dir.Z
}

func shapeKeepsCirclesRound(t *rapid.T) {
	wide := rapid.Float64Range(wideLeast, wideMost).Draw(t, "wide")
	shape := rapid.Float64Range(pixelLeast, pixelMost).Draw(t, "pixel")
	glass := film.NewPinhole(wide, shape)
	across := rapid.IntRange(third, pictureMost).Draw(t, "across")
	down := rapid.IntRange(third, pictureMost).Draw(t, "down")
	x := rapid.IntRange(none, across-pair).Draw(t, "x")
	y := rapid.IntRange(none, down-pair).Draw(t, "y")
	x0, y0 := slopes(glass, across, down, x, y)
	x1, _ := slopes(glass, across, down, x+whole, y)
	_, y1 := slopes(glass, across, down, x, y+whole)
	want := shape
	if got := (x1 - x0) / (y0 - y1); math.Abs(got-want) > loose {
		t.Fatalf("across over down is %v, want %v", got, want)
	}
}

// The edge pixel of a very wide picture looks out at nearly half the
// field. It is short of it by less than a pixel's share of the picture.
func TestTheEdgePixelLooksOutNearlyHalfTheField(t *testing.T) {
	rapid.Check(t, edgeLooksHalfTheField)
}

func edgeLooksHalfTheField(t *rapid.T) {
	wide := rapid.Float64Range(wideLeast, wideMost).Draw(t, "wide")
	glass := film.NewPinhole(wide, none)
	ray := glass.Ray(edgeWidth-whole, none, edgeWidth, whole, maths.Band{})
	got := math.Atan2(ray.Dir.X, -ray.Dir.Z)
	if gap := wide/pair - got; gap < none || gap > whole/float64(edgeWidth) {
		t.Fatalf("the edge pixel of a field of %v looks %v out", wide, got)
	}
}

// The default pinhole sees sixty degrees across, and the middle of a
// picture an odd number of pixels across looks straight ahead.
func TestTheDefaultPinholeSeesSixtyDegrees(t *testing.T) {
	var glass film.Pinhole
	ray := glass.Ray(edgeWidth-whole, none, edgeWidth, whole, maths.Band{})
	out := math.Atan2(ray.Dir.X, -ray.Dir.Z) / maths.Degree
	if want := data.Picked("pinhole-wide-degrees") / pair; want-out < none ||
		want-out > whole/float64(edgeWidth)*whole/maths.Degree {
		t.Errorf("the rightmost pixel looks %v degrees out, want %v", out,
			want)
	}
	middle := glass.Ray(pictureMost, pictureMost, pictureMost*pair+whole,
		pictureMost*pair+whole, maths.Band{})
	if !middle.Dir.Near(maths.V(none, none, ahead), tight) {
		t.Errorf("the middle of a picture looks along %v", middle.Dir)
	}
}

// anti-tests: a picture of no pixels looks straight ahead.
func TestAPictureOfNoPixelsLooksAhead(t *testing.T) {
	for _, row := range table(t, emptyFile) {
		across, down := number(t, row, "width"), number(t, row, "height")
		got := film.Pinhole{}.Ray(none, none, int(across), int(down),
			maths.Band{})
		if got.Dir != maths.V(none, none, ahead) || got.At != (maths.Vec{}) {
			label, _ := row.Text("label")
			t.Errorf("%s looks along %v from %v", label, got.Dir, got.At)
		}
	}
}

// anti-tests: fields that are no field are read as the default. So are
// pixels that have no width. Neither makes a mirrored or a broken picture.
func TestAFieldOrAPixelThatIsNoneIsTheDefault(t *testing.T) {
	rapid.Check(t, oddPinholeIsTheDefault)
}

func oddPinholeIsTheDefault(t *rapid.T) {
	below := rapid.Float64Range(-eyeMost, -pair)
	past := rapid.Float64Range(math.Pi+whole, math.Pi+pair)
	not := rapid.Just(math.NaN())
	halfTurn := rapid.Just(math.Pi)
	wide := rapid.OneOf(below, not, halfTurn, past).Draw(t, "wide")
	shape := rapid.OneOf(below, not).Draw(t, "pixel")
	across, down := picture(t)
	x := rapid.IntRange(none, across-whole).Draw(t, "x")
	y := rapid.IntRange(none, down-whole).Draw(t, "y")
	band := maths.Band{}
	got := film.NewPinhole(wide, shape).Ray(x, y, across, down, band)
	want := film.Pinhole{}.Ray(x, y, across, down, band)
	if got != want {
		t.Fatalf("a field of %v and a pixel of %v looks along %v, not %v",
			wide, shape, got.Dir, want.Dir)
	}
}

// Fields of half a radian are that field and not the default's.
func TestASmallFieldIsItself(t *testing.T) {
	band := maths.Band{}
	got := film.NewPinhole(half, none).Ray(none, none, pair, pair, band)
	if got == (film.Pinhole{}).Ray(none, none, pair, pair, band) {
		t.Error("a field of half a radian is the default's")
	}
}

// eye draws a place for an eye to stand.
func eye(t *rapid.T) maths.Subspace {
	place := rapid.Float64Range(-eyeMost, eyeMost)
	from := maths.V(place.Draw(t, "x"), place.Draw(t, "y"),
		place.Draw(t, "z"))
	depth := math.Abs(place.Draw(t, "ahead z"))
	step := maths.V(place.Draw(t, "ahead x"), place.Draw(t, "up"),
		-whole-depth)
	to := from.Add(step)
	return maths.Subspace{Origin: from, Orientation: maths.LookAt(from, to)}
}

// Look takes the ray the projection makes out of the eye's space. The
// projection is asked once about the pixel and the band. The ray comes
// out where the eye puts it.
func TestLookTakesTheProjectionsRayOutOfTheEyesSpace(t *testing.T) {
	rapid.Check(t, lookTakesRayOut)
}

func lookTakesRayOut(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	glass := mocks.NewMockProjection(ctrl)
	across, down := picture(t)
	x := rapid.IntRange(none, across-whole).Draw(t, "x")
	y := rapid.IntRange(none, down-whole).Draw(t, "y")
	band := maths.Band{Nanometres: rapid.Float64Range(none,
		bandMost).Draw(t, "band")}
	place := rapid.Float64Range(-eyeMost, eyeMost)
	depth := math.Abs(place.Draw(t, "dir z"))
	aim := maths.V(place.Draw(t, "dir x"), place.Draw(t, "dir y"),
		-eyeMost-depth)
	made := maths.Ray{At: maths.V(place.Draw(t, "at x"), none, none),
		Dir: aim.Unit()}
	glass.EXPECT().Ray(x, y, across, down, band).Return(made).Times(once)
	standing := eye(t)
	got := film.Look(standing, glass, x, y, across, down, band)
	want := maths.Ray{At: standing.Out(made.At),
		Dir: standing.Orientation.Out(made.Dir).Unit()}
	if !got.At.Near(want.At, tight) || !got.Dir.Near(want.Dir, tight) {
		t.Fatalf("looking from %v gave %v, want %v", standing, got, want)
	}
}

// Nil projections are pinholes of sixty degrees. The middle of the picture
// looks along the way the eye faces.
func TestLookThroughNothingIsAPinholeAndTheMiddleLooksForward(t *testing.T) {
	rapid.Check(t, lookThroughNothing)
}

func lookThroughNothing(t *rapid.T) {
	standing := eye(t)
	across := rapid.IntRange(none, pictureMost).Draw(t, "across")
	down := rapid.IntRange(none, pictureMost).Draw(t, "down")
	wide, high := across*pair+whole, down*pair+whole
	band := maths.Band{}
	middle := film.Look(standing, nil, across, down, wide, high, band)
	if !middle.Dir.Near(standing.Forward(), tight) ||
		!middle.At.Near(standing.Origin, tight) {
		t.Fatalf("the middle ray %v, forward %v from %v", middle,
			standing.Forward(), standing.Origin)
	}
	plain := film.Look(standing, film.Pinhole{}, none, none, wide, high, band)
	if got := film.Look(standing, nil, none, none, wide, high, band); got != plain {
		t.Fatalf("a nil projection looks along %v, not %v", got, plain)
	}
}
