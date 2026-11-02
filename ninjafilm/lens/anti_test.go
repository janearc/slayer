package lens_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"pgregory.net/rapid"
)

// listOf is its arguments as a list. It lets a test build a list of
// frames without naming the type of one.
func listOf[Each any](each ...Each) []Each { return each }

// anti-tests: a scene with no light exposes nothing. Every channel of every
// pixel is zero, whatever the time.
func TestADarkSceneExposesNothing(t *testing.T) {
	glass := mustThin(t, focus)
	dark := listOf(lit(pictureMost, pictureLeast, none))
	exposed, err := glass.Expose(time.Second, dark)
	if err != nil {
		t.Fatal(err)
	}
	for index, each := range exposed.Pix {
		if each.R != none || each.G != none || each.B != none {
			t.Errorf("pixel %d of a dark scene is %+v", index, each)
		}
	}
}

// anti-tests: twice the time is not the same light. Pixels that are lit
// take more light in a longer shutter and not the same.
func TestTwiceTheShutterIsNotTheSameLight(t *testing.T) {
	rapid.Check(t, longerShutterIsNotTheSameLight)
}

func longerShutterIsNotTheSameLight(t *rapid.T) {
	glass := mustThin(t, focus)
	columns := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "columns")
	rows := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "rows")
	still := glimpsesOf(t, columns, rows)
	once, err := glass.Expose(time.Second, still)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := glass.Expose(pair*time.Second, still)
	if err != nil {
		t.Fatal(err)
	}
	for index := range once.Pix {
		if once.Pix[index].R > none && twice.Pix[index].R == once.Pix[index].R {
			t.Fatalf("pixel %d took %g in one second and in two", index,
				once.Pix[index].R)
		}
	}
}

// anti-tests: the mean of a changing scene is neither glimpse alone. It
// lies between the exposure of the dim one and the bright one.
func TestAChangingSceneIsNeitherOfItsGlimpsesAlone(t *testing.T) {
	glass := mustThin(t, focus)
	dim, bright := float32(radiance), float32(radiance*pair)
	dimmer, brighter := lit(whole, whole, dim), lit(whole, whole, bright)
	one, err := glass.Expose(time.Second, listOf(dimmer))
	if err != nil {
		t.Fatal(err)
	}
	other, err := glass.Expose(time.Second, listOf(brighter))
	if err != nil {
		t.Fatal(err)
	}
	both, err := glass.Expose(time.Second, listOf(dimmer, brighter))
	if err != nil {
		t.Fatal(err)
	}
	got := both.Pix[none].G
	if !(one.Pix[none].G < got && got < other.Pix[none].G) {
		t.Errorf("the mean %g is not between %g and %g", got,
			one.Pix[none].G, other.Pix[none].G)
	}
}

// anti-tests: an exposure is not the glimpse it was made from. Film
// gathers part of the light, so a lit pixel comes out changed.
func TestAnExposureIsNotTheGlimpseItWasMadeFrom(t *testing.T) {
	glass := mustThin(t, focus)
	seen := lit(whole, whole, float32(radiance))
	exposed, err := glass.Expose(time.Second, listOf(seen))
	if err != nil {
		t.Fatal(err)
	}
	if exposed.Pix[none].G == seen.Pix[none].G {
		t.Errorf("the exposure came out as the glimpse, %g", seen.Pix[none].G)
	}
}

// anti-tests: an exposure of something is not a pixel that met nothing.
// Lit pixels have cover and are a finite way off.
func TestAnExposureOfSomethingMetSomething(t *testing.T) {
	glass := mustThin(t, focus)
	level := float32(radiance)
	seen := listOf(lit(whole, whole, level))
	exposed, err := glass.Expose(time.Second, seen)
	if err != nil {
		t.Fatal(err)
	}
	if got := exposed.Pix[none]; !(got.Cover > none) || math.IsInf(
		float64(got.Depth), upward) {
		t.Errorf("a lit pixel exposed as one that met nothing: %+v", got)
	}
}

// anti-tests: other glimpses make another frame. Two scenes that differ
// do not expose to one frame.
func TestOtherGlimpsesMakeAnotherExposure(t *testing.T) {
	rapid.Check(t, otherGlimpsesMakeAnother)
}

func otherGlimpsesMakeAnother(t *rapid.T) {
	glass := mustThin(t, focus)
	columns := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "columns")
	rows := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "rows")
	first, second := drawnFrame(t, columns, rows), drawnFrame(t, columns, rows)
	one, err := glass.Expose(time.Second, listOf(first))
	if err != nil {
		t.Fatal(err)
	}
	other, err := glass.Expose(time.Second, listOf(second))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(first.Pix, second.Pix) {
		t.Skip("scenes that do not differ")
	}
	if reflect.DeepEqual(one.Pix, other.Pix) {
		t.Fatal("two scenes that differ exposed to one frame")
	}
}

// anti-tests: an exposure's depth is not the farthest any glimpse met,
// and its cover is not the sum of the covers. It is the nearest depth
// and the mean cover.
func TestAnExposureTakesTheNearestDepthAndNotTheFarthest(t *testing.T) {
	rapid.Check(t, depthIsTheNearest)
}

func depthIsTheNearest(t *rapid.T) {
	glass := mustThin(t, focus)
	columns := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "columns")
	rows := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "rows")
	caught := glimpsesOf(t, columns, rows)
	exposed, err := glass.Expose(time.Second, caught)
	if err != nil {
		t.Fatal(err)
	}
	for index, got := range exposed.Pix {
		var most float32
		for _, each := range caught {
			most = max(most, each.Pix[index].Cover)
			if got.Depth > each.Pix[index].Depth {
				t.Fatalf("pixel %d is %g away, a glimpse met %g", index,
					got.Depth, each.Pix[index].Depth)
			}
		}
		if got.Cover > most {
			t.Fatalf("pixel %d has cover %g, more than any glimpse's %g",
				index, got.Cover, most)
		}
	}
}

// sharpOf is the point at the focus distance that the middle of a pixel
// sees through the lens's middle. It is worked from the lens law.
func sharpOf(x, y int) maths.Vec {
	image := imageAt(focal*millimetre, focus)
	wide, high := frameWide*millimetre, frameHigh*millimetre
	right := ((float64(x)+half)/float64(pixelsAcross) - half) * wide
	up := (half - (float64(y)+half)/float64(pixelsDown)) * high
	return maths.V(right*focus/image, up*focus/image, -focus)
}

// anti-tests: a ray does not aim at the sharp point of the pixel beside
// it. The neighbour's point is further off the ray than its own.
func TestARayDoesNotAimAtItsNeighboursSharpPoint(t *testing.T) {
	rapid.Check(t, rayMissesItsNeighboursPoint)
}

func rayMissesItsNeighboursPoint(t *rapid.T) {
	glass := mustThin(t, focus)
	x := rapid.IntRange(none, pixelsAcross-pair).Draw(t, "x")
	y := rapid.IntRange(none, pixelsDown-whole).Draw(t, "y")
	ray := glass.Ray(x, y, pixelsAcross, pixelsDown, maths.Band{})
	if miss := apart(ray, sharpOf(x+whole, y)); miss <= missError*focus {
		t.Fatalf("pixel %d,%d aims at the point of %d,%d", x, y, x+whole, y)
	}
}

// anti-tests: a lens focused nearer does not see wider than one focused
// far off. The film stands further back for the near focus.
func TestANearerFocusDoesNotSeeWiderThanAFarOne(t *testing.T) {
	near, far := mustThin(t, focus), mustThin(t, farFocus)
	if !(near.Wide() < far.Wide()) {
		t.Errorf("focused near it sees %g and far off %g", near.Wide(),
			far.Wide())
	}
}

// anti-tests: rays do not all start at one point. The pixels of an
// aperture start spread across it.
func TestRaysDoNotAllStartAtOnePoint(t *testing.T) {
	glass := mustThin(t, focus)
	seen := map[maths.Vec]bool{}
	for y := range apertureDown {
		for x := range apertureAcross {
			seen[glass.Ray(x, y, apertureAcross, apertureDown,
				maths.Band{}).At] = true
		}
	}
	if len(seen) < apertureAcross {
		t.Errorf("%d rays started at %d points", apertureAcross*apertureDown,
			len(seen))
	}
}

// anti-tests: the lens gathers less toward the corner than on the axis.
// The corner is dimmer by the cosine to the fourth.
func TestTheLensGathersLessAtTheCornerThanOnTheAxis(t *testing.T) {
	glass := mustThin(t, focus)
	axis := glass.Gather(none, none, whole, whole)
	corner := glass.Gather(none, none, pixelsAcross, pixelsDown)
	if !(corner < axis) {
		t.Errorf("the corner gathers %g and the axis %g", corner, axis)
	}
}

// anti-tests: frames of lost or negative sizes are not made, whichever
// side is out. What comes back is no frame.
func TestAFrameOfALostOrNegativeSizeIsNotMade(t *testing.T) {
	rapid.Check(t, frameOfNoSizeIsRefused)
}

func frameOfNoSizeIsRefused(t *rapid.T) {
	lost := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward)})
	below := rapid.Float64Range(oddLeast, none)
	bad := rapid.OneOf(below, lost).Draw(t, "bad")
	good := rapid.Float64Range(millimetresMin, millimetresMax).Draw(t, "good")
	var none lens.Frame
	if got, err := lens.FrameOf(bad, good); err == nil || got != none {
		t.Fatalf("a frame %v wide was made", bad)
	}
	if got, err := lens.FrameOf(good, bad); err == nil || got != none {
		t.Fatalf("a frame %v high was made", bad)
	}
}

// anti-tests: a lens focused no further than its focal length is not
// made, however the rest is. The focus has to be further off.
func TestALensFocusedNoFurtherThanItsFocalLengthIsNotMade(t *testing.T) {
	rapid.Check(t, nearFocusIsRefused)
}

func nearFocusIsRefused(t *rapid.T) {
	length := rapid.Float64Range(focalLeast, focalMost).Draw(t, "focal")
	away := length * millimetre * rapid.Float64Range(none, whole).Draw(t,
		"share")
	frame, err := lens.FrameOf(frameWide, frameHigh)
	if err != nil {
		t.Fatal(err)
	}
	var noLens lens.Thin
	if got, err := lens.NewThin(frame, length, stop, away); err == nil ||
		got != noLens {
		t.Fatalf("a lens of %v at %v was made", length, away)
	}
}

// anti-tests: a longer lens does not see wider. Focused far off, the
// angle across is smaller the longer the focal length.
func TestALongerLensDoesNotSeeWider(t *testing.T) {
	rapid.Check(t, longerLensSeesNarrower)
}

func longerLensSeesNarrower(t *rapid.T) {
	frame, err := lens.FrameOf(frameWide, frameHigh)
	if err != nil {
		t.Fatal(err)
	}
	short := rapid.Float64Range(focalLeast, focalMost).Draw(t, "short")
	long := short + rapid.Float64Range(focalLeast, focalMost).Draw(t, "more")
	away := long * millimetre * focusMost
	first, err := lens.NewThin(frame, short, stop, away)
	if err != nil {
		t.Fatal(err)
	}
	second, err := lens.NewThin(frame, long, stop, away)
	if err != nil {
		t.Fatal(err)
	}
	if !(second.Wide() <= first.Wide()) {
		t.Fatalf("%v mm sees %g and %v mm sees %g", short, first.Wide(), long,
			second.Wide())
	}
}
