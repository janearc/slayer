package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func TestAPlanetInTheSkyNeedsGroundAndAHiddenSun(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "    gravity: 24.8", "    gravity: 24.8\n    sky: {azimuth: 180, altitude: 35, distance: 1.0e9}")
	if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), "needs ground under it") {
		t.Fatalf("floating sky planet: %v", err)
	}
	block := spaceScene[strings.Index(spaceScene, "  planet:"):]
	block = strings.Replace(block, "    gravity: 24.8", "    gravity: 24.8\n    sky: {azimuth: 180, altitude: 35, distance: 1.0e9}", 1)
	given.set = onFixture(t, "  terrain:", block+"  terrain:")
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "light: moon", "light: sun\n  sun: {azimuth: 180, altitude: 0}", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), "not hidden by the earth") {
		t.Fatalf("daylight twilight planet: %v", err)
	}
}

func TestPlanetBodyAndRingGlowPreserveTheirRadiusRefusals(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "", "")
	stage := staging{given: given, world: thing.NewWorld()}
	if _, err := stage.body(set.Planet{}); err == nil || !strings.Contains(err.Error(), "radius 0") {
		t.Fatalf("empty body: %v", err)
	}
	ring, err := planet.NewRing(2, 3, .1, .5, 0)
	if err != nil {
		t.Fatal(err)
	}
	body, err := (planet.Body{}).Ringed([]planet.Ring{ring})
	if err != nil {
		t.Fatal(err)
	}
	made := weatherBoundaryPlanet(t, "    xrays: {times: 1, of: jupiter, band: 3}\n")
	if err := stage.ringed(body, made, maths.YAxis, maths.Vec{}); err == nil || !strings.Contains(err.Error(), "planet of 0 m") {
		t.Fatalf("zero-radius ring glow: %v", err)
	}
}

func TestPinnedMalformedMethaneFailsThroughThePlanetCaller(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "", "")
	read := readBoundarySet(t, given.set)
	made, has := read.Planet()
	if !has {
		t.Fatal("fixture has no planet")
	}
	data := []byte("a table which is not numeric\n")
	sum := sha256.Sum256(data)
	path := filepath.Join(os.Getenv("SLAYER_SHELF"), "karkoschka-1994-methane.txt")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	catalogue := filepath.Join(filepath.Dir(filepath.Dir(given.set)), "sources.yaml")
	content := "sources:\n  - key: karkoschka-1994-methane\n    from: https://example.test/table.txt\n    licence: fixture\n    sha256: " + hex.EncodeToString(sum[:]) + "\n"
	if err := os.WriteFile(catalogue, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	stage := staging{given: given, world: thing.NewWorld(), places: map[string]maths.Vec{made.Name(): {}}}
	_, cause := planet.MethaneAbsorbing(strings.NewReader(string(data)))
	if cause == nil {
		t.Fatal("malformed table fixture parsed")
	}
	if _, err := stage.gases(); err == nil || err.Error() != cause.Error() {
		t.Fatalf("gas loader lost the actual table cause: %v", err)
	}
	if err := stage.planetLit(read, made, maths.ZAxis.Scale(1e9)); err == nil || err.Error() != cause.Error() {
		t.Fatalf("planet caller lost the actual table cause: %v", err)
	}
}
