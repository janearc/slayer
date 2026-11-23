package beamstep

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/scene/thing"
)

// Sizes of the beam these cases check, in metres. The beam starts at the
// origin and runs east along x.
const (
	// cylinderRadius is the beam's launch radius.
	cylinderRadius = 0.5
	// cylinderReach is how far the beam runs.
	cylinderReach = 100.0
	// cylinderFar is how far the checked rays run.
	cylinderFar = 1000.0
	// cylinderTolerance is how far apart two distances along a ray may
	// be and still be the same: a nanometre, far finer than any cell.
	cylinderTolerance = 1e-9
)

// cylinderBeam is the checked beam as the package prepares it, so that
// the cylinder these cases call is the one a draw uses.
func cylinderBeam(test *testing.T) source {
	test.Helper()
	test.Setenv(scrooge.Variable, "1")
	beam, err := emit.NewBeam(emit.Lit{Toward: maths.V(1, 0, 0),
		Strength: maths.Grey(1)}, measure.Metres(cylinderRadius),
		measure.Metres(cylinderReach), measure.Degrees(0))
	if err != nil {
		test.Fatal(err)
	}
	planned := Of([]thing.Lamp{beam})
	if len(planned.sources) != 1 {
		test.Fatal("one beam gave", len(planned.sources), "sources")
	}
	return planned.sources[0]
}

// stretch is the part of a ray inside the checked beam's cylinder. A
// leave that is not past the enter is no part.
func stretch(test *testing.T, at, way maths.Vec) (enter, leave float64) {
	test.Helper()
	return cylinderBeam(test).cylinder(maths.NewRay(at, way), cylinderFar)
}

// near is whether two distances are the same to the tolerance.
func near(got, want float64) bool {
	return math.Abs(got-want) <= cylinderTolerance
}

// A ray across the beam is inside the cylinder for its diameter, and a
// ray that starts inside it has an interval that starts at nought.
//
// Worked: the ray goes up y at x = 50 from y = -10, and the axis is the
// x axis, so it is inside where |y| <= 1.5: from 10 - 1.5 = 8.5 to 11.5.
// From y = 1 it leaves where y = 1.5, half a metre on.
func TestARayThroughABeamHasItsDiameterOfStretch(t *testing.T) {
	enter, leave := stretch(t, maths.V(50, -10, 0), maths.V(0, 1, 0))
	if !near(enter, 8.5) || !near(leave, 11.5) {
		t.Errorf("across: %g to %g, want 8.5 to 11.5", enter, leave)
	}
	enter, leave = stretch(t, maths.V(50, 1, 0), maths.V(0, 1, 0))
	if enter != 0 || !near(leave, 0.5) {
		t.Errorf("from inside: %g to %g, want 0 to 0.5", enter, leave)
	}
}

// A ray along the axis is inside for the beam's length and the caps'
// margin at each end.
//
// Worked: from x = -10 going east, it is inside the caps from
// -1.5, which is 8.5 on, to 100 + 1.5 = 101.5, which is 111.5 on.
//
// Anti: a ray beside the axis and parallel to it, at 2.5 m, past the
// 1.5 m margin, is not inside at all, and at 1.4 m, inside it, it is for
// the same stretch as the axis.
func TestARayAlongABeamHasItsLengthOfStretch(t *testing.T) {
	enter, leave := stretch(t, maths.V(-10, 0, 0), maths.V(1, 0, 0))
	if !near(enter, 8.5) || !near(leave, 111.5) {
		t.Errorf("along: %g to %g, want 8.5 to 111.5", enter, leave)
	}
	enter, leave = stretch(t, maths.V(-10, 1.4, 0), maths.V(1, 0, 0))
	if !near(enter, 8.5) || !near(leave, 111.5) {
		t.Errorf("beside, inside: %g to %g, want 8.5 to 111.5", enter,
			leave)
	}
	if enter, leave = stretch(t, maths.V(-10, 2.5, 0),
		maths.V(1, 0, 0)); leave > enter {
		t.Errorf("a parallel ray past the margin has %g to %g", enter,
			leave)
	}
}

// Anti: a ray beside the beam, past its end, before its start, or
// pointing away from it has no stretch, so a cell there is one step and
// nothing is refined for a beam it misses. Each is worked from the
// cylinder of 1.5 m round the x axis, from -1.5 to 101.5.
func TestARayBesideOrPastABeamHasNoStretch(t *testing.T) {
	for name, ray := range map[string][2]maths.Vec{
		"beside": {maths.V(50, -10, 2.5), maths.V(0, 1, 0)},
		"past its end": {maths.V(102.5, -10, 0),
			maths.V(0, 1, 0)},
		"before it":     {maths.V(-2.5, -10, 0), maths.V(0, 1, 0)},
		"pointing away": {maths.V(50, 10, 0), maths.V(0, 1, 0)},
	} {
		if enter, leave := stretch(t, ray[0], ray[1]); leave > enter {
			t.Errorf("%s: a stretch of %g to %g", name, enter, leave)
		}
	}
}

// A ray through the corner of the beam's box, outside its cylinder, has
// no stretch in the cylinder where the box holds it: the case the
// cylinder is for.
//
// Worked: at y = 1.4 and z = 1.4 the ray is inside the box, which is
// 1.5 m each way of the axis, and 1.4 * sqrt(2) = 1.98 m from the axis,
// past the 1.5 m of the cylinder.
//
// Anti: at y = 1.4 and z = 0 it is 1.4 m from the axis, inside the
// cylinder, which it holds from the near cap at -1.5, 3.5 m on from
// x = -5, to the far cap at 101.5, which is 106.5 m on.
func TestARayThroughTheCornerOfTheBoxIsOutsideTheCylinder(t *testing.T) {
	corner := maths.NewRay(maths.V(-5, 1.4, 1.4), maths.V(1, 0, 0))
	beam := cylinderBeam(t)
	if enter, leave := beam.cylinder(corner, cylinderFar); leave > enter {
		t.Errorf("the corner ray has %g to %g of the cylinder", enter,
			leave)
	}
	if _, _, hit := beam.box.Hit(corner, 0, cylinderFar); !hit {
		t.Error("the box does not hold the corner ray: nothing to tell")
	}
	edge := maths.NewRay(maths.V(-5, 1.4, 0), maths.V(1, 0, 0))
	enter, leave := beam.cylinder(edge, cylinderFar)
	if !near(enter, 3.5) || !near(leave, 106.5) {
		t.Errorf("the edge ray has %g to %g, want 3.5 to 106.5", enter,
			leave)
	}
}
