package bodies_test

import (
	"testing"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

const (
	// retroSpin is the spin share of the hole turned the wrong way, defined
	// as a share celestial refuses: one below zero.
	retroSpin = -0.6
	// forwardSpin is the same share turned the right way, defined as
	// retroSpin with its sign taken off.
	forwardSpin = -retroSpin
	// holeScale is the units a kilometre the holes are placed at, defined
	// as the scale the hole tests of this package place a body at.
	holeScale = 0.5
	// holeDetail is how finely the ground is tiled, defined as the detail
	// the hole tests of this package place a body at.
	holeDetail = 2
	// halfRadius is what the schwarzschild radius is divided by to give a
	// hole's mass as a length, defined: the mass length is half the radius.
	halfRadius = 2
	// massTolerance is the share of its value the mass and spin may be off
	// by, defined as the digits the sun's radius is given to.
	massTolerance = 1e-4
	// placeTolerance is the distance a direction or a point may be off by,
	// defined as a few units in the last place of a float64 sum.
	placeTolerance = 1e-12
	// holeEast is how far along x the retrograde hole is put, defined as
	// the distance the hole tests of this package put a hole at.
	holeEast = 10
	// atStart is the moment a field is looked at, defined as the start of
	// its motion, when it has not moved.
	atStart = 0
	// oneField is how many fields a hole is, defined: a hole is one field.
	oneField = 1
)

// onlyField is the one field of the world, which must have just that.
func onlyField(test *testing.T, world *thing.World) *thing.Field {
	test.Helper()
	if len(world.Fields) != oneField {
		test.Fatalf("a hole has %d fields, want %d", len(world.Fields), oneField)
	}
	var only *thing.Field
	for _, field := range world.Fields {
		only = field
	}
	return only
}

// retroHole is a new world with a hole of the sun's mass placed in it, at
// a point and spinning at a share of its mass as a length.
func retroHole(test *testing.T, spin float64, at maths.Vec) *thing.World {
	test.Helper()
	world := thing.NewWorld()
	body := hole(test).Spinning(spin)
	if err := body.Place(world, at, holeScale, holeDetail); err != nil {
		test.Fatal(err)
	}
	return world
}

// celestial refuses a spin below zero and says to turn z about instead
// (hole.NewHole, hole/hole.go). A hole spinning at minus 0.6 of its mass
// is so a field of spin 0.6 of its mass, in a space whose up is turned
// to down. At half a unit a kilometre the sun's mass as a length is
// 2.95325 x 0.5 / 2 = 0.7383125, so the spin is 0.6 x 0.7383125.
func TestARetrogradeHoleIsAFieldCelestialAcceptsWithItsAxisTurned(test *testing.T) {
	east := maths.XAxis.Scale(holeEast)
	world := retroHole(test, retroSpin, east)
	field := onlyField(test, world)
	length := sunRs * holeScale / halfRadius
	if _, err := kerr.NewHole(field.Mass, field.Spin); err != nil {
		test.Errorf("celestial refuses the field the hole makes: %v", err)
	}
	near(test, "the field's spin", field.Spin, forwardSpin*length, massTolerance)
	north := field.At(atStart).Orientation.Out(maths.Up)
	nearVec(test, "the field's north", north, maths.Up.Neg(), placeTolerance)
	nearVec(test, "where the field is", field.At(atStart).Origin, east, placeTolerance)
}

// A hole spinning forward keeps its north up.
func TestAProgradeHoleKeepsItsAxisUp(test *testing.T) {
	world := retroHole(test, forwardSpin, maths.Local().Origin)
	field := onlyField(test, world)
	north := field.At(atStart).Orientation.Out(maths.Up)
	nearVec(test, "the field's north", north, maths.Up, placeTolerance)
}
