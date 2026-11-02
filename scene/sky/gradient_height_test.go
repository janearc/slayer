package sky

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a gradient black at the horizon and white at the zenith has the height
// itself for its light: 0*(1-h) + 1*h is exactly h. so a test can ask for
// a height to the last bit. the ground is a grey of a half, apart from
// both ends and from every height between.
var ramp = Gradient{Horizon: maths.Grey(0), Zenith: maths.Grey(1),
	Ground: maths.Grey(0.5)}

// the heights below are worked by hand, or by a sum to thirty digits that
// no sky code was asked. where two roundings of one number differ by a
// last bit, only the one this sky is bound to is asked for.
//
// root two over two, 0.70710678118654752440, is a vector of equal x and y.
// sixteen over root 257, 0.99805257848288855189, is a vector whose y is a
// sixteenth of its x.
func TestAHeightToTheLastBit(t *testing.T) {
	const (
		diagonal  = math.Sqrt2 / 2
		sixteenth = 0.99805257848288855189
	)
	small, large := 1e-150, 5e149
	cases := []struct {
		name       string
		toward, up maths.Vec
		want       float64
		slack      float64
		why        string
	}{
		{"a way measured, an up too long", maths.V(3, 3, 0), maths.V(0, 1e300, 0),
			diagonal, 0, "a way Unit can measure is measured by Unit"},
		{"a way measured, an up too short", maths.V(3, 3, 0), maths.V(0, 1e-300, 0),
			diagonal, 0, "a way Unit can measure is measured by Unit"},
		{"the smallest part kept off Unit", maths.V(small, small/16, 0), maths.V(1, 0, 0),
			sixteenth, 0, "a part of exactly 1e-150 is measured at its own size"},
		{"the largest part kept off Unit", maths.V(large, large/16, 0), maths.V(1, 0, 0),
			sixteenth, 0, "a part of exactly 5e149 is measured at its own size"},
		{"under a part of 1e150, along x", maths.V(6e149, 0, 0), maths.V(1, 0, 0),
			1, 0, "a part past 5e149 is measured at its own size"},
		{"under a part of 1e150, along z", maths.V(0, 0, 9.99e149), maths.V(0, 0, 1),
			1, 0, "a part past 5e149 is measured at its own size"},
		{"a big way with its largest part in y", maths.V(1, 1e200, 1), maths.Up,
			1, 0, "the largest part is of all three, not two"},
		{"a big way with its largest part in z", maths.V(1, 1, 1e200), maths.ZAxis,
			1, 0, "the largest part is of all three, not two"},
		{"a big 3-4-5 way, every part scaled", maths.V(3e200, 4e200, 0), maths.Up,
			0.8, 1e-15, "every part is divided by the largest"},
		{"a small 3-4-5 way, every part scaled", maths.V(4e-200, 0, 3e-200), maths.ZAxis,
			0.6, 1e-15, "every part is divided by the largest"},
		{"a big way level with the horizon", maths.V(1e200, 0, 0), maths.Up,
			0, 0, "level is the horizon and not the ground"},
		{"a big way at 45 degrees", maths.V(1e200, 1e200, 0), maths.Up,
			diagonal, 1e-15, "a height between is the blend and not the ground"},
		{"a big way straight up", maths.V(0, 1e200, 0), maths.Up,
			1, 0, "straight up is the zenith and not the ground"},
		{"a big way below the horizon", maths.V(1e200, -1e200, 0), maths.Up,
			0.5, 0, "below the horizon is the ground"},
		{"a big way straight down", maths.V(0, -1e200, 0), maths.Up,
			0.5, 0, "below the horizon is the ground"},
		{"a way measured, below the horizon", maths.V(3, -3, 0), maths.V(0, 1e300, 0),
			0.5, 0, "below the horizon is the ground"},
	}
	for _, c := range cases {
		g := ramp
		g.Up = c.up
		if c.up == maths.Up {
			g.Up = maths.Vec{}
		}
		got := g.Light(c.toward)
		want := maths.Grey(c.want)
		if got != want && !near(got, want, c.slack) {
			t.Errorf("%s: toward %v is %v, want %v (%s)", c.name, c.toward,
				got, want, c.why)
		}
	}
}

// an up too long to measure is the same up as one of length one, for a
// way level with the horizon, below it, or between.
func TestAnUpTooLongIsTheSameUp(t *testing.T) {
	big := ramp
	big.Up = maths.V(0, 1e250, 0)
	for _, v := range []maths.Vec{maths.V(1, 0, 0), maths.V(1, 1, 0),
		maths.V(1, -1, 0), maths.V(0, 1, 0), maths.V(2, 3, 5), maths.V(-2, -3, 5)} {
		if got, want := big.Light(v), ramp.Light(v); !near(got, want, 1e-15) {
			t.Errorf("an up of 1e250 toward %v is %v, an up of one gives %v",
				v, got, want)
		}
	}
}
