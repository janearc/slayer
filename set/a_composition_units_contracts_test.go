package set

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/flame"
)

func compositionUnitInput(t *testing.T, raw string) (*flame.CompositionSpec, error) {
	t.Helper()
	node, err := parse("composition-unit.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var spec *flame.CompositionSpec
	err = (reader{file: "composition-unit.yaml"}).fireComposition(&spec)(nil, node)
	return spec, err
}

const compositionUnitText = "continuum: 0.7\nlines:\n" +
	"  - species: Na I\n" +
	"    source: https://physics.nist.gov/PhysRefData/Handbook/Tables/sodiumtable2.htm\n" +
	"    convention: air\n" +
	"    wavelength: {value: 589.5924, unit: nm}\n" +
	"    radiance: {value: 0.01, unit: W/m^2/sr}\n"

func TestEmissionLineConvertsNanometresBeforeBuildingAnySpectrum(t *testing.T) {
	spec, err := compositionUnitInput(t, compositionUnitText)
	if err != nil {
		t.Fatal(err)
	}
	if spec == nil || len(spec.Lines) != 1 {
		t.Fatal("missing parsed composition")
	}
	line := spec.Lines[0]
	if !(math.Abs(line.Wavelength.InMetres()-589.5924e-9) <= 1e-20) {
		t.Fatalf("air wavelength %.17g m, want 589.5924 nm", line.Wavelength.InMetres())
	}
	if line.Species != "Na I" || line.Convention != "air" || line.Radiance != .01 || spec.Continuum != .7 {
		t.Fatal("emission provenance or line power changed")
	}
}

func TestCompositionReaderLimitsRunBeforeSpectrumConstruction(t *testing.T) {
	for _, row := range []struct {
		before, prefix, key string
		low, high           float64
	}{
		{"continuum: 0.7", "continuum: ", "continuum", 0, 1},
		{"wavelength: {value: 589.5924", "wavelength: {value: ", "value", 1, 1e6},
		{"radiance: {value: 0.01", "radiance: {value: ", "value", 0, 1e12},
	} {
		for _, value := range []float64{math.Nextafter(row.low, math.Inf(-1)), math.Nextafter(row.high, math.Inf(1))} {
			raw := strings.Replace(compositionUnitText, row.before, row.prefix+fmt.Sprintf("%.17g", value), 1)
			_, err := compositionUnitInput(t, raw)
			want := fmt.Sprintf("%s takes a number from %v to %v", row.key, row.low, row.high)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s: %v, want %s", row.before, err, want)
			}
		}
		for _, value := range []float64{row.low, row.high} {
			raw := strings.Replace(compositionUnitText, row.before, row.prefix+fmt.Sprintf("%.17g", value), 1)
			if _, err := compositionUnitInput(t, raw); err != nil {
				t.Fatalf("legal parsing endpoint %s=%g: %v", row.prefix, value, err)
			}
		}
	}
	line := strings.SplitN(compositionUnitText, "lines:\n", 2)[1]
	for _, count := range []int{32, 33} {
		raw := "continuum: 0.7\nlines:\n" + strings.Repeat(line, count)
		spec, err := compositionUnitInput(t, raw)
		if count == 32 {
			if err != nil || spec == nil || len(spec.Lines) != 32 {
				t.Fatalf("32 lines: %v", err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "lines takes at most 32 entries") {
			t.Fatalf("33 lines: %v", err)
		}
	}
}
