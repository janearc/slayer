package flame

import (
	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/radiance"
	"math"
	"testing"
)

func lineSpec() CompositionSpec {
	return CompositionSpec{Reference: measure.Kelvin(1900), Continuum: 0, Lines: []LineSpec{
		{Species: "Na I", Wavelength: measure.Metres(588.995e-9), Radiance: 2, Convention: "air", Source: "https://physics.nist.gov/PhysRefData/Handbook/Tables/sodiumtable2.htm"},
		{Species: "Na I", Wavelength: measure.Metres(589.5924e-9), Radiance: 1, Convention: "air", Source: "https://physics.nist.gov/PhysRefData/Handbook/Tables/sodiumtable2.htm"},
	}}
}

func spectralPower(light bands.Light) float64 {
	first, last := light.Span()
	sum := 0.0
	for b := first; b <= last; b++ {
		sum += light.Power(b)
	}
	return sum
}

func TestCompositionPreservesIntegratedLinesAndItsAuthoredThermalLaw(t *testing.T) {
	spec := lineSpec()
	profile, err := NewComposition(spec)
	if err != nil {
		t.Fatal(err)
	}
	// Independent band construction and Planck ratio, not a colour-name check.
	atHot := bands.Lines(map[float64]float64{588.995: 2, 589.5924: 1})
	got := profile.Spectrum(spec.Reference)
	lo, hi := atHot.Span()
	for b := lo; b <= hi; b++ {
		if math.Abs(got.Power(b)-atHot.Power(b)) > 1e-12 {
			t.Fatal("line energy moved or changed")
		}
	}
	if math.Abs(spectralPower(got)-3) > 1e-12 {
		t.Fatal("line radiance was treated as spectral density")
	}
	cold := measure.Kelvin(1200)
	want := 0.0
	for _, line := range spec.Lines {
		want += line.Radiance * radiance.Planck(cold, line.Wavelength).InWattsPerNanometre() / radiance.Planck(spec.Reference, line.Wavelength).InWattsPerNanometre()
	}
	if math.Abs(spectralPower(profile.Spectrum(cold))-want) > 1e-12 {
		t.Fatal("wrong temperature response")
	}
	spec.Lines[0].Radiance = 900
	copy := profile.Lines()
	copy[0].Radiance = 800
	if math.Abs(spectralPower(profile.Spectrum(profile.Reference()))-3) > 1e-12 {
		t.Fatal("caller changed immutable composition")
	}
}

func TestCompositionAddsContinuumAndRejectsInvalidInputs(t *testing.T) {
	spec := lineSpec()
	spec.Continuum = .25
	profile, err := NewComposition(spec)
	if err != nil {
		t.Fatal(err)
	}
	heat := measure.Kelvin(1700)
	expected := spectralPower(Glowing(heat)) * .25
	lineOnly, _ := NewComposition(lineSpec())
	expected += spectralPower(lineOnly.Spectrum(heat))
	if math.Abs(spectralPower(profile.Spectrum(heat))-expected) > 1e-10 {
		t.Fatal("continuum not added")
	}
	for _, alter := range []func(*CompositionSpec){
		func(s *CompositionSpec) { s.Continuum = -1 }, func(s *CompositionSpec) { s.Continuum = 2 },
		func(s *CompositionSpec) { s.Continuum = math.NaN() }, func(s *CompositionSpec) { s.Reference = measure.Kelvin(0) },
		func(s *CompositionSpec) { s.Lines[0].Wavelength = measure.Metres(0) },
		func(s *CompositionSpec) { s.Lines[0].Radiance = math.Inf(1) },
		func(s *CompositionSpec) { s.Lines[0].Radiance = -1 },
		func(s *CompositionSpec) { s.Lines[0].Source = "" },
		func(s *CompositionSpec) { s.Lines[0].Convention = "guess" },
		func(s *CompositionSpec) { s.Lines = nil },
	} {
		bad := lineSpec()
		alter(&bad)
		if _, err := NewComposition(bad); err == nil {
			t.Fatal("invalid composition accepted")
		}
	}
}
