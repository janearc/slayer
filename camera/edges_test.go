package camera

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// Reach is half the square root of the largest float64, and a vector of
// exactly that length is in reach where one a step longer is not.
func TestAVectorIsInReachUpToExactlyReachAndNoFurther(test *testing.T) {
	want := math.Sqrt(math.MaxFloat64) / 2
	if reach() != want {
		test.Fatalf("reach is %g, want half the root of the largest "+
			"float64, %g", reach(), want)
	}
	at := maths.V(reach(), 0, 0)
	if !inReach(at) {
		test.Errorf("a vector of length reach is out of reach")
	}
	beyond := math.Nextafter(reach(), math.Inf(1))
	past := maths.V(beyond, 0, 0)
	if inReach(past) {
		test.Errorf("a vector a step past reach is in reach")
	}
	inside := math.Nextafter(reach(), nothing)
	before := maths.V(inside, 0, 0)
	if !inReach(before) {
		test.Errorf("a vector a step inside reach is out of reach")
	}
}

// A vector may be finite and still far past reach; it is out of reach.
func TestAFiniteVectorFarPastReachIsOutOfReach(test *testing.T) {
	for _, length := range []float64{2 * reach(), 1e154} {
		if inReach(maths.V(length, nothing, nothing)) {
			test.Errorf("a vector of length %g is in reach", length)
		}
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1)} {
		if inReach(maths.V(bad, nothing, nothing)) {
			test.Errorf("a vector with %g in it is in reach", bad)
		}
	}
}

// A look is refused when either target, or the way from one to the
// other, cannot be measured, each of the three alone.
func TestALookIsRefusedForAnyOneUnmeasurableTargetOrWay(test *testing.T) {
	stand := maths.V(0, 10, 0)
	hold, err := Timed(stand, stand, time.Second, "")
	if err != nil {
		test.Fatal(err)
	}
	// squaring 1.2e154 passes a float64's range once, squaring it twice over does not
	huge := 1.2e154
	for _, each := range []struct {
		name     string
		from, to maths.Vec
	}{
		// each target measures; the way between them overflows
		{"the way between", maths.V(1e154, 0, 0), maths.V(-1e154, 0, 0)},
		// the first target's length overflows; the second's, and the way
		// between, do not
		{"the first", maths.V(huge, huge, 0), maths.V(huge, 0, 0)},
		// the second target's length overflows; the first's, and the way
		// between, do not
		{"the second", maths.V(huge, 0, 0), maths.V(huge, huge, 0)},
	} {
		if _, err := hold.Looking(each.from, each.to); err == nil {
			test.Errorf("a look with only %s unmeasurable was taken",
				each.name)
		}
	}
	if _, err := hold.Looking(maths.V(1, 0, 0), maths.V(-1, 0, 0)); err != nil {
		test.Errorf("a look of measurable targets was refused: %v", err)
	}
}

// A path is refused when the camera's difference from its target, at the
// start, at the end or in its travel, is past reach, each of the three
// alone, and it is taken when all three are within.
func TestAPathIsRefusedForAnyOneDifferenceBeyondReach(test *testing.T) {
	r := reach()
	for _, each := range []struct {
		name     string
		from, to float64
	}{
		{"the start", 1.2 * r, 0.8 * r},
		{"the end", 0.8 * r, 1.2 * r},
		{"the travel", -0.9 * r, 0.9 * r},
	} {
		move, err := Timed(maths.V(each.from, 0, 0),
			maths.V(each.to, 0, 0), time.Second, "")
		if err != nil {
			test.Fatal(err)
		}
		_, err = New([]Move{move}, maths.V(0, 0, 0), maths.Up)
		if err == nil || !strings.Contains(err.Error(),
			"numeric range") {
			test.Errorf("a difference past reach at %s: %v",
				each.name, err)
		}
	}
	inside, err := Timed(maths.V(r, nothing, nothing),
		maths.V(r, nothing, nothing), time.Second, "")
	if err != nil {
		test.Fatal(err)
	}
	if _, err := New([]Move{inside}, maths.V(0, 0, 0),
		maths.Up); err != nil {
		test.Errorf("a difference of exactly reach was refused: %v", err)
	}
}

// A refusal names the move by its place in the sequence counting from
// one: the second move is move 2, not move 0.
func TestALookThatJumpsNamesTheMoveByItsPlace(test *testing.T) {
	stand := maths.V(0, 10, 0)
	first, err := Timed(stand, stand, time.Second, "")
	if err != nil {
		test.Fatal(err)
	}
	second := first
	second, err = second.Looking(maths.V(5, 10, 5), maths.V(6, 10, 6))
	if err != nil {
		test.Fatal(err)
	}
	_, err = New([]Move{first, second}, maths.V(100, 10, 0), maths.Up)
	if err == nil || !strings.Contains(err.Error(), "camera move 2: "+
		"does not start looking where the previous move ended") {
		test.Errorf("the jump was not named at move 2: %v", err)
	}
}

// A zoom may last as long as the longest duration, and no longer.
func TestAZoomMayLastExactlyTheLongestDuration(test *testing.T) {
	if _, err := NewZoom(20, 60, maxDuration, ""); err != nil {
		test.Errorf("the longest duration was refused: %v", err)
	}
	if _, err := NewZoom(20, 60, maxDuration+1, ""); err == nil {
		test.Error("a duration past the longest was taken")
	}
}

// At the end of the zoom the lens is exactly where it ends, however the
// sum of the start and the travel would round: 10.1 + (0.3-10.1) is not
// 0.3 in a float64.
func TestAZoomAtItsEndIsExactlyWhereItEnds(test *testing.T) {
	for _, ease := range []string{"", "linear", "smooth"} {
		zoom, err := NewZoom(10.1, 0.3, time.Second, ease)
		if err != nil {
			test.Fatal(err)
		}
		got, err := zoom.At(time.Second)
		if err != nil || got != 0.3 {
			test.Errorf("%q at its end: %v, %v, want 0.3", ease, got,
				err)
		}
		got, err = zoom.At(time.Second + 1)
		if err != nil || got != 0.3 {
			test.Errorf("%q past its end: %v, %v, want 0.3", ease,
				got, err)
		}
	}
}

// A field of view whose half in radians is too small to tell from zero is
// refused, though it is above zero in degrees: forty of the smallest
// float64 come to a seventh of the smallest float64 in radians.
func TestAFieldOfViewTooSmallInRadiansIsRefused(test *testing.T) {
	tiny := 40 * math.SmallestNonzeroFloat64
	if _, err := NewZoom(tiny, 60, time.Second, ""); err == nil {
		test.Error("a field of view of nothing in radians was taken")
	}
	if _, err := NewZoom(60, tiny, time.Second, ""); err == nil {
		test.Error("a field of view of nothing in radians was taken")
	}
}
