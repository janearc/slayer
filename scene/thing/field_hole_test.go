package thing

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// fieldHoleTurns is the csv file of the points the tests turn, each with
// its place in the field and its image in the hole, worked by hand.
const fieldHoleTurns = "testdata/field_hole-turns.csv"

// The parts of a hole's vector, and their places.
const (
	fieldHoleParts = 3 // defined: a vector in space has three parts
	fieldHoleX     = 0 // defined: the first part, x
	fieldHoleY     = 1 // defined: the second part, y
	fieldHoleZ     = 2 // defined: the third part, z
)

// fieldHolePoint is the point in the field that a row of the turns names.
func fieldHolePoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := readSheet(t, fieldHoleTurns).row(t, fieldHoleTurns, name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// fieldHoleImage is the image in the hole that a row of the turns names.
func fieldHoleImage(t *testing.T, name string) [fieldHoleParts]float64 {
	t.Helper()
	row := readSheet(t, fieldHoleTurns).row(t, fieldHoleTurns, name)
	return [fieldHoleParts]float64{
		cell(t, row, "hole_x"), cell(t, row, "hole_y"), cell(t, row, "hole_z"),
	}
}

// fieldHoleCross is the cross product of two vectors in the hole.
func fieldHoleCross(
	left, right [fieldHoleParts]float64) [fieldHoleParts]float64 {
	return [fieldHoleParts]float64{
		left[fieldHoleY]*right[fieldHoleZ] - left[fieldHoleZ]*right[fieldHoleY],
		left[fieldHoleZ]*right[fieldHoleX] - left[fieldHoleX]*right[fieldHoleZ],
		left[fieldHoleX]*right[fieldHoleY] - left[fieldHoleY]*right[fieldHoleX],
	}
}

// the hole's maths has its spin axis z where the field's is up, and it
// is a turn: x stays, y goes to z and z goes to minus y. worked by hand,
// (1, 2, 3) goes to (1, -3, 2).
func TestInHoleIsAQuarterTurnAboutX(t *testing.T) {
	field := &Field{}
	worked := fieldHolePoint(t, "worked")
	if got := field.InHole(worked); got != fieldHoleImage(t, "worked") {
		t.Errorf("(1, 2, 3) went to %v, want [1 -3 2]", got)
	}
	if got := field.InHole(maths.Up); got != fieldHoleImage(t, "up") {
		t.Errorf("up went to %v, want the hole's z", got)
	}
	if got := field.InHole(maths.XAxis); got != fieldHoleImage(t, "x-axis") {
		t.Errorf("x went to %v, want the hole's x", got)
	}
	if got := field.InHole(maths.ZAxis); got != fieldHoleImage(t, "z-axis") {
		t.Errorf("z went to %v, want the hole's minus y", got)
	}
}

// a turn keeps handedness: x cross y is z in the field, and the same
// of their images in the hole is the image of z, which is [0 -1 0].
// a mirror would give [0 1 0].
func TestInHoleKeepsItsHandedness(t *testing.T) {
	field := &Field{}
	x, y := field.InHole(maths.XAxis), field.InHole(maths.YAxis)
	cross := fieldHoleCross(x, y)
	if want := field.InHole(maths.ZAxis); cross != want {
		t.Errorf("the images of x and y cross to %v, want the image of z, %v",
			cross, want)
	}
}

// FromHole is InHole undone, worked by hand: [1 -3 2] comes back to
// (1, 2, 3); the hole's z is up, its x is x, its minus y is z.
func TestFromHoleUndoesInHole(t *testing.T) {
	field := &Field{}
	worked := fieldHolePoint(t, "worked")
	if got := field.FromHole(fieldHoleImage(t, "worked")); got != worked {
		t.Errorf("[1 -3 2] came back as %v, want (1, 2, 3)", got)
	}
	if got := field.FromHole(fieldHoleImage(t, "up")); got != maths.Up {
		t.Errorf("the hole's z came back as %v, want up", got)
	}
	if got := field.FromHole(fieldHoleImage(t, "z-axis")); got != maths.ZAxis {
		t.Errorf("the hole's minus y came back as %v, want z", got)
	}
	if got := field.FromHole(fieldHoleImage(t, "x-axis")); got != maths.XAxis {
		t.Errorf("the hole's x came back as %v, want x", got)
	}
	origin := maths.V(nought, nought, nought)
	round := fieldHolePoint(t, "round-trip")
	for _, each := range []maths.Vec{worked, round, origin} {
		if got := field.FromHole(field.InHole(each)); got != each {
			t.Errorf("%v went there and back as %v", each, got)
		}
	}
}
