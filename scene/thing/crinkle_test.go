package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// crinkleUp is the way up, the normal the crinkle tests tilt.
func crinkleUp() maths.Vec {
	return maths.V(nought, unity, nought)
}

// crinkleLook is the unit way the eye looks along in the crinkle tests.
func crinkleLook(t *testing.T) maths.Vec {
	t.Helper()
	return maths.V(picked(t, "crinkle-look-x"), picked(t, "crinkle-look-y"),
		picked(t, "crinkle-look-z")).Unit()
}

// crinkleSeeded is the wrinkles the crinkle tests read, from the seed
// held by the name given.
func crinkleSeeded(t *testing.T, seedName string) *Crinkle {
	t.Helper()
	return &Crinkle{
		Size:  picked(t, "crinkle-size"),
		Slope: picked(t, "crinkle-slope"),
		Seed:  uint32(pickedCount(t, seedName)),
	}
}

// crinkleWrinkles is the wrinkles of the usual seed.
func crinkleWrinkles(t *testing.T) *Crinkle {
	t.Helper()
	return crinkleSeeded(t, "crinkle-seed")
}

// crinklePoint is the place numbered place on the ground, in a grid
// that is columns places along x.
func crinklePoint(t *testing.T, place, columns int) maths.Vec {
	t.Helper()
	return maths.V(float64(place%columns)*picked(t, "crinkle-step-x"), nought,
		float64(place/columns)*picked(t, "crinkle-step-z"))
}

// crinkleTilt is the angle between a normal and the way up.
func crinkleTilt(normal, up maths.Vec) float64 {
	along := normal.Dot(up)
	return math.Acos(math.Min(unity, along))
}

// crinkleColumn is the numbers of a column of a csv table, in order.
func crinkleColumn(t *testing.T, file, column string) []float64 {
	t.Helper()
	var values []float64
	for _, each := range readSheet(t, file) {
		values = append(values, cell(t, each, column))
	}
	return values
}

// crinkleSweep tries the places of the first grid and holds each normal
// a unit long and turned toward the eye. It gives how many places were
// moved, and the largest tilt.
func crinkleSweep(t *testing.T, wrinkles *Crinkle, up, look maths.Vec) (
	moved int, largest float64) {
	t.Helper()
	columns := pickedCount(t, "crinkle-columns")
	tolerance := picked(t, "crinkle-unit-tolerance")
	floor := picked(t, "crinkle-facing-floor")
	movedTilt := picked(t, "crinkle-moved-tilt")
	for i := range pickedCount(t, "crinkle-places") {
		point := crinklePoint(t, i, columns)
		got := wrinkles.CrinkledWithin(point, up, look, nought)
		unit := math.Abs(got.Len()-unity) <= tolerance
		if !unit || !(got.Dot(look) > floor) {
			t.Fatalf("at %v the normal is %v", point, got)
		}
		tilt := crinkleTilt(got, up)
		largest = math.Max(largest, tilt)
		if tilt > movedTilt {
			moved++
		}
	}
	return moved, largest
}

// crinkleKeepsNormal holds that a surface with no wrinkles, or none of
// any size or slope, keeps its normal.
func crinkleKeepsNormal(t *testing.T, up, look maths.Vec) {
	t.Helper()
	probe := maths.V(picked(t, "crinkle-probe-x"), picked(t, "crinkle-probe-y"),
		picked(t, "crinkle-probe-z"))
	var none *Crinkle
	flats := []*Crinkle{none, {Size: nought, Slope: unity},
		{Size: unity, Slope: nought}}
	for _, flat := range flats {
		if got := flat.CrinkledWithin(probe, up, look, nought); got != up {
			t.Errorf("%+v tilts the normal to %v", flat, got)
		}
	}
}

// A crinkle tilts a face's normal, a little and differently from place
// to place, keeping it a unit long and turned toward the eye. The
// anti-test: a surface with none, or none of any size or slope, keeps
// its normal.
func TestACrinkleTiltsTheFaceFromPlaceToPlace(t *testing.T) {
	up, look := crinkleUp(), crinkleLook(t)
	moved, largest := crinkleSweep(t, crinkleWrinkles(t), up, look)
	least := pickedCount(t, "crinkle-least-moved")
	most := picked(t, "crinkle-most-tilt")
	if moved < least || !(largest < most) {
		t.Errorf("%d of %d places tilted, the most by %v radians", moved,
			pickedCount(t, "crinkle-places"), largest)
	}
	crinkleKeepsNormal(t, up, look)
}

// A tilt that would turn the face from the eye is not taken. Seen from
// behind, every tilt of a face turns it away, so the normal is left as
// it was at every place; the anti-test is the same places seen from in
// front, where most are tilted.
func TestACrinkleNeverTurnsAFaceFromTheEye(t *testing.T) {
	up := crinkleUp()
	wrinkles := crinkleWrinkles(t)
	columns := pickedCount(t, "crinkle-few-columns")
	places := pickedCount(t, "crinkle-few-places")
	var tilted int
	for i := range places {
		point := crinklePoint(t, i, columns)
		if got := wrinkles.CrinkledWithin(point, up, up.Scale(-unity),
			nought); got != up {
			t.Fatalf("at %v, seen from behind, the normal is %v", point, got)
		}
		if wrinkles.CrinkledWithin(point, up, up, nought) != up {
			tilted++
		}
	}
	if tilted < pickedCount(t, "crinkle-least-tilted-in-front") {
		t.Errorf("%d of %d places tilted seen from in front", tilted, places)
	}
}

// The same wrinkles at the same place tilt the same way, every time.
// The anti-test: another seed tilts another way.
func TestACrinkleIsTheSameEveryTime(t *testing.T) {
	up := crinkleUp()
	point := maths.V(picked(t, "crinkle-sample-x"), nought,
		picked(t, "crinkle-sample-z"))
	one := crinkleWrinkles(t).CrinkledWithin(point, up, up, nought)
	two := crinkleWrinkles(t).CrinkledWithin(point, up, up, nought)
	other := crinkleSeeded(t, "crinkle-other-seed").CrinkledWithin(point, up,
		up, nought)
	if one != two || one == other {
		t.Errorf("one seed gives %v then %v; another gives %v", one, two, other)
	}
}
