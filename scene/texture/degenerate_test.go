package texture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
)

// checkerPoints are three points the checker tests ask about: the origin,
// one near it and one far off behind it.
func checkerPoints(test *testing.T) []maths.Vec {
	test.Helper()
	return []maths.Vec{
		maths.V(nothing, nothing, nothing),
		sampleAt(test, "place-near", "place-near", "place-near"),
		sampleAt(test, "place-far", "place-near", "place-behind"),
	}
}

// A checker of no size has no cubes to tell apart: it is the first colour
// everywhere, as is one of a size that is no number, and a negative one.
func TestACheckerOfNoSizeIsItsFirstColourEverywhere(test *testing.T) {
	first, second := grey(test, "first-grey"), grey(test, "second-grey")
	for _, size := range []float64{nothing, -whole, math.NaN()} {
		checks := texture.Checker(size, first, second)
		for _, point := range checkerPoints(test) {
			if got := checks.At(point); got != first {
				test.Errorf("a checker of size %v at %v is %v, not %v", size,
					point, got, first)
			}
		}
	}
}

// The anti-test of the one above: a checker of a size does alternate, so
// the guard against no size has not made every checker one colour.
func TestACheckerOfASizeStillAlternates(test *testing.T) {
	first, second := grey(test, "first-grey"), grey(test, "second-grey")
	checks := texture.Checker(whole, first, second)
	origin := maths.V(nothing, nothing, nothing)
	if got := checks.At(origin); got != first {
		test.Errorf("the cell at the origin is %v, not %v", got, first)
	}
	if got := checks.At(maths.XAxis); got != second {
		test.Errorf("the next cell along x is %v, not %v", got, second)
	}
}

// acrossAndUp is a surface laid flat on x and z.
func acrossAndUp(point maths.Vec) (across, up float64) {
	return point.X, point.Z
}

// ornamentOf is an ornament of the cell and ring given, over a checker.
func ornamentOf(test *testing.T, cell, ring float64) (texture.Ornament,
	texture.Texture) {
	test.Helper()
	under := texture.Checker(whole, grey(test, "first-grey"),
		grey(test, "second-grey"))
	inks := [inkCount]maths.Light{grey(test, "ink-bright"), grey(test, "ink-mid"),
		grey(test, "ink-dim")}
	return texture.Ornament{Under: under, Inks: inks, Cell: cell, Ring: ring,
		Coords: acrossAndUp}, under
}

// An ornament whose cell or ring is not over nought has no lozenges, and
// shows what is beneath; it must not index its inks by a ring that is
// no number.
func TestAnOrnamentOfNoCellOrRingShowsWhatIsBeneath(test *testing.T) {
	cell, ring := chosen(test, "ornament-cell"), chosen(test, "ornament-ring")
	missing := math.NaN()
	point := sampleAt(test, "sample-across", "place-near", "sample-up")
	for _, shape := range [][]float64{{nothing, ring}, {cell, nothing},
		{-whole, ring}, {cell, -whole}, {missing, ring}, {cell, missing}} {
		ornament, under := ornamentOf(test, shape[nothing], shape[whole])
		if got := ornament.At(point); got != under.At(point) {
			test.Errorf("an ornament of cell %v and ring %v is %v, not what is beneath %v",
				shape[nothing], shape[whole], got, under.At(point))
		}
	}
}

// The anti-test of the one above: at the middle of a lozenge of a cell
// and a ring there is, the ornament is the first ink and not what is
// beneath.
func TestAnOrnamentOfACellAndARingPaintsItsInks(test *testing.T) {
	cell, ring := chosen(test, "ornament-cell"), chosen(test, "ornament-ring")
	ornament, under := ornamentOf(test, cell, ring)
	middle := maths.V(cell/pairOfThem, nothing, cell/pairOfThem)
	want := grey(test, "ink-bright")
	if got := ornament.At(middle); got != want {
		test.Errorf("the middle of a lozenge is %v, not the first ink %v", got,
			want)
	}
	if want == under.At(middle) {
		test.Error("the first ink is the colour beneath, so nothing was told apart")
	}
}

// pairOfThem is defined: two, the halves a whole is cut into.
const pairOfThem = 2

// inkCount is defined: the inks of an ornament, red, orange and blue.
const inkCount = 3

// A street of no count, or a negative one, has no storms and the giant is
// painted by its bands alone.
func TestAStreetOfNoCountHasNoStorms(test *testing.T) {
	paints := []maths.Light{grey(test, "first-grey"), grey(test, "second-grey")}
	bare := texture.Banded{Paints: paints}
	point := sampleAt(test, "place-near", "place-far", "place-near")
	for _, count := range []int{nothing, -pairOfThem - whole} {
		street := texture.Street{Lat: chosen(test, "street-latitude"),
			Count: count, Size: chosen(test, "street-size")}
		empty := texture.Banded{Paints: paints, Streets: []texture.Street{street}}
		if bare.At(point) != empty.At(point) {
			test.Errorf("a street of %d storms changed the giant's colour", count)
		}
	}
}

// The anti-test of the one above: a street of storms, laid over the point
// asked, does change the giant's colour, so the test above could fail.
func TestAStreetOfStormsChangesTheColourWhereItLies(test *testing.T) {
	paints := []maths.Light{grey(test, "first-grey"), grey(test, "second-grey")}
	bare := texture.Banded{Paints: paints}
	street := texture.Street{Lat: chosen(test, "street-latitude"),
		Count: whole, Size: whole, Turn: whole, Paint: grey(test, "ink-bright")}
	stormy := texture.Banded{Paints: paints, Streets: []texture.Street{street}}
	samples := int(chosen(test, "samples-round"))
	for step := range samples {
		angle := float64(step) * pairOfThem * math.Pi / float64(samples)
		point := maths.V(math.Cos(angle), chosen(test, "street-latitude"),
			math.Sin(angle))
		if bare.At(point) != stormy.At(point) {
			return
		}
	}
	test.Error("a storm as big as the band left the colour alone all the way round")
}
