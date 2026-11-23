package beamstep_test

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/render/beamstep"
	"github.com/janearc/slayer/scene/thing"
)

// fineSplits is what a cell half a metre across takes by a step of a
// sixteenth of a metre, the step of a beam a quarter of a metre wide.
const fineSplits = 8

// gapUnionSplits is what a cell a metre across takes by that step.
const gapUnionSplits = 16

// boxCornerSplits is what a cell a quarter of a metre across takes by that
// step.
const boxCornerSplits = 4

// beamSpacing is how far apart the beams of a row stand, in metres.
const beamSpacing = 10

// rowBeams is one more than MostIntervals, so the last place is shared.
const rowBeams = beamstep.MostIntervals + 1

// farEnough is a far that holds the whole of every ray these tests cast.
const farEnough = 400

// beamsAt makes beams that point up from each east position, a quarter of a
// metre wide and ten metres long, from the reference's source.
func beamsAt(test *testing.T, east ...float64) *beamstep.Beams {
	test.Helper()
	fixture := readReference(test)
	var lamps []thing.Lamp
	for _, x := range east {
		point := fixture.Source
		point.At.X = x
		lamps = append(lamps, fixture.beam(test, point))
	}
	return beamstep.Of(lamps)
}

// asked sets the switch for the test, and checks it reads as set.
func asked(test *testing.T, value string) {
	test.Helper()
	test.Setenv(scrooge.Variable, value)
	if scrooge.Asked() != (value == "1") {
		test.Fatal("scrooge does not read as set", value)
	}
}

// across is a ray along east at this height and depth, from two metres west.
func across(height, depth float64) maths.Ray {
	return maths.NewRay(maths.V(-2, height, depth), maths.V(1, 0, 0))
}

func expect(test *testing.T, refine beamstep.Refinement, enter, leave float64,
	want int) {
	test.Helper()
	if got := refine.Splits(enter, leave); got != want {
		test.Fatal("wrong splits", enter, leave, got, want)
	}
}

// A ray through the beam is refined where it is inside the cylinder only.
func TestApartRayThroughBeam(test *testing.T) {
	asked(test, "1")
	refine := beamsAt(test, 0).Ray(across(5, 0), farEnough)
	expect(test, refine, 0, 1, 1)
	expect(test, refine, 1.5, 2, fineSplits)
	expect(test, refine, 2.75, 3, 1)
}

// A ray beside the beam, past its padding, is not refined.
func TestApartRayBesideBeam(test *testing.T) {
	asked(test, "1")
	refine := beamsAt(test, 0).Ray(across(5, 2), farEnough)
	expect(test, refine, 0, farEnough, 1)
}

// A ray along the axis is refined over the beam's length and its caps.
func TestApartRayAlongBeam(test *testing.T) {
	asked(test, "1")
	beams := beamsAt(test, 0)
	along := maths.NewRay(maths.V(0.1, -5, 0), maths.V(0, 1, 0))
	refine := beams.Ray(along, farEnough)
	expect(test, refine, 5, 5.5, fineSplits)
	expect(test, refine, 0, 1, 1)
	expect(test, refine, 16, 17, 1)
	apart := maths.NewRay(maths.V(2, -5, 0), maths.V(0, 1, 0))
	expect(test, beams.Ray(apart, farEnough), 5, 5.5, 1)
}

// A ray across the line of the beam, past its end, is not refined.
func TestApartRayPastBeamEnd(test *testing.T) {
	asked(test, "1")
	refine := beamsAt(test, 0).Ray(across(11, 0), farEnough)
	expect(test, refine, 0, farEnough, 1)
}

// A ray that starts inside the beam is refined from its start.
func TestApartRayStartsInsideBeam(test *testing.T) {
	asked(test, "1")
	start := maths.NewRay(maths.V(0, 5, 0), maths.V(1, 0, 0))
	refine := beamsAt(test, 0).Ray(start, farEnough)
	expect(test, refine, 0, 0.5, fineSplits)
	expect(test, refine, 0.75, 1, 1)
}

// A ray that stops short of the beam is not refined.
func TestApartRayStopsShort(test *testing.T) {
	asked(test, "1")
	refine := beamsAt(test, 0).Ray(across(5, 0), 1)
	expect(test, refine, 0, 1, 1)
}

// A cell between two beams overlaps neither, so it is one step, where the
// union of their boxes would cut it.
func TestApartCellBetweenBeamsIsOneStep(test *testing.T) {
	asked(test, "1")
	refine := beamsAt(test, 0, beamSpacing).Ray(across(5, 0), farEnough)
	expect(test, refine, 4, 5, 1)
	expect(test, refine, 1.5, 2, fineSplits)
	expect(test, refine, beamSpacing+1.5, beamSpacing+2, fineSplits)
}

// Past MostIntervals beams the last place holds the union of the rest, and
// the beams before it keep their gaps.
func TestApartLastPlaceHoldsTheRest(test *testing.T) {
	asked(test, "1")
	east := make([]float64, rowBeams)
	for place := range east {
		east[place] = float64(place * beamSpacing)
	}
	refine := beamsAt(test, east...).Ray(across(5, 0), farEnough)
	before := float64(beamstep.MostIntervals-2)*beamSpacing + 3
	expect(test, refine, before, before+1, 1)
	lastGap := float64(beamstep.MostIntervals-1)*beamSpacing + 3
	expect(test, refine, lastGap, lastGap+1, gapUnionSplits)
}

// With the switch unset the beams' boxes are joined, as they were.
func TestUnaskedRayIsTheUnionOfBoxes(test *testing.T) {
	asked(test, "")
	beams := beamsAt(test, 0, beamSpacing)
	refine := beams.Ray(across(5, 0), farEnough)
	expect(test, refine, 4, 5, gapUnionSplits)
	expect(test, refine, 1.5, 2, fineSplits)
	expect(test, refine, 0, 1, 1)
	beside := beams.Ray(across(5, 2), farEnough)
	expect(test, beside, 0, farEnough, 1)
}

// With the switch unset a ray through the corner of a beam's box, outside
// its cylinder, is refined, as the box would have it; asked, it is not.
func TestUnaskedRayHasBoxCorners(test *testing.T) {
	asked(test, "")
	corner := across(5, 0.7)
	refine := beamsAt(test, 0).Ray(corner, farEnough)
	expect(test, refine, 1.25, 1.5, boxCornerSplits)
	asked(test, "1")
	apart := beamsAt(test, 0).Ray(corner, farEnough)
	expect(test, apart, 1.25, 1.5, 1)
}

// A beam lamp's absence leaves the zero Refinement, one step everywhere.
func TestNoBeamsIsOneStep(test *testing.T) {
	asked(test, "1")
	var none *beamstep.Beams
	ray := across(5, 0)
	expect(test, none.Ray(ray, farEnough), 0, farEnough, 1)
	lit := beamstep.Of([]thing.Lamp{emit.Lit{}})
	expect(test, lit.Ray(ray, farEnough), 0, farEnough, 1)
}
