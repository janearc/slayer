package materials

import (
	"math"
	"testing"
)

func TestConductorEstimateReproducesNormalReflectance(t *testing.T) {
	source, err := newSource("test reflectance", "sample", "basis", "R measured")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ r, k float64 }{{.2, 1}, {.5, 2}, {.8, 4}} {
		got, err := conductorFromReflectance(550, row.r, source)
		if err != nil {
			t.Fatal(err)
		}
		if got.N() != 1 || math.Abs(got.K()-row.k) > 1e-12 || got.Wavelength() != 550 {
			t.Fatalf("R=%g: got n=%g k=%g at%g", row.r, got.N(), got.K(), got.Wavelength())
		}
		if !got.Where().Estimated() || got.Where().Row() != source.Row() {
			t.Fatal("fit lost explicit estimate provenance")
		}
	}
	if source.Estimated() {
		t.Fatal("fit mutated the original measured source")
	}
}

func TestConductorEstimateRefusesUnphysicalOrUnattributedInput(t *testing.T) {
	source, _ := newSource("test", "sample", "basis", "R")
	for _, r := range []float64{-1, 0, 1, 2, math.NaN(), math.Inf(1)} {
		if _, err := conductorFromReflectance(550, r, source); err == nil {
			t.Fatalf("accepted R=%g", r)
		}
	}
	for _, w := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := conductorFromReflectance(w, .5, source); err == nil {
			t.Fatalf("accepted wavelength=%g", w)
		}
	}
	if _, err := conductorFromReflectance(550, .5, Source{}); err == nil {
		t.Fatal("accepted no provenance")
	}
}

func TestNamedNickelAluminiumBronzeCarriesCompositionDensityAndEstimate(t *testing.T) {
	kind, err := ByName("nickel-aluminium-bronze")
	if err != nil {
		t.Fatal(err)
	}
	if kind.Density() != 7.64 || kind.DensityFrom().Paper() == "" {
		t.Fatal("missing cited density")
	}
	wanted := map[string]float64{"Cu": .815, "Al": .09, "Ni": .045, "Fe": .04, "Mn": .01}
	mass := kind.made.Mass()
	for symbol, share := range wanted {
		actual := kind.made.parts[symbol] * weights[symbol] / mass
		if math.Abs(actual-share) > 1e-12 {
			t.Fatalf("%s mass share=%g want%g", symbol, actual, share)
		}
	}
	for band, r := range map[Band]float64{Red: .46, Green: .40, Blue: .27} {
		read := kind.Optics().At(band)
		n, k := read.N(), read.K()
		actual := ((n-1)*(n-1) + k*k) / ((n+1)*(n+1) + k*k)
		if !read.Where().Estimated() || math.Abs(actual-r) > 1e-12 {
			t.Fatalf("%s proxy reflectance=%g", band, actual)
		}
	}
	titanium, err := ByName("titanium")
	if err != nil {
		t.Fatal(err)
	}
	if titanium.Optics().At(Red).Where().Estimated() || titanium.Optics().At(Red).N() != 2.67 {
		t.Fatal("adding bronze changed measured titanium")
	}
}
