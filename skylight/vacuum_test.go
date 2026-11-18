package skylight

import (
	"fmt"
	"testing"
)

// skyless is nil when no bin of the field holds any light and the sun
// reaches the ground whole above the horizon and not at all below it.
func skyless(field *Field) error {
	for index, value := range field.radiance {
		if value != none {
			return fmt.Errorf("bin %d holds %g", index, value)
		}
	}
	risen := []float64{0, 30, 60, 89}      // picked: degrees above it
	hidden := []float64{91, 120, 150, 180} // picked: degrees below it
	for _, zenith := range risen {
		if got := field.Transmittance(none, zenith*degree); got != one {
			return fmt.Errorf("sun at %g degrees: %g", zenith, got)
		}
	}
	for _, zenith := range hidden {
		if got := field.Transmittance(none, zenith*degree); got != none {
			return fmt.Errorf("sun at %g degrees: %g", zenith, got)
		}
	}
	return nil
}

// with no air there is no sky, at any height, and the sun reaches the
// ground whole wherever it is up and not at all where it has set.
func TestVacuumHasNoSky(t *testing.T) {
	column := uniformColumn(earthRadius, airTop, Shell{}, none)
	field, _, err := Solve(column, keptGrid(), fewPhotons, seedOf,
		workersOf)
	if err != nil {
		t.Fatal(err)
	}
	if err := skyless(field); err != nil {
		t.Error(err)
	}
}

// the anti-test: a thin Rayleigh sky over the same ground is not
// skyless, so the check would see light that should not be there.
func TestThinAirHasASky(t *testing.T) {
	thin := Shell{GasScattering: thinDepth / airTop}
	column := uniformColumn(earthRadius, airTop, thin, none)
	field, _, err := Solve(column, keptGrid(), fewPhotons, seedOf,
		workersOf)
	if err != nil {
		t.Fatal(err)
	}
	if skyless(field) == nil {
		t.Error("thin air has no sky")
	}
}
