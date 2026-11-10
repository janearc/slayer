package flame

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"testing"
	"time"
)

func TestCompositionAcceptsZeroPowerAndTheExactLineBudget(t *testing.T) {
	line := LineSpec{Species: "fixture", Source: "authored boundary test", Convention: "vacuum", Wavelength: measure.Metres(589e-9), Radiance: 0}
	for _, count := range []int{1, 32, 33} {
		lines := make([]LineSpec, count)
		for i := range lines {
			lines[i] = line
		}
		profile, err := NewComposition(CompositionSpec{Reference: measure.Kelvin(1800), Continuum: 1, Lines: lines})
		if count > 32 {
			if err == nil {
				t.Fatal("33 lines accepted")
			}
			continue
		}
		if err != nil {
			t.Fatalf("%d zero-power lines refused: %v", count, err)
		}
		if len(profile.Lines()) != count {
			t.Fatal("line provenance lost")
		}
	}
}

func TestFieldKeepsItsExactCapacityAndNanosecondWindows(t *testing.T) {
	for _, count := range []int{64, 65} {
		cells := make([]Ignition, count)
		for i := range cells {
			cells[i] = Ignition{Fire: campfire}
		}
		field, err := NewField(cells, 1)
		if count > 64 {
			if err == nil {
				t.Fatal("65 patches accepted")
			}
			continue
		}
		if err != nil || len(field.Pieces()) != count {
			t.Fatalf("64 patches refused: %v", err)
		}
	}
	field, err := NewField([]Ignition{{Fire: campfire, Start: time.Second, Duration: time.Nanosecond}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(field.Tick(time.Second).(*Field).Pieces()) != 1 || len(field.Tick(time.Second+time.Nanosecond).(*Field).Pieces()) != 0 {
		t.Fatal("nanosecond window endpoints changed")
	}
	fading, err := NewField([]Ignition{{Fire: campfire, Start: time.Second, Duration: 4 * time.Nanosecond, Fade: time.Nanosecond}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(0, .05, 0)
	if got := fading.Tick(time.Second).(*Field).Density(point); got != 0 {
		t.Fatalf("nanosecond fade begins at opacity %g", got)
	}
	if got := fading.Tick(time.Second + time.Nanosecond).(*Field).Density(point); !(got > 0) {
		t.Fatal("nanosecond fade lost its plateau")
	}
	if got := field.Shine().Tick(0).At(point); got != (maths.Light{}) {
		t.Fatal("inactive field emits before ignition")
	}
	if len(field.Sources(0)) != 0 || len(field.Sources(-1)) != 0 {
		t.Fatal("nonpositive quadrature produces sources")
	}
}

func TestIsothermalColourPaletteRemainsValid(t *testing.T) {
	heat := measure.Kelvin(1000)
	palette, err := Colours(ColourSpec{Coolest: heat, Hottest: heat, Exposure: 1})
	if err != nil {
		t.Fatal(err)
	}
	value := palette.At(heat)
	if !(value.R+value.G+value.B > 0) {
		t.Fatal("isothermal palette lost its source")
	}
	for _, k := range []float64{1, 500, 1000, 2000} {
		if palette.At(measure.Kelvin(k)) != value {
			t.Fatal("isothermal palette changed with lookup temperature")
		}
	}
}
