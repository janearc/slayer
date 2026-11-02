package maths_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the box every hit is made against: the two corners of vectors.csv.
func unitBox(test reporter) maths.Box {
	test.Helper()
	return maths.Box{Min: vector(test, "box min"), Max: vector(test, "box max")}
}

// Boxes are met where a ray enters them and left where it leaves them.
// That holds between near and far. Rays that miss meet nothing. So do
// rays that meet the box only past every distance. So does anything that
// is no number. hits.csv has each case by name.
func TestABoxIsMetWhereARayCrossesIt(test *testing.T) {
	for _, row := range read(test, hitsFile) {
		met, err := strconv.ParseBool(row["ok"])
		if err != nil {
			test.Fatal(err)
		}
		if !met {
			continue
		}
		enter, left, ok := hit(test, row)
		if !ok || enter != float(test, row, "enter") ||
			left != float(test, row, "left") {
			test.Errorf("%s: %v, %v, %v; want %v, %v, true", row["name"], enter,
				left, ok, row["enter"], row["left"])
		}
	}
}

// hit is the unit box's answer to the ray a row of hits.csv names.
func hit(test reporter, row map[string]string) (float64, float64, bool) {
	test.Helper()
	ray := maths.Ray{At: vector(test, row["at"]), Dir: vector(test, row["dir"])}
	return unitBox(test).Hit(ray, float(test, row, "near"), float(test, row, "far"))
}

// Nothing is met where enter would be plus infinity or leave minus
// infinity. One row starts from a far that is no number. Two make one
// from infinity minus infinity. Two have none, and overflow instead.
// Each row says which guard refuses it.
// The next test has four of them again from numbers.
func TestAnInfiniteReachMeetsNothing(test *testing.T) {
	inf := math.Inf(infinity)
	for _, row := range []struct {
		name      string
		box       maths.Box
		at, dir   maths.Vec
		near, far float64
	}{
		// the final guard on leave: Min clears the no-number far to minus infinity
		{"a nan far past a ray at infinity", unitBox(test),
			maths.V(inf, nought, nought), maths.XAxis, -inf, math.NaN()},
		// the early return: x leaves leave as no number
		{"a face at infinity on x, then a face on y", maths.Box{
			Min: maths.V(-inf, -unity, -unity), Max: maths.V(inf, unity, unity)},
			maths.V(inf, inf, nought), maths.V(unity, unity, nought), -inf, unity},
		// the final guard on leave: x's minus infinity clears y's no number
		{"a face at infinity on y, then a face on x", maths.Box{
			Min: maths.V(-unity, -inf, -unity), Max: maths.V(unity, inf, unity)},
			maths.V(inf, inf, nought), maths.V(unity, unity, nought), -inf, unity},
		// the final guard on leave: no no-number at all, both distances overflow
		{"a way too short to reach anything, pointing away", unitBox(test),
			vector(test, "high above"), vector(test, "tiny up"), -inf, inf},
		// the final guard on enter: no no-number at all, both distances overflow
		{"a way too short to reach anything, pointing in", unitBox(test),
			vector(test, "high above"), vector(test, "tiny down"), -inf, inf},
	} {
		enter, left, ok := row.box.Hit(maths.Ray{At: row.at, Dir: row.dir}, row.near, row.far)
		if ok || enter != nought || left != nought {
			test.Errorf("%s: %v, %v, %v; want nothing", row.name, enter, left, ok)
		}
	}
}

// Four of those from numbers are met as worked.
// The last row is the way pointing in, long enough to reach.
func TestTheSameReachFromNumbersIsMet(test *testing.T) {
	inf := math.Inf(infinity)
	for _, row := range []struct {
		name        string
		box         maths.Box
		at, dir     maths.Vec
		near, far   float64
		enter, left float64
	}{
		{"a unit box along x", unitBox(test), maths.V(nought, nought, nought),
			maths.XAxis, -inf, 10, -1, 1},
		{"a face at infinity on x", maths.Box{
			Min: maths.V(-inf, -unity, -unity), Max: maths.V(inf, unity, unity)},
			maths.V(nought, nought, nought), maths.V(unity, unity, nought), -inf, unity, -1, 1},
		{"a face at infinity on y", maths.Box{
			Min: maths.V(-unity, -inf, -unity), Max: maths.V(unity, inf, unity)},
			maths.V(nought, nought, nought), maths.V(unity, unity, nought), -inf, unity, -1, 1},
		{"a way that reaches down to the box", unitBox(test),
			vector(test, "high above"), vector(test, "down"), nought, inf, 4, 6},
	} {
		enter, left, ok := row.box.Hit(maths.Ray{At: row.at, Dir: row.dir}, row.near, row.far)
		if !ok || enter != row.enter || left != row.left {
			test.Errorf("%s: %v, %v, %v; want %v, %v, true", row.name, enter, left, ok,
				row.enter, row.left)
		}
	}
}

// Take a box flat in x, with minus zero for one face and plus zero for the
// other.
//
// It is met at those zeros with their signs. The ray runs along x from the
// origin. Near and far are minus and plus zero. Given the faces either way
// round, the zero entered and the zero left follow the arithmetic and are never
// swapped.
func TestAFlatBoxIsMetAtTheZerosItIsGiven(test *testing.T) {
	minus, plus := math.Copysign(nought, -unity), math.Copysign(nought, unity)
	ray := maths.Ray{At: maths.V(nought, nought, nought), Dir: maths.XAxis}
	for _, row := range []struct {
		name         string
		min, max     float64
		wantEnterNeg bool
		wantLeaveNeg bool
	}{
		{"minus face first", minus, plus, true, false},
		{"plus face first", plus, minus, false, true},
	} {
		box := maths.Box{Min: maths.V(row.min, nought, nought), Max: maths.V(row.max, nought, nought)}
		enter, left, ok := box.Hit(ray, minus, plus)
		if !ok || enter != nought || left != nought {
			test.Fatalf("%s: %v, %v, %v; want zeros", row.name, enter, left, ok)
		}
		if math.Signbit(enter) != row.wantEnterNeg || math.Signbit(left) != row.wantLeaveNeg {
			test.Errorf("%s: enter negative %v, left negative %v; want %v, %v", row.name,
				math.Signbit(enter), math.Signbit(left), row.wantEnterNeg, row.wantLeaveNeg)
		}
	}
}

// the box of a shape with no edge is met by a ray from beside it. The
// property test next to the ray tests holds the box against drawn rays.
func TestTheBoxOfNoEdgeCatchesTheRayBesideIt(test *testing.T) {
	ray := maths.Ray{At: vector(test, "beside"), Dir: vector(test, "down")}
	if _, _, ok := maths.Everywhere().Hit(ray, nought, math.Inf(unity)); !ok {
		test.Fatal("missed")
	}
}

// the box of no edge reaches to infinity every way.
func TestTheBoxOfNoEdgeReachesEveryWay(test *testing.T) {
	everywhere := maths.Everywhere()
	for _, corner := range []maths.Vec{everywhere.Min, everywhere.Max} {
		for _, part := range []float64{corner.X, corner.Y, corner.Z} {
			if !math.IsInf(part, unity) && !math.IsInf(part, -unity) {
				test.Errorf("a corner %v is not at infinity", corner)
			}
		}
	}
	if everywhere.Min.X >= nought || everywhere.Max.X <= nought {
		test.Error("the corners are the wrong way round")
	}
}
