package flame

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

func TestCompositionPaletteTracksThePrescribedSpectrumAcrossItsRange(t *testing.T) {
	spec := lineSpec()
	spec.Continuum = .4
	profile, err := NewComposition(spec)
	if err != nil {
		t.Fatal(err)
	}
	const cold, hot, exposure = 900.0, 1900.0, 1e-5
	table := compositionGlowing(profile, cold, hot, exposure)
	for _, temperature := range []float64{0, 900, 1100, 1400, 1700, 1900, 2400} {
		expectedHeat := measure.Kelvin(math.Max(cold, math.Min(hot, temperature)))
		spectrum := profile.Spectrum(expectedHeat).Scale(exposure)
		r, g, b := srgb.Light(observer.Swatch(spectrum))
		expected := maths.RGB(math.Max(0, r), math.Max(0, g), math.Max(0, b))
		got := table.At(measure.Kelvin(temperature))
		difference := math.Abs(got.R-expected.R) + math.Abs(got.G-expected.G) + math.Abs(got.B-expected.B)
		total := expected.R + expected.G + expected.B
		// Interpolation is approximate; endpoint clamping is exact. The half
		// percent interior bound includes curvature between adjacent table knots.
		bound := .005 * total
		if temperature <= cold || temperature >= hot {
			bound = 1e-12 * total
		}
		if math.IsNaN(difference) || !(difference <= bound) {
			t.Fatalf("temperature %g: palette %v direct spectrum %v, error %g bound %g", temperature, got, expected, difference, bound)
		}
	}
}
