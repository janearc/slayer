package sky

import (
	"math"
	"runtime/debug"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// ends runs call and fails the test if it has not come back in a few
// seconds. a sky that asks itself the same question again never ends, and
// would hold up the whole run instead of failing in it.
func ends(t *testing.T, what string, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		call()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s did not finish in three seconds", what)
	}
}

// capStack holds a stack to 16 MB for the rest of the test. a stack that
// runs out is fatal, not a panic or a timeout, so the watchdog in ends
// cannot catch it: the cap makes it come early and cheap, at 16 MB and
// not at a gigabyte, for a sky that asks itself the same question again.
func capStack(t *testing.T) {
	old := debug.SetMaxStack(16 << 20)
	t.Cleanup(func() { debug.SetMaxStack(old) })
}

// every kind of sky, each with a star or a tile to find.
func someSkies() map[string]Sky {
	bright := Stars{Tiling: coarse, Seed: 11, Share: 1, Size: 0.5, Dark: dusk}
	return map[string]Sky{
		"gradient":        g,
		"stars":           bright,
		"default stars":   Stars{Seed: 2},
		"hexed gradient":  Hexed{Of: g, Tiling: hexes.Sphere{N: 6}},
		"hexed stars":     Hexed{Of: bright, Tiling: hexes.Sphere{N: 6}, Soft: 0.2},
		"hexed no tiling": Hexed{Of: bright},
	}
}

// a direction that points nowhere is no number of light in all three parts,
// whichever part is the one that is wrong, and whether the largest part is
// zero, endless or no number. endless wins over no number in the largest part,
// as math.Max has it, and still leaves no number.
func TestAWayThatPointsNowhereIsNoNumberInEveryPart(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	ways := []maths.Vec{{}, maths.V(0, nan, 0), maths.V(0, 0, nan), maths.V(nan, nan, nan),
		maths.V(0, inf, 0), maths.V(0, 0, -inf), maths.V(-inf, -inf, -inf),
		maths.V(inf, nan, 0), maths.V(nan, inf, nan), maths.V(inf, 1, 1),
		maths.V(1, 1, -inf), maths.V(1e300, inf, 1e300)}
	for name, sky := range someSkies() {
		for _, way := range ways {
			got := sky.Light(way)
			if !math.IsNaN(got.R) || !math.IsNaN(got.G) || !math.IsNaN(got.B) {
				t.Errorf("%s toward %v is %v, not no number in every part", name, way, got)
			}
		}
	}
	for _, up := range []maths.Vec{maths.V(0, nan, 0), maths.V(nan, 0, 0),
		maths.V(0, inf, 0), maths.V(0, 0, -inf)} {
		sky := Gradient{Horizon: maths.Grey(1), Up: up}
		got := sky.Light(maths.V(1, 2, 3))
		if !math.IsNaN(got.R) || !math.IsNaN(got.G) || !math.IsNaN(got.B) {
			t.Errorf("an up of %v gives %v, not no number in every part", up, got)
		}
	}
}

// a way as small as a number gets is a direction, not a point at nothing:
// 5e-324 along x is the same as 1 along x.
func TestAWayAsSmallAsANumberGetsIsADirection(t *testing.T) {
	for name, sky := range someSkies() {
		for _, c := range []struct{ tiny, one maths.Vec }{
			{maths.V(5e-324, 0, 0), maths.V(1, 0, 0)},
			{maths.V(0, -5e-324, 0), maths.V(0, -1, 0)},
			{maths.V(0, 0, 5e-324), maths.V(0, 0, 1)},
		} {
			if got, want := sky.Light(c.tiny), sky.Light(c.one); got != want {
				t.Errorf("%s toward %v is %v, toward %v %v", name, c.tiny, got, c.one, want)
			}
		}
	}
}

// doubling or halving a way changes no bit of the light, in the range Unit
// measures and again out of it: a power of two moves only the exponent, so the
// light at a way is the light at that way times 2^k.
//
// from one range to the other it is the same to a millionth only: out of Unit's
// range a way is measured by its largest part, a different roundoff.
//
// the axes are in it: a way along one axis has zero for two parts.
func TestALightIsTheSameAtEveryPowerOfTwoOfAWay(t *testing.T) {
	capStack(t)
	ways := []maths.Vec{maths.V(0.3, 0.5, -0.2), maths.V(-0.7, 0.1, 0.4),
		maths.XAxis, maths.YAxis, maths.ZAxis, maths.YAxis.Neg(), maths.ZAxis.Neg(),
		placed(coarse, coarse.Tiles()[100], 11), placed(coarse, coarse.Tiles()[900], 11)}
	measured := []int{-400, -2, 1, 3, 400}
	beyond := []int{600, 900, 1000, -600, -1000}
	for name, sky := range someSkies() {
		for _, w := range ways {
			var base, far maths.Light
			ends(t, name, func() { base = sky.Light(w) })
			ends(t, name, func() { far = sky.Light(w.Scale(math.Ldexp(1, beyond[0]))) })
			if !near(base, far, 1e-6) {
				t.Errorf("%s toward %v: %v, and out of range %v", name, w, base, far)
			}
			for _, k := range measured {
				var got maths.Light
				ends(t, name, func() { got = sky.Light(w.Scale(math.Ldexp(1, k))) })
				if got != base {
					t.Errorf("%s toward %v times 2^%d: %v, want %v", name, w, k, got, base)
				}
			}
			for _, k := range beyond {
				var got maths.Light
				ends(t, name, func() { got = sky.Light(w.Scale(math.Ldexp(1, k))) })
				if got != far {
					t.Errorf("%s toward %v times 2^%d: %v, want %v", name, w, k, got, far)
				}
			}
		}
	}
}
