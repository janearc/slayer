package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

func TestOpticalRecipeAcceptsItsExactDimensionAndPointingEdges(t *testing.T) {
	for _, opening := range []float64{5, 90} {
		for _, altitude := range []float64{0, 90} {
			spec := opticalSpec()
			spec.Diameter = measure.Metres(200)
			spec.Opening = measure.Degrees(opening)
			spec.Altitude = measure.Degrees(altitude)
			if _, err := Observatory(spec); err != nil {
				t.Fatalf("opening %g altitude %g: %v", opening, altitude, err)
			}
		}
	}
	spec := opticalSpec()
	spec.Diameter = measure.Metres(10)
	spec.Aperture = measure.Metres(math.Nextafter(6, 0))
	if _, err := Observatory(spec); err != nil {
		t.Fatalf("aperture below sixty percent: %v", err)
	}
	spec.Aperture = measure.Metres(6)
	if _, err := Observatory(spec); err == nil {
		t.Fatal("accepted aperture at sixty percent of diameter")
	}
}

func TestOpticalRecipeRejectsEveryIndependentInvalidDimension(t *testing.T) {
	for _, value := range []float64{0, -math.SmallestNonzeroFloat64, math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, field := range []string{"diameter", "aperture"} {
			spec := opticalSpec()
			if field == "diameter" {
				spec.Diameter = measure.Metres(value)
			} else {
				spec.Aperture = measure.Metres(value)
			}
			if _, err := Observatory(spec); err == nil {
				t.Fatalf("accepted %s=%g", field, value)
			}
		}
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, field := range []string{"opening", "azimuth", "altitude"} {
			spec := opticalSpec()
			switch field {
			case "opening":
				spec.Opening = measure.Degrees(value)
			case "azimuth":
				spec.Azimuth = measure.Degrees(value)
			case "altitude":
				spec.Altitude = measure.Degrees(value)
			}
			if _, err := Observatory(spec); err == nil {
				t.Fatalf("accepted %s=%g", field, value)
			}
		}
	}
	for _, row := range []struct {
		field string
		value float64
	}{
		{"diameter", math.Nextafter(200, math.Inf(1))}, {"opening", 4.5}, {"opening", 90.5},
		{"altitude", -.5}, {"altitude", 90.5},
	} {
		spec := opticalSpec()
		switch row.field {
		case "diameter":
			spec.Diameter = measure.Metres(row.value)
		case "opening":
			spec.Opening = measure.Degrees(row.value)
		case "altitude":
			spec.Altitude = measure.Degrees(row.value)
		}
		if _, err := Observatory(spec); err == nil {
			t.Fatalf("accepted %s=%g", row.field, row.value)
		}
	}
}
