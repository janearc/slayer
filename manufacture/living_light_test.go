package manufacture

import (
	"math"
	"testing"
)

func TestLivingLightSpectrumAndFlash(t *testing.T) {
	spec := DinoflagellateLight()
	blue, err := NewLivingLight(spec)
	if err != nil {
		t.Fatal(err)
	}
	peak := blue.Colour()
	if peak.B <= peak.G || peak.B <= peak.R ||
		math.Abs(peak.Luminance()-spec.Luminance) > 1e-12 {
		t.Fatalf("unexpected 475 nm colour or luminance: %+v", peak)
	}
	if blue.Flash(-1).Luminance() != 0 ||
		blue.Flash(spec.FlashSeconds).Luminance() != 0 ||
		math.Abs(blue.Flash(spec.FlashSeconds/2).Luminance()-spec.Luminance) > 1e-12 {
		t.Fatal("flash envelope did not respect its duration and peak")
	}
	if blue.Spec() != spec {
		t.Fatal("spectral evidence and flash parameters were lost")
	}
	if livingLightSpectrum(spec, spec.PeakNanometres) != 1 ||
		math.Abs(livingLightSpectrum(spec, spec.PeakNanometres+spec.FWHMNanometres/2)-.5) > 1e-12 {
		t.Fatal("emission peak or band width is incorrect")
	}
}

func TestLivingLightRejectsInvalidSpec(t *testing.T) {
	for _, change := range []func(*LivingLightSpec){
		func(s *LivingLightSpec) { s.PeakNanometres = math.NaN() },
		func(s *LivingLightSpec) { s.PeakNanometres = 0 },
		func(s *LivingLightSpec) { s.FWHMNanometres = 0 },
		func(s *LivingLightSpec) { s.Luminance = -1 },
		func(s *LivingLightSpec) { s.FlashSeconds = 0 },
	} {
		spec := DinoflagellateLight()
		change(&spec)
		if _, err := NewLivingLight(spec); err == nil {
			t.Fatalf("accepted invalid spec %+v", spec)
		}
	}
}
