package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// spinTurnsQuarter is a quarter of a turn, in radians.
const spinTurnsQuarter = math.Pi / pair // defined: a right angle is half of pi

// spinTurnsPointsFile holds the points the spin tests expect, by name.
const spinTurnsPointsFile = "testdata/spin_turns_points.csv"

// spinTurnsPoint is the point called name in spinTurnsPointsFile.
func spinTurnsPoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	rows := readSheet(t, spinTurnsPointsFile)
	row := rows.row(t, spinTurnsPointsFile, name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// spinTurnsPeriod is the time the spins tried take to turn once.
func spinTurnsPeriod(t *testing.T) time.Duration {
	t.Helper()
	return time.Duration(picked(t, "spin-turns-period-seconds")) * time.Second
}

// spinTurnsOut is where the spin at an instant takes a direction.
func spinTurnsOut(spin Spin, at time.Duration, axis maths.Vec) maths.Vec {
	return spin.At(at).Orientation.Out(axis)
}

// a spin with no period never turns, at any time, but starts at its
// Phase. worked by hand: a quarter turn about x takes y to z and z to
// -y, and a quarter turn about up takes x to -z.
func TestASpinOfNoPeriodStaysAtItsPhase(t *testing.T) {
	about := Spin{Axis: maths.XAxis, Phase: spinTurnsQuarter}
	for _, at := range []time.Duration{nought, time.Second, time.Hour} {
		orientation := about.At(at).Orientation
		near(t, "y", orientation.Out(maths.YAxis), maths.ZAxis)
		near(t, "z", orientation.Out(maths.ZAxis), maths.YAxis.Neg())
		near(t, "x", orientation.Out(maths.XAxis), maths.XAxis)
	}
	up := Spin{Phase: spinTurnsQuarter}
	minusZ := spinTurnsPoint(t, "minus-z")
	near(t, "x about up", spinTurnsOut(up, time.Hour, maths.XAxis), minusZ)
}

// a spin turns from its Phase, and the other way when Period is below zero.
// worked by hand, about up, where x goes to -z a quarter on and to z a quarter
// back: a Phase of a quarter and a quarter turn on is a half turn, which takes
// x to -x; a period of minus 4 s at 1 s is a quarter back, which takes x to z.
func TestASpinTurnsFromItsPhaseEitherWay(t *testing.T) {
	period := spinTurnsPeriod(t)
	half := Spin{Period: period, Phase: spinTurnsQuarter}
	minusX := spinTurnsPoint(t, "minus-x")
	near(t, "x half way", spinTurnsOut(half, time.Second, maths.XAxis), minusX)
	back := Spin{Period: -period}
	backX := spinTurnsOut(back, time.Second, maths.XAxis)
	near(t, "x a quarter back", backX, maths.ZAxis)
	// about x, y goes to z in a quarter
	about := Spin{Axis: maths.XAxis, Period: period}
	aboutY := spinTurnsOut(about, time.Second, maths.YAxis)
	near(t, "y about x", aboutY, maths.ZAxis)
}
