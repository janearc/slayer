package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the arithmetic of a vector, each answer worked by hand and kept in
// vectors.csv.
func TestAVectorsArithmetic(test *testing.T) {
	first, second := vector(test, "a"), vector(test, "b")
	exactly(test, "add", first.Add(second), vector(test, "a plus b"))
	exactly(test, "sub", first.Sub(second), vector(test, "a minus b"))
	doubling := picked(test, "doubling")
	exactly(test, "scale", first.Scale(doubling), vector(test, "a doubled"))
	exactly(test, "neg", first.Neg(), vector(test, "a negated"))
	exactly(test, "cross", first.Cross(second), vector(test, "a cross b"))
}

// the right-hand rule, on the three axes taken in turn.
func TestTheAxesCrossRightHanded(test *testing.T) {
	exactly(test, "x cross y", maths.XAxis.Cross(maths.YAxis), maths.ZAxis)
	exactly(test, "y cross z", maths.YAxis.Cross(maths.ZAxis), maths.XAxis)
	exactly(test, "z cross x", maths.ZAxis.Cross(maths.XAxis), maths.YAxis)
}

// up is the y axis, and the axes are what they are named.
func TestTheAxesAreWhatTheyAreNamed(test *testing.T) {
	exactly(test, "x", maths.XAxis, vector(test, "one on x"))
	exactly(test, "y", maths.YAxis, vector(test, "one on y"))
	exactly(test, "z", maths.ZAxis, vector(test, "up from z"))
	exactly(test, "up", maths.Up, maths.YAxis)
}

// Rays are at a point after a distance, along their way.
func TestARayIsAtAPointAfterADistance(test *testing.T) {
	ray := maths.Ray{At: vector(test, "ray start"), Dir: vector(test, "down")}
	point := ray.Point(picked(test, "doubling"))
	exactly(test, "point", point, vector(test, "ray end"))
}

func TestADotProductAndALength(test *testing.T) {
	first, second := vector(test, "a"), vector(test, "b")
	if got, want := first.Dot(second), picked(test, "dot-of-a-and-b"); got != want {
		test.Errorf("dot %v, want %v", got, want)
	}
	long := vector(test, "three four nought")
	if got, want := long.Len(), picked(test, "length-of-three-four-nought"); got != want {
		test.Errorf("len %v, want %v", got, want)
	}
}

// near is within, and the edge of within is in.
func TestNearIsWithin(test *testing.T) {
	origin, along := vector(test, "origin"), vector(test, "a tenth along x")
	if !origin.Near(along, picked(test, "near-just-enough")) {
		test.Error("a point a tenth away is not near a tenth")
	}
}

// Lengths and directions one unit long hold from 1e-150 to 1e150, inside
// what a float can square. Both are held to one part in 1e-12.
func TestALengthAndADirectionAtEverySize(test *testing.T) {
	for _, row := range read(test, scalesFile) {
		scale := float(test, row, "scale")
		sized := vector(test, "three four nought").Scale(scale)
		want := picked(test, "length-of-three-four-nought") * scale
		if got := sized.Len(); math.Abs(got/want-unity) > picked(test, "one-part-in") {
			test.Errorf("%s: a vector of 3, 4, 0 times %v is %v long",
				row["name"], scale, got)
		}
		direction := vector(test, "three four nought as a direction")
		if !sized.Unit().Near(direction, picked(test, "tight")) {
			test.Errorf("%s: the direction is %v, want %v", row["name"],
				sized.Unit(), direction)
		}
	}
}

func TestADirectionIsAsItWasWorkedBefore(test *testing.T) {
	for _, name := range []string{"three four nought", "sample one", "sample two"} {
		first := vector(test, name)
		root := math.Sqrt(first.Dot(first))
		was := first.Scale(unity / root)
		exactly(test, name+" points", first.Unit(), was)
		if first.Len() != root {
			test.Errorf("%s is not as long as before", name)
		}
	}
}
