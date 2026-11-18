package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// TestAFieldsShineNamesItsField checks the glow of a field says which field it belongs to.
func TestAFieldsShineNamesItsField(t *testing.T) {
	field, err := NewField([]Ignition{{Fire: campfire}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	named, ok := field.Shine().(interface{ Flames() *Field })
	if !ok {
		t.Fatal("the glow does not name its field")
	}
	if named.Flames() != field {
		t.Fatal("the glow names another field")
	}
}

// TestAFireballRefusesAnExposureOfNone checks a thermal with an exposure that is not above zero is refused.
func TestAFireballRefusesAnExposureOfNone(t *testing.T) {
	thermal := Thermal{Radius: 10, Height: 30, Hottest: measure.Kelvin(1800),
		Air: measure.Kelvin(290), Soot: 0.5, Seed: 1}
	for _, exposure := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := NewFireball(thermal, exposure); err == nil {
			t.Errorf("exposure %v accepted", exposure)
		}
	}
}

// TestAFireballsFinestEddyIsItsRadiusOverTheEddiesAndOctaves checks Finest against its worked value: a ball of radius 10 has eddies 10 / 3 across at the coarsest, halved four times, 10 / 48.
func TestAFireballsFinestEddyIsItsRadiusOverTheEddiesAndOctaves(t *testing.T) {
	ball := aFireball(t)
	if got, want := ball.Finest(), 10.0/48; math.Abs(got-want) > 1e-12 {
		t.Fatalf("Finest=%v, want %v", got, want)
	}
	later := thermalAt(ball, 4)
	if later.Finest() <= ball.Finest() {
		t.Errorf("a grown ball's finest eddy %v is not coarser than %v", later.Finest(), ball.Finest())
	}
}

// TestABallNotYetLetGoIsTheAirsHeat checks the heat before the start time is the air's at any point.
func TestABallNotYetLetGoIsTheAirsHeat(t *testing.T) {
	ball, err := NewFireball(Thermal{Radius: 10, Height: 30, Hottest: measure.Kelvin(1800),
		Air: measure.Kelvin(290), Soot: 0.5, Seed: 1, Start: 5 * time.Second}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := ball.Heat(maths.V(0, 30, 0)).InKelvin(); got != 290 {
		t.Fatalf("heat before the start is %v K, want 290", got)
	}
	if got := thermalAt(ball, 5).Heat(maths.V(0, ball.Centre(), 0)).InKelvin(); got <= 290 {
		t.Fatalf("at its start the heart is %v K, want above the air's 290", got)
	}
}

// TestAFireballsTintAndShine checks the soot tint is the flame's and the glow ticks to the fireball at that time.
func TestAFireballsTintAndShine(t *testing.T) {
	ball := aFireball(t)
	if ball.Tint() != Tilt() {
		t.Errorf("tint %v, want the flame's %v", ball.Tint(), Tilt())
	}
	at := maths.V(0, ball.Centre(), 0)
	shine := ball.Shine()
	later := shine.Tick(3 * time.Second)
	want := thermalAt(ball, 3).Shine().At(at)
	if got := later.At(at); got != want {
		t.Errorf("ticked glow %v, want the ball's glow at 3 s %v", got, want)
	}
	if later.At(at) == shine.At(at) {
		t.Error("ticking the glow did not move the fireball on")
	}
}
