package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// The streaks are noise along the orbit, and are worked here at the point (4,
// 0, 0) of a disc from 2 to 14. there the radius is 4, two inner radii, so the
// share of the peak is 0.8966016358667303 and its fourth power
// 0.6462463565813805 (bc).
//
// across is 6*(4 - 2)/(14 - 2) = 1. a shear of pi/4 over ln 2 turns the streaks
// by pi/4 at two inner radii, since ln(4/2) is ln 2, and the angle from +X is
// zero.
//
// so the noise is asked at (1, 2cos(pi/4), 2sin(pi/4)), three octaves, seed 1,
// where it is -0.2748.
func streakDisc(streaks float64) Thin {
	return Thin{Inner: 2, Outer: 14, Hot: 25000, Bright: 2, Streaks: streaks,
		Shear: math.Pi / 4 / math.Ln2, Seed: 1}
}

var streakPoint = maths.V(4, 0, 0)

func streakNoise() float64 {
	return noise.Fractal(1, 1, 2*math.Cos(math.Pi/4), 2*math.Sin(math.Pi/4), 3)
}

// The streaks scale the light by one plus twice their strength times
// the noise there.
func TestThinStreaksScaleTheLightByTheNoise(t *testing.T) {
	n := streakNoise()
	if !(n < -0.2 && n > -0.3) {
		t.Fatalf("the noise at the point was meant to be about -0.27 and is %v", n)
	}
	for _, s := range []float64{0.5, 1, 1.5} {
		got := streakDisc(s).At(streakPoint).Luminance()
		want := 2 * 0.6462463565813805 * (1 + s*2*n)
		if !within(got, want, 1e-9) {
			t.Errorf("streaks of %v: a luminance of %v, want %v", s, got, want)
		}
	}
	plain := streakDisc(0).At(streakPoint).Luminance()
	if want := 2 * 0.6462463565813805; !within(plain, want, 1e-9) {
		t.Errorf("no streaks: a luminance of %v, want %v", plain, want)
	}
}

// Streaks so strong that one plus twice the noise is less than zero darken the
// disc to black and no further.
func TestThinStreaksNeverTakeLightAway(t *testing.T) {
	if n := streakNoise(); !(1+10*2*n < 0) {
		t.Fatalf("a strength of 10 was meant to take the factor below nought, at %v", 1+10*2*n)
	}
	if got := streakDisc(10).At(streakPoint); got != (maths.Light{}) {
		t.Errorf("streaks of ten gave %v, not black", got)
	}
}

// The streaks follow the shear: the angle they are asked at is the
// angle round the disc plus the shear times the log of the radius in
// inner radii.
//
// a quarter turn of the point round the disc is a quarter turn of the
// noise, so the shear of pi/4 over ln 2 at (0, 0, 4), angle pi/2, asks
// at 3pi/4; at (-4, 0, 0) it asks at 5pi/4.
func TestThinStreaksAreAskedAtTheAngleRoundTheDisc(t *testing.T) {
	for _, c := range []struct {
		p   maths.Vec
		phi float64
	}{
		{maths.V(0, 0, 4), 3 * math.Pi / 4},
		{maths.V(-4, 0, 0), 5 * math.Pi / 4},
		{maths.V(0, 0, -4), math.Pi/4 - math.Pi/2},
	} {
		n := noise.Fractal(1, 1, 2*math.Cos(c.phi), 2*math.Sin(c.phi), 3)
		got := streakDisc(0.5).At(c.p).Luminance()
		want := 2 * 0.6462463565813805 * (1 + 0.5*2*n)
		if !within(got, want, 1e-9) {
			t.Errorf("at %v: a luminance of %v, want %v", c.p, got, want)
		}
	}
}
