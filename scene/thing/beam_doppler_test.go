package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The tables the beam tests read, each row named in its name column.
const (
	beamDopplerBoosts      = "testdata/beam-doppler-boosts.csv"
	beamDopplerSpeeds      = "testdata/beam-doppler-speeds.csv"
	beamDopplerRoundBoosts = "testdata/beam-doppler-round-boosts.csv"
	beamDopplerJetFlows    = "testdata/beam-doppler-jet-flows.csv"
	beamDopplerRoundFlows  = "testdata/beam-doppler-round-flows.csv"
	beamDopplerSlowings    = "testdata/beam-doppler-slowings.csv"
)

// beamDopplerNowhere is the vector of no length and no way.
var beamDopplerNowhere = maths.V(nought, nought, nought)

// beamDopplerPick is a number the beam tests chose, by its name in
// numbers.yaml after the stem beam-doppler.
func beamDopplerPick(t *testing.T, name string) float64 {
	t.Helper()
	return picked(t, "beam-doppler-"+name)
}

// beamDopplerVec is the vector whose three columns are the prefix with x,
// y and z after it.
func beamDopplerVec(
	t *testing.T, row map[string]string, prefix string,
) maths.Vec {
	t.Helper()
	return maths.V(cell(t, row, prefix+"x"), cell(t, row, prefix+"y"),
		cell(t, row, prefix+"z"))
}

// the boost is delta cubed, with delta = 1 / (gamma (1 - beta cos)).
// worked by hand for a jet at 0.6 c, where gamma = 1 / sqrt(1 - 0.36)
// = 1.25:
//
//	at the eye:      delta = 1 / (1.25 * 0.4) = 2,       boost 8
//	away from it:    delta = 1 / (1.25 * 1.6) = 0.5,     boost 0.125
//	across the way:  delta = 1 / 1.25 = 0.8,             boost 0.512
//	at cos 0.8:      delta = 1 / (1.25 * 0.52) = 1/0.65, boost 1/0.274625
func TestBeamBoostIsDeltaCubed(t *testing.T) {
	beam := Beam{Speed: beamDopplerPick(t, "jet-speed")}
	tolerance := beamDopplerPick(t, "tolerance")
	for _, row := range readSheet(t, beamDopplerBoosts) {
		toward := beamDopplerVec(t, row, "toward-")
		at := beamDopplerVec(t, row, "at-")
		want := cell(t, row, "want") / cell(t, row, "over")
		got := beam.Boost(toward, at)
		if !(math.Abs(got-want) <= tolerance*want) {
			t.Errorf("%s: %v, want %v", row["name"], got, want)
		}
	}
}

// a beam at light's speed or past it, or at no speed or a negative one,
// boosts nothing: one, whichever way it is seen from. past light's speed
// gamma is no number, and at it, endless.
func TestABeamOutOfRangeBoostsNothing(t *testing.T) {
	radius := beamDopplerPick(t, "unit-radius")
	directions := []maths.Vec{maths.Up, maths.Up.Neg(), maths.XAxis}
	for _, row := range readSheet(t, beamDopplerSpeeds) {
		speed := cell(t, row, "speed")
		beam := Beam{Speed: speed}
		for _, toward := range directions {
			if got := beam.Boost(toward, maths.Up); got != unity {
				t.Errorf("a beam at %v seen along %v boosts %v, want 1",
					speed, toward, got)
			}
		}
		round := Beam{Speed: speed, Round: true, At: radius}
		toward := maths.V(nought, nought, -unity)
		if got := round.Boost(toward, maths.XAxis); got != unity {
			t.Errorf("a round beam at %v boosts %v, want 1", speed, got)
		}
	}
}

// a beam of no speed boosts nothing even where the slowing outside At has no
// number: At is so large that At over the radius, at a point a ten-billionth
// out, is endless, and zero of endless is no number.
func TestABeamOfNoSpeedBoostsNothingAnywhere(t *testing.T) {
	huge := beamDopplerPick(t, "huge-radius")
	beam := Beam{Speed: nought, Round: true, At: huge}
	offset := beamDopplerPick(t, "near-axis-offset")
	at := maths.V(offset, nought, nought)
	if got := beam.Boost(maths.Up, at); got != unity {
		t.Errorf("a beam of no speed boosts %v, want 1", got)
	}
}

// round, the gas on the axis is still, so light from it is not boosted,
// whichever way it goes, down to a direction of no number.
func TestStillGasOnTheAxisIsNotBoosted(t *testing.T) {
	beam := Beam{Speed: beamDopplerPick(t, "jet-speed"), Round: true,
		At: beamDopplerPick(t, "unit-radius")}
	up := beamDopplerPick(t, "axis-point-up")
	down := beamDopplerPick(t, "axis-point-down")
	unknown := maths.V(math.NaN(), nought, nought)
	directions := []maths.Vec{maths.Up, maths.XAxis, beamDopplerNowhere,
		unknown}
	points := []maths.Vec{beamDopplerNowhere, maths.V(nought, up, nought),
		maths.V(nought, down, nought)}
	for _, toward := range directions {
		for _, at := range points {
			if got := beam.Boost(toward, at); got != unity {
				t.Errorf("on the axis at %v seen along %v the boost is %v, "+
					"want 1", at, toward, got)
			}
		}
	}
}

// a round beam boosts by its flow where the gas is: worked by hand for
// a speed of 0.8 with At = 9, 16 out, where the flow is
// 0.8 * sqrt(9/16) = 0.6, and so gamma 1.25 and the same deltas as a
// jet at 0.6 c, with the flow along -z at +x.
func TestARoundBeamBoostsByItsFlowThere(t *testing.T) {
	beam := Beam{Speed: beamDopplerPick(t, "round-beam-speed"), Round: true,
		At: beamDopplerPick(t, "round-beam-radius")}
	at := maths.V(beamDopplerPick(t, "round-point-east"),
		beamDopplerPick(t, "round-point-north"), nought)
	tolerance := beamDopplerPick(t, "tolerance")
	for _, row := range readSheet(t, beamDopplerRoundBoosts) {
		toward := beamDopplerVec(t, row, "toward-")
		want := cell(t, row, "want")
		got := beam.Boost(toward, at)
		if !(math.Abs(got-want) <= tolerance*want) {
			t.Errorf("seen along %v the boost is %v, want %v",
				toward, got, want)
		}
	}
}

// the flow of a beam along its axis is away from the origin on each
// side, on the equator as well, and along an axis of its own, as long
// as one unit.
func TestAJetFlowsAwayFromItsOrigin(t *testing.T) {
	speed := beamDopplerPick(t, "jet-speed")
	jet := Beam{Speed: speed}
	for _, row := range readSheet(t, beamDopplerJetFlows) {
		at := beamDopplerVec(t, row, "at-")
		want := beamDopplerVec(t, row, "want-")
		if dir, flow := jet.Flow(at); dir != want || flow != speed {
			t.Errorf("at %v the jet flows %v at %v, want %v at 0.6",
				at, dir, flow, want)
		}
	}
	beamDopplerCheckTipped(t, speed)
}

// beamDopplerCheckTipped holds an axis of three units of x to flow one
// unit long, away from the origin on either side.
func beamDopplerCheckTipped(t *testing.T, speed float64) {
	t.Helper()
	length := beamDopplerPick(t, "tipped-axis-length")
	tipped := Beam{Speed: speed, Axis: maths.V(length, nought, nought)}
	point := beamDopplerPick(t, "tipped-point")
	east := maths.V(point, nought, nought)
	if dir, _ := tipped.Flow(east); dir != maths.XAxis {
		t.Errorf("an axis of (3, 0, 0) flows %v at +x, want one unit of x", dir)
	}
	west := maths.V(-point, nought, nought)
	if dir, _ := tipped.Flow(west); dir != maths.XAxis.Neg() {
		t.Errorf("an axis of (3, 0, 0) flows %v at -x", dir)
	}
}

// beamDopplerRoundFlow is the round beam of a row, at the speed, about the
// axis in its columns, with the point it is asked at and the way it should
// flow there.
func beamDopplerRoundFlow(
	t *testing.T, row map[string]string, speed float64,
) (beam Beam, at, want maths.Vec) {
	t.Helper()
	axis := beamDopplerVec(t, row, "axis-")
	beam = Beam{Speed: speed, Round: true, Axis: axis}
	return beam, beamDopplerVec(t, row, "at-"), beamDopplerVec(t, row, "want-")
}

// a round flow goes round the axis, counterclockwise seen from its end,
// by the part of the point square to the axis.
//
// at (1, 2, 0) about up that is (1, 0, 0), and up cross x is -z. about
// x, the point (5, 0, 2) is (0, 0, 2) square to it, and x cross z is -y.
func TestARoundFlowGoesRoundTheAxis(t *testing.T) {
	rows := readSheet(t, beamDopplerRoundFlows)
	speed := beamDopplerPick(t, "round-flow-speed")
	row := rows.row(t, beamDopplerRoundFlows, "about up at (1 2 0)")
	beam, at, want := beamDopplerRoundFlow(t, row, speed)
	if dir, flow := beam.Flow(at); dir != want || flow != speed {
		t.Errorf("about up at (1, 2, 0) the flow is %v at %v, want -z at 0.4",
			dir, flow)
	}
	row = rows.row(t, beamDopplerRoundFlows, "about up at (1 -7 0)")
	beam, at, want = beamDopplerRoundFlow(t, row, speed)
	if dir, _ := beam.Flow(at); dir != want {
		t.Errorf("about up at (1, -7, 0) the flow is %v, want -z", dir)
	}
	row = rows.row(t, beamDopplerRoundFlows, "about x at (5 0 2)")
	beam, at, want = beamDopplerRoundFlow(t, row, speed)
	if dir, _ := beam.Flow(at); dir != want {
		t.Errorf("about x at (5, 0, 2) the flow is %v, want -y", dir)
	}
}

// on the axis the gas is still: a flow of no way and no speed.
func TestTheFlowOnTheAxisIsStill(t *testing.T) {
	beam := Beam{Speed: beamDopplerPick(t, "round-flow-speed"), Round: true,
		At: beamDopplerPick(t, "unit-radius")}
	up := beamDopplerPick(t, "axis-point-up")
	points := []maths.Vec{beamDopplerNowhere, maths.V(nought, up, nought),
		maths.V(nought, -unity, nought)}
	for _, at := range points {
		dir, speed := beam.Flow(at)
		if dir != beamDopplerNowhere || speed != nought {
			t.Errorf("on the axis at %v the flow is %v at %v", at, dir, speed)
		}
	}
}

// a round flow slows outside At by the square root, never passes 0.99
// of light's speed, inside At as well, and with no At does not slow.
//
// worked by hand: 0.8 * sqrt(9/16) = 0.6; 0.9 * sqrt(4/1) = 1.8, which
// is held to 0.99; 0.5 * sqrt(1/0.0625) = 2, held to 0.99; and with no
// At, 0.4 anywhere.
func TestARoundFlowSlowsByTheSquareRootAndIsHeld(t *testing.T) {
	tolerance := beamDopplerPick(t, "tolerance")
	for _, row := range readSheet(t, beamDopplerSlowings) {
		beam := Beam{Speed: cell(t, row, "speed"), Round: true,
			At: cell(t, row, "at")}
		at := maths.V(cell(t, row, "east"), nought, nought)
		want := cell(t, row, "want")
		if _, got := beam.Flow(at); !(math.Abs(got-want) <= tolerance) {
			t.Errorf("%+v at %v flows at %v, want %v", beam, at, got, want)
		}
	}
}
