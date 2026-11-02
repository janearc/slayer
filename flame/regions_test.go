package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// axisHeat is how hot a fire is a share of its height up its axis, at its time
// zero.
func axisHeat(t *testing.T, fire Fire, up float64) float64 {
	t.Helper()
	flame := body(t, fire)
	return flame.Heat(maths.V(0, up*fire.Height, 0)).InKelvin()
}

// up its axis the heat holds at the fire's hottest through the
// continuous flame, 0.4 of its height, as mccaffrey measured, and above
// it falls as the height grows: at twice that height, half the hottest's
// lead over the edge, within a tenth of it, as the flame's breath and
// eddies move the point. a straight fall to the edge at the tip would
// leave a fifth.
//
// in the continuous flame the eddies move an axis point a little off the
// axis too: with six octaves and mccaffrey's speeds, measured 10-18, up
// to 1.3 K under the hottest. held to 3 K.
func TestTheContinuousFlameHoldsItsHeat(t *testing.T) {
	fire := campfire
	fire.Seed = 11
	hot, edge := fire.Hottest.InKelvin(), fire.Edge.InKelvin()
	for _, up := range []float64{0.05, 0.2, 0.35} {
		if got := axisHeat(t, fire, up); math.Abs(got-hot) > 3 {
			t.Errorf("%g of the way up, %g K, want the hottest %g", up, got,
				hot)
		}
	}
	middle := 2 * continuous
	want := edge + (hot-edge)/2
	if got := axisHeat(t, fire, middle); math.Abs(got-want) >
		(hot-edge)/10 {
		t.Errorf("at %g of the way up, %g K, want about %g", middle, got,
			want)
	}
}

// the continuous flame is always there on its axis; above, its tongues
// come and go, fewer the higher, all but none near the tip.
func TestTheTonguesThinWithHeight(t *testing.T) {
	flame := body(t, campfire)
	there := func(up float64) float64 {
		held := 0
		const looks = 400
		for k := range looks {
			at := flame.Tick(time.Duration(k) * 37 *
				time.Millisecond).(*Body)
			p := maths.V(0.02*math.Sin(float64(k)), up*campfire.Height,
				0.02*math.Cos(float64(k)))
			if at.Density(p) > 0 {
				held++
			}
		}
		return float64(held) / looks
	}
	low, half, high, tip := there(0.2), there(0.5), there(0.8),
		there(0.97)
	if low < 0.95 || !(half > high) || !(high > tip) || tip > 0.1 ||
		half < 0.3 {
		t.Errorf("there %g low, %g half way, %g high, %g at the tip", low,
			half, high, tip)
	}
}
