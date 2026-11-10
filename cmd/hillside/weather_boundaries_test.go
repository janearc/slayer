package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/set"
)

func weatherBoundaryPlanet(t *testing.T, extra string) set.Planet {
	t.Helper()
	text := strings.Replace(spaceScene, "neon: 0.5, methane: 0.5", "neon: 1", 1)
	text = strings.Replace(text, "    columns:", extra+"    columns:", 1)
	read, err := set.Read("weather-boundary.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	made, has := read.Planet()
	if !has {
		t.Fatal("fixture has no planet")
	}
	return made
}

func TestPlanetCookingRefusesAColumnWithoutHaze(t *testing.T) {
	made := weatherBoundaryPlanet(t, "    cooking: {column: clear, time: 10}\n")
	gases := map[string]planet.Gas{"neon": planet.Neon()}
	if _, err := lit(planet.Body{}, made, gases); err == nil || !strings.Contains(err.Error(), "no haze") {
		t.Fatalf("cooking without haze: %v", err)
	}
	if _, err := weathered(planet.Body{}, made, gases); err == nil || !strings.Contains(err.Error(), "no haze") {
		t.Fatalf("weather lost cooking cause: %v", err)
	}
}

func TestWeatherConstructionPropagatesGridAndCacheFailures(t *testing.T) {
	extra := "    weather:\n" +
		"      cells: 16\n      day: 100000\n" +
		"      drag: 100000\n      relax: 100000\n" +
		"      storms: {rate: 1.0e-20, wide: 1000, strength: 0.001}\n" +
		"      seed: 5\n      moment: 0\n      every: 1\n"
	made := weatherBoundaryPlanet(t, extra)
	if _, err := weathered(planet.Body{}, made, nil); err == nil || !strings.Contains(err.Error(), "grid") {
		t.Fatalf("zero radius grid: %v", err)
	}
	zone, err := planet.NewZone(-90, 90, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := planet.NewBody(1e7, 10, .8, []planet.Zone{zone})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", "")
	t.Setenv("SLAYER_WEATHER", "")
	if _, err := weathered(body, made, nil); err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Fatalf("cache home refusal: %v", err)
	}
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_WEATHER", blocked)
	if _, err := weathered(body, made, nil); err == nil || !strings.Contains(err.Error(), blocked) {
		t.Fatalf("cache writer refusal: %v", err)
	}
}
