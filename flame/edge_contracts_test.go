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
		temperature := measure.Kelvin(heat)
		if got := spectralPower(profile.spectrum(temperature)); got != 0 {
			t.Fatalf("invalid temperature %g emitted %g", heat, got)
		}
	}
}
