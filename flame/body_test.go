package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// campfire is a fire half a metre across, as quintiere and mccaffrey's
// cribs were fitted: 0.8 a metre, 1100 K, cooling to 800 K at its edge.
var campfire = Fire{Radius: 0.25, Height: 0.6, Absorption: 0.8,
	Hottest: measure.Kelvin(1100), Edge: measure.Kelvin(800), Seed: 7}

// body is a fire's body at an exposure of one, or the test stops.
func body(t *testing.T, fire Fire) *Body {
	t.Helper()
	made, err := NewBody(fire, 1)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// low is a point on the axis a fifth of the way up.
func low(fire Fire) maths.Vec { return maths.V(0, 0.2*fire.Height, 0) }

// a flame is the same at a time however often it is asked, bit for bit,
// and another time moves it.
func TestAFlameIsTheSameAtATime(t *testing.T) {
	first, again := body(t, campfire).Tick(time.Second).(*Body),
		body(t, campfire).Tick(time.Second).(*Body)
	moved := false
	later := first.Tick(time.Second + 130*time.Millisecond).(*Body)
	for i := range 50 {
		p := maths.V(0.1*math.Sin(float64(i)), 0.012*float64(i), 0.05)
		if first.Density(p) != again.Density(p) ||
			first.Heat(p) != again.Heat(p) {
			t.Fatalf("at %v the flame was two flames", p)
		}
		moved = moved || later.Density(p) != first.Density(p)
	}
	if !moved {
		t.Error("an eighth of a second later the flame had not moved")
	}
}

// the glow at every point is a black body at the flame's temperature
// there, by kirchhoff's law, and at its hottest it is srgb's reading of
// planck's law at that temperature.
func TestAFlameGlowsAtItsOwnTemperature(t *testing.T) {
	fire := body(t, campfire)
	glow := fire.Shine()
	for i := range 20 {
		p := maths.V(0.01*float64(i), 0.03*float64(i), 0)
		if glow.At(p) != fire.glows.at(fire.Heat(p)) {
			t.Fatalf("at %v the glow is not the heat's", p)
		}
	}
	red, green, blue := srgb.Light(observer.Swatch(Glowing(campfire.Hottest)))
	hottest := fire.glows.at(campfire.Hottest)
	for c, want := range [3]float64{red, green, blue} {
		got := [3]float64{hottest.R, hottest.G, hottest.B}[c]
		if !near(got, math.Max(0, want), 1e-12) && !(want < 0 && got == 0) {
			t.Errorf("channel %d at its hottest is %v, planck says %v", c,
				got, want)
		}
	}
	if !(hottest.Luminance() > fire.glows.at(campfire.Edge).Luminance()) {
		t.Error("the hottest middle is no brighter than the edge")
	}
}

// soot is there in the middle and nowhere past the flame's box, its
// base or its tip.
func TestAFlameIsInsideItsBox(t *testing.T) {
	fire := body(t, campfire)
	if !(fire.Density(low(campfire)) > 0) {
		t.Errorf("the middle low down is clear: %v",
			fire.Density(low(campfire)))
	}
	box := fire.Bounds()
	for _, p := range []maths.Vec{maths.V(0, -0.01, 0),
		maths.V(0, box.Max.Y+0.01, 0), maths.V(box.Max.X+0.01, 0.1, 0),
		maths.V(0, 0.1, box.Min.Z-0.01)} {
		if got := fire.Density(p); got != 0 {
			t.Errorf("at %v, outside, there is soot: %v", p, got)
		}
	}
}

// a flame puffs at the frequency its width gives: one four times as wide puffs
// half as often, by 1.5 over the square root of the width, so it breathes at
// twice the time as the narrow one does at once. over many puffs its height
// keeps its mean.
//
// each period is kept to the nanosecond, so the two agree to a millionth of the
// height, not bit for bit.
func TestAFlameBreathesAtItsFrequency(t *testing.T) {
	wide := campfire
	wide.Radius *= 4
	narrow, broad := body(t, campfire), body(t, wide)
	sum := 0.0
	for i := range 4000 {
		at := time.Duration(i) * 37 * time.Millisecond
		one := narrow.Tick(at).(*Body).breath()
		other := broad.Tick(2 * at).(*Body).breath()
		if math.Abs(one-other) > 1e-6 {
			t.Fatalf("at %v the narrow breathes %v, the wide at twice "+
				"%v", at, one, other)
		}
		sum += one
	}
	if mean := sum / 4000; math.Abs(mean) > 0.01 {
		t.Errorf("over many puffs the flame is %v of itself taller", mean)
	}
}

// the flame is thicker to blue than red by soot's tilt at the eye's
// three wavelengths, worked out here: (550/610)^1.39, 1, (550/465)^1.39.
func TestAFlameIsThickerToBlue(t *testing.T) {
	tint := body(t, campfire).Tint()
	want := [3]float64{math.Pow(550.0/610, 1.39), 1,
		math.Pow(550.0/465, 1.39)}
	for c, got := range [3]float64{tint.R, tint.G, tint.B} {
		if !near(got, want[c], 1e-12) {
			t.Errorf("channel %d meets %v of its density, not %v", c, got,
				want[c])
		}
	}
}

// a fire that cannot be is refused: no size, soot less than zero, no heat, an
// edge hotter than its middle, or no exposure.
func TestABodyThatCannotBeIsRefused(t *testing.T) {
	broken := []func(*Fire){
		func(fire *Fire) { fire.Radius = 0 },
		func(fire *Fire) { fire.Height = math.NaN() },
		func(fire *Fire) { fire.Absorption = -1 },
		func(fire *Fire) { fire.Hottest = measure.Kelvin(0) },
		func(fire *Fire) { fire.Edge = measure.Kelvin(2000) },
	}
	for i, breaking := range broken {
		fire := campfire
		breaking(&fire)
		if _, err := NewBody(fire, 1); err == nil {
			t.Errorf("broken fire %d was made", i)
		}
	}
	if _, err := NewBody(campfire, 0); err == nil {
		t.Error("a fire with no exposure was made")
	}
}
