package flame

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

func TestFlameRejectsIncompatiblePaletteAndUnrepresentableDiameter(t *testing.T) {
	profile, err := NewComposition(lineSpec())
	if err != nil {
		t.Fatal(err)
	}
	fire := campfire
	fire.Composition = profile
	if _, err := NewBody(fire, 1); !errors.Is(err, errFlame) {
		t.Fatalf("incompatible reference: %v", err)
	}
	fire = campfire
	fire.Radius = math.MaxFloat64
	if _, err := NewBody(fire, 1); !errors.Is(err, errFlame) {
		t.Fatalf("overflowing diameter: %v", err)
	}
}

func TestSpectralReferenceMustHaveRepresentablePlanckRadiance(t *testing.T) {
	spec := lineSpec()
	spec.Reference = measure.Kelvin(1)
	if _, err := NewComposition(spec); !errors.Is(err, errFlame) {
		t.Fatalf("zero reference radiance: %v", err)
	}
	profile, err := NewComposition(lineSpec())
	if err != nil {
		t.Fatal(err)
	}
	for _, heat := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if got := spectralPower(profile.Spectrum(measure.Kelvin(heat))); got != 0 {
			t.Fatalf("invalid temperature %g emitted %g", heat, got)
		}
	}
}

func TestSootRetainsTheTemperatureUsedByItsRadiance(t *testing.T) {
	heat := measure.Kelvin(1735)
	soot, err := NewSoot(heat, .4, measure.Metres(550e-9), Visible)
	if err != nil {
		t.Fatal(err)
	}
	if soot.Heat() != heat {
		t.Fatal("soot lost its thermal reference")
	}
}
