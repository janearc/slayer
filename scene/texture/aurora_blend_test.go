package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The aurora blends by height, worked on a shell from 8 to 16 round a
// body of radius 8, so the height share h is (length - 8) / 8. Below
// 0.15 the fringe eases into the bottom by Smooth(0, 0.15, h); from 0.15
// to 0.35 it is the bottom; above, the bottom eases into the top.
//
// the ease there is Smooth(0.35, 1, h). Smooth at a quarter of the way
// is 0.0625*(3 - 0.5) = 0.15625, and at half way 0.5.
func TestAuroraBlendsByHeightAsTheEaseSays(t *testing.T) {
	fringe, bottom, top := maths.RGB(0.8, 0.2, 0.4), maths.RGB(0.1, 0.6, 0.3), maths.RGB(0.5, 0.9, 0.1)
	au := Aurora{Radius: 8, Low: 0, High: 8, Fringe: fringe, Bottom: bottom, Top: top}
	mix := func(x, y maths.Light, share float64) maths.Light {
		return x.Scale(1 - share).Add(y.Scale(share))
	}
	for _, c := range []struct {
		why    string
		length float64
		want   maths.Light
	}{
		{"below the shell, h of -0.1", 7.2, fringe},
		{"h of 0.0375, a quarter of the way up the fringe's ease", 8.3, mix(fringe, bottom, 0.15625)},
		{"h of 0.075, half way up the fringe's ease", 8.6, mix(fringe, bottom, 0.5)},
		{"h of 0.25, in the bottom", 10, bottom},
		{"h of 0.5125, a quarter of the way up the top's ease", 12.1, mix(bottom, top, 0.15625)},
		{"h of 0.675, half way up the top's ease", 13.4, mix(bottom, top, 0.5)},
		{"h of 1, the top", 16, top},
		{"h of 2, past the shell", 24, top},
	} {
		if got := au.At(maths.V(c.length, 0, 0)); !near(got, c.want) {
			t.Errorf("%s: %v, want %v", c.why, got, c.want)
		}
	}
}

// A shell of no height, or a negative one, is all bottom, at any length.
func TestAuroraOfNoHeightIsAllBottom(t *testing.T) {
	fringe, bottom, top := maths.RGB(0.8, 0.2, 0.4), maths.RGB(0.1, 0.6, 0.3), maths.RGB(0.5, 0.9, 0.1)
	for _, high := range []float64{2, 1} {
		au := Aurora{Radius: 8, Low: 2, High: high, Fringe: fringe, Bottom: bottom, Top: top}
		for _, length := range []float64{5, 10, 12, 30} {
			if got := au.At(maths.V(length, 0, 0)); got != bottom {
				t.Errorf("a shell from 2 to %v, at length %v: %v", high, length, got)
			}
		}
	}
}
