package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestAnEmptySubspaceIsItsParent(test *testing.T) {
	point := vector(test, "a point")
	empty := space(test, "empty")
	near(test, "out", empty.Out(point), point)
	near(test, "in", empty.In(point), point)
}

func TestInUndoesOut(test *testing.T) {
	tilted, point := space(test, "tilted"), vector(test, "a far point")
	out := tilted.Out(point)
	near(test, "in of out", tilted.In(out), point)
}

// Take a subspace moved, turned and twice as large. Points go out and
// back, worked by hand. Rays go in, and their distances are their outside
// ones over the scale.
func TestASubspaceTakesPointsAndRaysAcross(test *testing.T) {
	doubled, start := space(test, "turned and doubled"), vector(test, "one on x")
	out := vector(test, "subspace out")
	near(test, "out", doubled.Out(start), out)
	near(test, "in", doubled.In(out), start)
	ray := maths.Ray{At: vector(test, "subspace in ray start"),
		Dir: vector(test, "subspace in ray way")}
	in := doubled.RayIn(ray)
	near(test, "ray in, start", in.At, vector(test, "subspace in start"))
	near(test, "ray in, way", in.Dir, vector(test, "subspace in way"))
	inside := in.Point(unity)
	near(test, "a distance of one inside", doubled.Out(inside), out)
}

// Ships on turning planets turn with them. The absolute answer is one
// multiplication a level.
func TestWithinComposesAChildInItsParent(test *testing.T) {
	planet, ship := space(test, "planet"), space(test, "ship in place")
	abs := ship.Within(planet)
	near(test, "origin", abs.Origin, planet.Out(ship.Origin))
	point := vector(test, "a point on the ship")
	onShip := ship.Out(point)
	near(test, "a point on the ship", abs.Out(point), planet.Out(onShip))
}

// Take a child turned and smaller in a turned parent. Within is the same
// as going out of each in turn.
func TestWithinComposesATurnedChild(test *testing.T) {
	planet, ship := space(test, "planet"), space(test, "ship")
	abs := ship.Within(planet)
	for _, name := range []string{"origin", "a point on the ship",
		"a point further out"} {
		point := vector(test, name)
		onShip := ship.Out(point)
		near(test, name+" on the ship", abs.Out(point), planet.Out(onShip))
	}
	if want := picked(test, "doubling") * picked(test, "half"); abs.Scale != want {
		test.Errorf("a half in a double is %v, want %v", abs.Scale, want)
	}
}

func TestASubspaceFacesItsMinusZ(test *testing.T) {
	look := maths.Subspace{Orientation: turned(test, "up", quarter)}
	near(test, "forward", look.Forward(), vector(test, "minus x"))
}
