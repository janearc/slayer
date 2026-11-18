package materials

import (
	"math"
	"testing"
)

func TestGalliumArsenideMatchesThePublishedOptics(t *testing.T) {
	made, err := ByName("gallium-arsenide")
	if err != nil {
		t.Fatal(err)
	}
	formula, err := NewFormula(map[string]float64{"Ga": 1, "As": 1})
	if err != nil || math.Abs(formula.Mass()-(69.723+74.921595)) > 1e-12 {
		t.Fatal("GaAs formula differs from CIAAW element weights", formula, err)
	}
	for band, wavelength := range map[Band]string{Red: "0.6199", Green: "0.5391", Blue: "0.4592"} {
		_, n, k, line := publishedRow(t, "main/GaAs/nk/Aspnes.yml", wavelength)
		got := made.Optics().At(band)
		if got.N() != n || got.K() != k || got.Where().Row() != line {
			t.Fatalf("%s differs from its source: n=%g k=%g", band, got.N(), got.K())
		}
	}
}
