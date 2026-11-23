package named

import (
	"io"
	"os"
	"testing"
	"time"
)

// TestOrbitalReliefProductionScene builds the actual proof configuration,
// including dry sites, all landmarks, its moving camera and flare clock.
func TestOrbitalReliefProductionScene(t *testing.T) {
	if os.Getenv("ORBITAL_RELIEF_FULL") != "1" {
		t.Skip("set ORBITAL_RELIEF_FULL=1 to build the full surface proof")
	}
	view, err := orbitalScene().Build(map[string]any{
		"floor-hex-metres": 10, "surface-structures": 24,
		"surface-landforms": 24, "surface-district": 24,
		"surface-style": "mixed", "surface-tour": true,
		"surface-tour-duration": "15s", "surface-relief": true,
		"zappy-flares": true, "day": "0s",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Timed() || len(view.World().Things) < 25 || len(view.World().Lamps) != 3 {
		t.Fatal("production surface lost geometry, camera or lighting")
	}
	if err := view.Advance(8 * time.Second); err != nil {
		t.Fatal(err)
	}
}

// TestOrbitalCityProductionScene constructs the actual full-detail combination
// used by the film, including the populated district and all nine lights.
func TestOrbitalCityProductionScene(t *testing.T) {
	if os.Getenv("ORBITAL_CITY_FULL") != "1" {
		t.Skip("set ORBITAL_CITY_FULL=1 to build the full city proof")
	}
	view, err := orbitalScene().Build(map[string]any{
		"floor-hex-metres": 10, "surface-structures": 24,
		"surface-landforms": 24, "surface-district": 24,
		"surface-style": "mixed", "surface-tour": true,
		"surface-tour-duration": "15s", "surface-relief": true,
		"surface-city": true, "zappy-flares": true, "day": "0s",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Timed() || len(view.World().Things) < 25 || len(view.World().Lamps) != 9 {
		things, lamps := len(view.World().Things), len(view.World().Lamps)
		t.Fatalf("production city lost geometry or lights: %d things, %d lamps", things, lamps)
	}
	if err := view.Advance(8745454545 * time.Nanosecond); err != nil {
		t.Fatal(err)
	}
}
