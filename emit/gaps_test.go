package emit

import (
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// TestLitStandsAndAtMost checks where a lit emitter stands and that its
// greatest light at a distance is its brightest channel over the distance
// squared, whatever way the point lies.
func TestLitStandsAndAtMost(t *testing.T) {
	lit := Lit{At: maths.V(1, 2, 3), Strength: maths.Light{R: 0.2, G: 0.9, B: 0.5}}
	if got := lit.Stands(); got != maths.V(1, 2, 3) {
		t.Errorf("stands at %v", got)
	}
	cases := map[float64]float64{1: 0.9, 2: 0.225, 3: 0.1}
	for distance, want := range cases {
		if got := lit.AtMost(distance); !near(got, want) {
			t.Errorf("at %g got %g, want %g", distance, got, want)
		}
	}
}

// TestLitAtMostPicksTheBrightestChannel checks that each of the three
// channels can be the brightest.
func TestLitAtMostPicksTheBrightestChannel(t *testing.T) {
	for name, strength := range map[string]maths.Light{
		"red":   {R: 8, G: 1, B: 2},
		"green": {R: 1, G: 8, B: 2},
		"blue":  {R: 1, G: 2, B: 8},
	} {
		if got := (Lit{Strength: strength}).AtMost(2); !near(got, 2) {
			t.Errorf("%s: got %g, want 2", name, got)
		}
	}
}

// TestVolumeStandsAndAtMost checks the ball's middle and that its greatest
// light follows the ball's own falloff: three quarters of the brightest
// channel at the middle and three eighths at the surface of a ball of
// radius two.
func TestVolumeStandsAndAtMost(t *testing.T) {
	source, err := NewVolume(Lit{At: maths.V(4, 5, 6), Strength: maths.Light{R: 1, G: 2, B: 3}}, measure.Metres(2))
	if err != nil {
		t.Fatal(err)
	}
	if got := source.Stands(); got != maths.V(4, 5, 6) {
		t.Errorf("stands at %v", got)
	}
	if got := source.AtMost(0); !near(got, 2.25) {
		t.Errorf("at the middle %g, want 2.25", got)
	}
	if got := source.AtMost(2); !near(got, 1.125) {
		t.Errorf("at the surface %g, want 1.125", got)
	}
}

// TestBeamsStrengthIsTheirLightAtOneUnit checks that the strength of a
// beam is the light of the same star as a point, at one unit.
func TestBeamsStrengthIsTheirLightAtOneUnit(t *testing.T) {
	clock := time0()
	beams := upright().Beams(&clock)
	want := Point{Light: sunlike()}.When(0).Strength
	if got := beams.Strength(); got != want || !(got.G > 0) {
		t.Errorf("strength %v, want %v", got, want)
	}
}

// time0 is a clock reading of nought.
func time0() time.Duration { return 0 }
