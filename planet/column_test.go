package planet

import (
	"math"
	"testing"
)

// flat is a cross-section the same at every wavelength.
func flat(area float64) func(float64) float64 {
	return func(float64) float64 { return area }
}

// fourth is a cross-section falling as the wavelength to the fourth,
// area at 550 nm.
func fourth(area float64) func(float64) float64 {
	return func(nm float64) float64 { return area * math.Pow(550/nm, 4) }
}

// mustGas is a gas, or the test stops.
func mustGas(t *testing.T, name string, section func(float64) float64,
	molar float64) Gas {
	t.Helper()
	gas, err := NewGas(name, section, molar)
	if err != nil {
		t.Fatal(err)
	}
	return gas
}

// the molecules over a square metre are the air's weight over one
// molecule's mass: earth's sea-level air, 101,325 pascals under 9.80665,
// at 28.9644 grams a mole, is 2.148e29 of them.
func TestTheColumnHoldsTheAirsWeight(t *testing.T) {
	air := mustGas(t, "air", flat(0), 0.0289644)
	layer, err := NewLayer(0, 101325, []Gas{air}, []float64{1}, Haze{})
	if err != nil {
		t.Fatal(err)
	}
	got := layer.molecules(9.80665)
	want := 101325 / 9.80665 / (0.0289644 / avogadro)
	if !near(got, want, 1e-12) || !near(got, 2.148e29, 1e-3) {
		t.Errorf("%g molecules, want %g", got, want)
	}
}

// a layer's gases scatter by their shares of its molecules, each at its
// own cross-section, and the absorbing ones absorb the same way.
func TestGasesScatterByTheirShares(t *testing.T) {
	light := mustGas(t, "light", fourth(1e-31), 0.004)
	heavy := mustGas(t, "heavy", flat(3e-31), 0.1).Absorbing(flat(2e-30))
	layer, err := NewLayer(1e4, 3e4, []Gas{light, heavy},
		[]float64{0.25, 0.75}, Haze{})
	if err != nil {
		t.Fatal(err)
	}
	gravity := 20.0
	count := 2e4 / gravity / ((0.25*0.004 + 0.75*0.1) / avogadro)
	made := layer.optics(275, gravity)
	scatter := count * (0.25*1e-31*16 + 0.75*3e-31)
	absorb := count * 0.75 * 2e-30
	if !near(made.scatter, scatter, 1e-12) ||
		!near(made.absorb, absorb, 1e-12) {
		t.Errorf("scatter %g, want %g; absorb %g, want %g",
			made.scatter, scatter, made.absorb, absorb)
	}
}

// a layer carries its haze as it is.
func TestALayerCarriesItsHaze(t *testing.T) {
	gas := mustGas(t, "gas", flat(0), 0.02)
	haze, err := NewHaze(1.5, 0.9, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	layer, err := NewLayer(0, 1e5, []Gas{gas}, []float64{1}, haze)
	if err != nil {
		t.Fatal(err)
	}
	made := layer.optics(500, 10)
	if made.haze != 1.5 || made.hazeAlbedo != 0.9 || made.hazeLean != 0.6 {
		t.Errorf("%+v", made)
	}
}

// what a gas, a haze or a layer cannot be is refused.
func TestImpossibleAirIsRefused(t *testing.T) {
	gas := mustGas(t, "gas", flat(0), 0.02)
	if _, err := NewGas("massless", flat(0), 0); err == nil {
		t.Error("a gas of no mass")
	}
	for _, haze := range [][3]float64{{-1, 0.5, 0}, {1, 1.1, 0},
		{1, -0.1, 0}, {1, 0.5, 1}, {1, 0.5, -1}} {
		if _, err := NewHaze(haze[0], haze[1], haze[2]); err == nil {
			t.Errorf("a haze of %v", haze)
		}
	}
	for _, try := range []struct {
		top, bottom float64
		gases       []Gas
		shares      []float64
	}{
		{-1, 1e5, []Gas{gas}, []float64{1}},
		{1e5, 1e5, []Gas{gas}, []float64{1}},
		{0, 1e5, []Gas{gas}, []float64{0.5}},
		{0, 1e5, []Gas{gas, gas}, []float64{1, 0}},
		{0, 1e5, []Gas{gas}, []float64{1, 0}},
		{0, 1e5, nil, nil},
	} {
		if _, err := NewLayer(try.top, try.bottom, try.gases, try.shares,
			Haze{}); err == nil {
			t.Errorf("a layer of %+v", try)
		}
	}
}
