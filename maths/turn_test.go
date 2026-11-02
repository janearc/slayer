package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// Turns are right-handed. Each case of turns.csv is worked by hand from
// the right-hand rule. In takes the answer back to where it started.
func TestATurnIsRightHanded(test *testing.T) {
	for _, row := range read(test, turnsFile) {
		turn := turned(test, row["axis"], float(test, row, "degrees"))
		direction, goesTo := vector(test, row["direction"]), vector(test, row["goes to"])
		near(test, row["name"], turn.Out(direction), goesTo)
		near(test, row["name"]+", and back", turn.In(goesTo), direction)
	}
}

func TestAboutIsRotationUnderItsOldName(test *testing.T) {
	axis, angle := maths.ZAxis, picked(test, "turn-radians")
	if maths.About(axis, angle) != maths.Rotation(axis, angle) {
		test.Error("About is not Rotation")
	}
}

// one turn then another is not the other then the one: each worked by
// hand from where the axes go.
func TestTurnsComposeInTheirOrder(test *testing.T) {
	aboutZ := turned(test, "up from z", quarter)
	aboutX := turned(test, "one on x", quarter)
	nearTurn(test, "z then x", aboutZ.Then(aboutX), orientation(test, "z then x"))
	nearTurn(test, "x then z", aboutX.Then(aboutZ), orientation(test, "x then z"))
}

// with no turn at all, a turn is the identity.
func TestNoTurnIsTheIdentity(test *testing.T) {
	if maths.Identity != orientation(test, "no turn") {
		test.Errorf("the identity is %v", maths.Identity)
	}
}

// Standing on the world's up is no turn. Standing on x takes the near-x
// branch, which is worked by hand and right-handed.
func TestUprightOnTheAxes(test *testing.T) {
	if got := maths.Upright(maths.Up); got != maths.Identity {
		test.Errorf("upright on up is %v", got)
	}
	if got, want := maths.Upright(maths.XAxis), orientation(test, "standing on x"); got != want {
		test.Errorf("upright on x is %v, want %v", got, want)
	}
}

// Take an up whose x is exactly 0.9. It pairs with x, since 0.9 is not
// past the limit. Take one whose x is the next number up. It pairs with z.
// Both orientations are recorded from the code and checked for the pairing.
func TestUprightPairsWithXUpToExactlyNineTenths(test *testing.T) {
	for _, row := range []struct{ name, up, want string }{
		{"at the edge", "edge stand", "standing at the edge of along x"},
		{"past the edge", "past edge stand", "standing past the edge of along x"},
	} {
		up := vector(test, row.up)
		got := maths.Upright(up)
		nearTurn(test, row.name, got, orientation(test, row.want))
		if got.Y != up {
			test.Errorf("%s: y is %v, want the up %v", row.name, got.Y, up)
		}
	}
}

// Upright stands on the up it is given, square and right-handed.
func TestUprightStandsOnItsUpAndIsSquare(test *testing.T) {
	up := vector(test, "upright up")
	got := maths.Upright(up)
	tight := picked(test, "tight")
	if got.Y.Sub(up.Unit()).Len() > tight {
		test.Fatalf("y is %v, not up", got.Y)
	}
	if abs(got.X.Dot(got.Y)) > tight || abs(got.X.Dot(got.Z)) > tight ||
		abs(got.Y.Dot(got.Z)) > tight {
		test.Fatal("the axes are not square")
	}
	if got.X.Cross(got.Y).Sub(got.Z).Len() > tight {
		test.Fatal("the axes are left-handed")
	}
}

// looking at something, up is up.
func TestLookingAtItUpIsUp(test *testing.T) {
	look := maths.LookAt(vector(test, "eye"), vector(test, "target"))
	if look.Y.Y <= nought {
		test.Errorf("looking at it, up is %v", look.Y)
	}
}

// Cameras face their targets and stay level.
func TestLookAtFacesTheTargetAndStaysLevel(test *testing.T) {
	from, to := vector(test, "eye"), vector(test, "target")
	space := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, to)}
	near(test, "forward", space.Forward(), to.Sub(from).Unit())
	if right := space.Orientation.Out(maths.XAxis); abs(right.Y) > within(test) {
		test.Errorf("its right tilts: %v", right)
	}
}

// Each look of looks.csv is the orientation of someone at from, facing
// to. It has the up given or the world's.
// The first three are worked by hand.
// The rest are recorded from the code at the limits.
// Each one's branch was checked by hand.
func TestEachLookIsWorkedOrRecorded(test *testing.T) {
	for _, row := range read(test, looksFile) {
		from, to := vector(test, row["from"]), vector(test, row["to"])
		got := maths.LookAtUp(from, to, upOf(test, row))
		nearTurn(test, row["name"], got, orientation(test, row["orientation"]))
	}
}

// upOf is the up a row of looks.csv gives, or the world's when it gives
// none.
func upOf(test reporter, row map[string]string) maths.Vec {
	test.Helper()
	if row["up"] == "" {
		return maths.Up
	}
	return vector(test, row["up"])
}

// LookAtUp faces the target with the up it is given, not the world's.
func TestLookAtUpKeepsTheUpGiven(test *testing.T) {
	look := maths.LookAtUp(vector(test, "side"), vector(test, "origin"),
		vector(test, "up from z"))
	if look.Z.Sub(maths.XAxis).Len() > picked(test, "tight") {
		test.Fatalf("back is %v, not +x", look.Z)
	}
	if abs(look.Y.Dot(maths.ZAxis)-unity) > picked(test, "tight") {
		test.Fatalf("up is %v, not +z", look.Y)
	}
}
