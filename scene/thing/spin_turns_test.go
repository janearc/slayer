package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// a spin with no period never turns, at any time, but starts at its
// Phase. worked by hand: a quarter turn about x takes y to z and z to
// -y, and a quarter turn about up takes x to -z.
func TestASpinOfNoPeriodStaysAtItsPhase(t *testing.T) {
	about := Spin{Axis: maths.XAxis, Phase: math.Pi / 2}
	for _, at := range []time.Duration{0, time.Second, time.Hour} {
		o := about.At(at).Orientation
		near(t, "y", o.Out(maths.YAxis), maths.ZAxis)
		near(t, "z", o.Out(maths.ZAxis), maths.YAxis.Neg())
		near(t, "x", o.Out(maths.XAxis), maths.XAxis)
	}
	up := Spin{Phase: math.Pi / 2}
	near(t, "x about up", up.At(time.Hour).Orientation.Out(maths.XAxis), maths.V(0, 0, -1))
}

// a spin turns from its Phase, and the other way when Period is below zero.
// worked by hand, about up, where x goes to -z a quarter on and to z a quarter
// back: a Phase of a quarter and a quarter turn on is a half turn, which takes
// x to -x; a period of minus 4 s at 1 s is a quarter back, which takes x to z.
func TestASpinTurnsFromItsPhaseEitherWay(t *testing.T) {
	half := Spin{Period: 4 * time.Second, Phase: math.Pi / 2}
	near(t, "x half way", half.At(time.Second).Orientation.Out(maths.XAxis), maths.V(-1, 0, 0))
	back := Spin{Period: -4 * time.Second}
	near(t, "x a quarter back", back.At(time.Second).Orientation.Out(maths.XAxis), maths.ZAxis)
	// about x, y goes to z in a quarter
	x := Spin{Axis: maths.XAxis, Period: 4 * time.Second}
	near(t, "y about x", x.At(time.Second).Orientation.Out(maths.YAxis), maths.ZAxis)
}
