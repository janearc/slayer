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

func fireGridFixture(t *testing.T, seed uint64, at time.Duration) boxed {
	t.Helper()
	body, err := flame.NewBody(flame.Fire{Radius: 0.7, Height: 2.4,
		Absorption: 1, Hottest: measure.Kelvin(1603), Edge: measure.Kelvin(900), Seed: seed, Intermittency: .08, HotTongues: true}, 2e-5)
	if err != nil {
		t.Fatal(err)
	}
	body = body.Tick(at).(*flame.Body)
	bounds := body.Bounds()
	return boxed{med: body, shine: body.Shine(), low: bounds.Min, high: bounds.Max,
		cells: [3]int{19, 33, 17}}
}

func TestFlameGridUsesItsTimeSeedAndEmission(t *testing.T) {
	for _, test := range []struct {
		seed uint64
		at   time.Duration
	}{
		{5, 0}, {1<<62 | 31, 137 * time.Millisecond}, {71, 630 * time.Millisecond},
	} {
		grid := fireGridFixture(t, test.seed, test.at)
		density, shine, elapsed, err := bakeFlame(grid)
		if err != nil {
			t.Fatal(err)
		}
		if elapsed <= 0 {
			t.Fatal("no GPU work measured")
		}
		truth, glow, _ := grid.asked()
		if len(density) != len(truth) || len(shine) != len(glow) {
			t.Fatal("wrong grid sizes")
		}
		total, apart, emitted, errorLight := 0.0, 0.0, 0.0, 0.0
		peakDensity, worstDensity, peakLight, worstLight := 0.0, 0.0, 0.0, 0.0
		for i, d := range truth {
			total += float64(d)
			apart += math.Abs(float64(d - density[i]))
			peakDensity = max(peakDensity, float64(d))
			worstDensity = max(worstDensity, math.Abs(float64(d-density[i])))
			if !(density[i] >= 0) || math.IsInf(float64(density[i]), 0) {
				t.Fatalf("invalid density in cell %d: %g", i, density[i])
			}
			for channel := range 3 {
				index := i*3 + channel
				want := float64(d) * float64(glow[index])
				got := float64(density[i]) * float64(shine[index])
				emitted += want
				errorLight += math.Abs(want - got)
				peakLight = max(peakLight, want)
				worstLight = max(worstLight, math.Abs(want-got))
				if math.IsNaN(got) || math.IsInf(got, 0) || got < 0 {
					t.Fatal("invalid grid emission")
				}
			}
		}
		if total <= 0 || emitted <= 0 {
			t.Fatal("fixture contains no fire")
		}
		t.Logf("seed %d at %v: density %.8g, emission %.8g", test.seed, test.at, apart/total, errorLight/emitted)
		if !(apart/total <= 1e-4 && errorLight/emitted <= 1e-4) {
			t.Fatal("GPU field differs from the CPU model")
		}
		// An authored per-cell budget also catches a locally missing flame.
		// Normalize by the grid peak so near-zero cells stay meaningful.
		t.Logf("worst cell: density %.8g, emission %.8g of peak",
			worstDensity/peakDensity, worstLight/peakLight)
		if !(worstDensity/peakDensity <= 1e-3 && worstLight/peakLight <= 1e-3) {
			t.Fatal("a local GPU error escaped the aggregate comparison")
		}
	}
}

func TestFlameGridRefusesCustomOrMismatchedEmission(t *testing.T) {
	grid := fireGridFixture(t, 5, 0)
	other := fireGridFixture(t, 6, 0)
	later := fireGridFixture(t, 5, time.Second)
	for _, alter := range []func(*boxed){
		func(g *boxed) { g.shine = nil },
		func(g *boxed) { g.shine = other.shine },
		func(g *boxed) { g.shine = later.shine },
		func(g *boxed) { g.paint = banded{} },
	} {
		invalid := grid
		alter(&invalid)
		if _, ok := flameField(invalid); ok {
			t.Fatal("custom field bypassed CPU fallback")
		}
		if _, _, _, err := bakeFlame(invalid); err == nil {
			t.Fatal("unsupported paired field accepted")
		}
	}
	if _, ok := flameField(grid); !ok {
		t.Fatal("standard paired field refused")
	}
}

func TestFlameGridRefusesInvalidCellCounts(t *testing.T) {
	grid := fireGridFixture(t, 5, 0)
	grid.cells = [3]int{0, 33, 17}
	if _, _, _, err := bakeFlame(grid); err == nil {
		t.Fatal("zero-width grid accepted")
	}
}

func TestFlameGridRetainsAUniformTemperature(t *testing.T) {
	body, err := flame.NewBody(flame.Fire{Radius: 1, Height: 2, Absorption: 0.7,
		Hottest: measure.Kelvin(1100), Edge: measure.Kelvin(1100), Seed: 1}, 1e-5)
	if err != nil {
		t.Fatal(err)
	}
	bounds := body.Bounds()
	grid := boxed{med: body, shine: body.Shine(), low: bounds.Min, high: bounds.Max, cells: [3]int{7, 9, 5}}
	_, glow, _, err := bakeFlame(grid)
	if err != nil {
		t.Fatal(err)
	}
	want := body.Shine().At(maths.V(0, 0.5, 0))
	for i := 0; i < len(glow); i += 3 {
		for c, value := range []float64{want.R, want.G, want.B} {
			if math.Abs(float64(glow[i+c])-value) > 1e-6*max(1e-30, value) {
				t.Fatal("constant-temperature lookup changed")
			}
		}
	}
}

func TestCompositionGridUsesTheSameSpectralTableOnCPUAndGPU(t *testing.T) {
	profile, err := flame.NewComposition(flame.CompositionSpec{Reference: measure.Kelvin(1900),
		Continuum: .2, Lines: []flame.LineSpec{{Species: "Na I", Wavelength: measure.Metres(588.995e-9),
			Radiance: .02, Convention: "air", Source: "https://physics.nist.gov/PhysRefData/Handbook/Tables/sodiumtable2.htm"}}})
	if err != nil {
		t.Fatal(err)
	}
	body, err := flame.NewBody(flame.Fire{Radius: .7, Height: 2.4, Absorption: 1,
		Hottest: measure.Kelvin(1900), Edge: measure.Kelvin(900), Seed: 71, Composition: profile,
		HotTongues: true, Intermittency: .06}, 2e-5)
	if err != nil {
		t.Fatal(err)
	}
	body = body.Tick(730 * time.Millisecond).(*flame.Body)
	bounds := body.Bounds()
	grid := boxed{med: body, shine: body.Shine(), low: bounds.Min, high: bounds.Max, cells: [3]int{19, 33, 17}}
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
	if !(sum > 0 && apart/sum <= 1e-4) {
		t.Fatalf("spectral GPU grid relative error %g", apart/sum)
	}
}
