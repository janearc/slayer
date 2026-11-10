package main

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func pressureBoundaryPlanet(t *testing.T, extra string, badZone bool) set.Planet {
	t.Helper()
	text := strings.Replace(spaceScene, "neon: 0.5, methane: 0.5", "neon: 1", 1)
	text = strings.Replace(text, "    columns:", extra+"    columns:", 1)
	text = strings.Replace(text, "    zones:", "      collapsed:\n        - {top: 1.5000000000000004, bottom: 1.5000000000000007, gases: {neon: 1}}\n    zones:", 1)
	if badZone {
		text = strings.Replace(text, "north: 90, column: clear", "north: 90, column: collapsed", 1)
	}
	read, err := set.Read("pressure-boundary.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	made, has := read.Planet()
	if !has {
		t.Fatal("fixture has no planet")
	}
	layers := made.Column("collapsed")
	if len(layers) != 1 {
		t.Fatal("fixture lost its pressure interval")
	}
	top, bottom := layers[0].Pressures()
	// Distinct binary64 bar endpoints can round to one pascal coordinate.
	// This is a unit-conversion boundary fixture, not a realistic thin layer.
	if !(bottom > top) || top*1e5 != bottom*1e5 {
		t.Fatal("pressure conversion no longer collapses this interval")
	}
	return made
}

func requireLayerCause(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "a layer from") {
		t.Fatalf("lost collapsed-layer refusal: %v", err)
	}
}

func TestPlanetBodyPropagatesAPressureConversionRefusal(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "", "")
	stage := staging{given: given, world: thing.NewWorld()}
	for _, test := range []struct {
		extra   string
		badZone bool
	}{
		{"", true},
		{boundaryStorm + "      churn: {column: collapsed, strength: 1}\n", false},
	} {
		_, err := stage.body(pressureBoundaryPlanet(t, test.extra, test.badZone))
		requireLayerCause(t, err)
	}

}

const boundaryStorm = "    storms:\n      seed: 5\n      turbulence: 0\n      eddies: 1\n      stretch: 1\n      edge: 0\n"

func TestCookingChurnAndBothStormMarksKeepTheirLayerCauses(t *testing.T) {
	gases := map[string]planet.Gas{"neon": planet.Neon()}
	made := pressureBoundaryPlanet(t, "    cooking: {column: collapsed, time: 10}\n", false)
	if _, err := columnOf(made.Column("clear"), gases); err != nil {
		t.Fatal(err)
	}
	_, err := lit(planet.Body{}, made, gases)
	requireLayerCause(t, err)
	made = pressureBoundaryPlanet(t, boundaryStorm+"      churn: {column: collapsed, strength: 1}\n", false)
	_, err = stormed(planet.Body{}, made, gases)
	requireLayerCause(t, err)
	for _, marks := range []string{"{cap: collapsed, clearing: clear}", "{cap: clear, clearing: collapsed}"} {
		made = pressureBoundaryPlanet(t, boundaryStorm+"      marks: "+marks+"\n", false)
		written, has := made.Stormy()
		if !has {
			t.Fatal("fixture has no storms")
		}
		_, err = marked(planet.Body{}, made, written, gases)
		requireLayerCause(t, err)
	}
}

func TestStormConstructionRefusesMissingSpotRadiusAndReversedSwarm(t *testing.T) {
	for _, test := range []struct{ extra, want string }{
		{"      spots: [{}]\n", "a spot at"},
		{"      swarm: {seed: 5, count: 1, smallest: 2, largest: 1, slope: 2}\n", "a swarm of"},
	} {
		made := pressureBoundaryPlanet(t, boundaryStorm+test.extra, false)
		if _, err := stormed(planet.Body{}, made, map[string]planet.Gas{"neon": planet.Neon()}); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("storm lost model-domain refusal: %v", err)
		}
	}
}
