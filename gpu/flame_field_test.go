//go:build darwin

package gpu

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
)

func compositeFireGrid(t *testing.T, at time.Duration) boxed {
	t.Helper()
	base := flame.Fire{Radius: 0.25, Height: 1.5, Absorption: 1,
		Hottest: measure.Kelvin(1750), Edge: measure.Kelvin(900), Seed: 7}
	other := base
	other.Hottest = measure.Kelvin(1400)
	other.Seed = 29
	field, err := flame.NewField([]flame.Ignition{
		{At: maths.V(-0.35, 0, 0), Fire: base}, {At: maths.V(0.35, 0, 0), Fire: other},
		{At: maths.V(0, 0, 0.3), Fire: base},
	}, 100)
	if err != nil {
		t.Fatal(err)
	}
	field = field.Tick(at).(*flame.Field)
	bounds := field.Bounds()
	return boxed{med: field, shine: field.Shine(), low: bounds.Min, high: bounds.Max, cells: [3]int{29, 37, 23}}
}

func TestCompositeFlameGridKeepsWeightedEmissionAndGaps(t *testing.T) {
	for _, at := range []time.Duration{0, 137 * time.Millisecond} {
		grid := compositeFireGrid(t, at)
		density, shine, _, err := bakeFlame(grid)
		if err != nil {
			t.Fatal(err)
		}
		want, glow, _ := grid.asked()
		total, apart, light, errorLight := 0.0, 0.0, 0.0, 0.0
		for i, d := range want {
			total += float64(d)
			apart += math.Abs(float64(d - density[i]))
			for c := range 3 {
				expected := float64(d) * float64(glow[3*i+c])
				actual := float64(density[i]) * float64(shine[3*i+c])
				light += expected
				errorLight += math.Abs(expected - actual)
			}
		}
		if total <= 0 || light <= 0 {
			t.Fatal("empty composite fixture")
		}
		t.Logf("at %v: density %.8g, emission %.8g", at, apart/total, errorLight/light)
		if apart/total > 1e-4 || errorLight/light > 1e-4 {
			t.Fatal("composite field changed absorption or emitted light")
		}
	}
}

func TestCompositeFlameGridRejectsAnUnrelatedGlow(t *testing.T) {
	grid := compositeFireGrid(t, 0)
	other := compositeFireGrid(t, time.Second)
	grid.shine = other.shine
	if _, ok := flameField(grid); ok {
		t.Fatal("incoherent field clocks accepted")
	}
	if _, _, _, err := bakeFlame(grid); err == nil {
		t.Fatal("mismatched emission was baked")
	}
}

func TestScheduledFieldGridKeepsEachIgnitionAge(t *testing.T) {
	fire := flame.Fire{Radius: .5, Height: 2, Absorption: .8, Hottest: measure.Kelvin(1800), Edge: measure.Kelvin(900), Seed: 17,
		Intermittency: .06, HotTongues: true}
	field, err := flame.NewField([]flame.Ignition{
		{At: maths.V(-.8, 0, 0), Fire: fire, Start: time.Second, Duration: 2 * time.Second, Fade: 500 * time.Millisecond},
		{At: maths.V(.8, 0, 0), Fire: fire, Start: 2 * time.Second, Duration: 2 * time.Second, Fade: 500 * time.Millisecond},
	}, 1e-5)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Duration{2250 * time.Millisecond, 3750 * time.Millisecond} {
		body := field.Tick(at).(*flame.Field)
		bounds := body.Bounds()
		grid := boxed{med: body, shine: body.Shine(), low: bounds.Min, high: bounds.Max, cells: [3]int{31, 25, 15}}
		density, shine, _, err := bakeFlame(grid)
		if err != nil {
			t.Fatal(err)
		}
		truth, glow, _ := grid.asked()
		sum, apart := 0.0, 0.0
		for i, d := range truth {
			for c := range 3 {
				want := float64(d * glow[3*i+c])
				got := float64(density[i] * shine[3*i+c])
				sum += math.Abs(want)
				apart += math.Abs(want - got)
			}
		}
		if sum <= 0 || apart/sum > 1e-4 {
			t.Fatalf("scheduled field at %v relative error %g", at, apart/sum)
		}
	}
}
