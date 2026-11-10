package set_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// The legacy reader also serves files without the opt-in JSON schema.
// Hold its own numeric refusals and locations rather than accepting a
// refusal from a later constructor or from schema validation.
func legacyNumericFixture(raw string) string {
	return strings.Replace(raw, "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v1.json\n", "", 1)
}

func requireLegacyNumericControl(t *testing.T, raw string) {
	t.Helper()
	if _, err := set.Read("numeric-control.yaml", []byte(raw)); err != nil {
		t.Fatalf("unmodified legacy control: %v", err)
	}
}

type numericContract struct {
	prefix, old, key string
	low, high        float64
}

func refuseNumericEdge(t *testing.T, raw string, row numericContract, value float64) {
	t.Helper()
	before := row.prefix + row.old
	amount := fmt.Sprintf("%.17g", value)
	after := row.prefix + amount
	if strings.Count(raw, before) != 1 {
		t.Fatalf("ambiguous numeric fixture %q", before)
	}
	index := strings.Index(raw, before) + len(row.prefix)
	edited := strings.Replace(raw, before, after, 1)
	_, err := set.Read("numeric-contract.yaml", []byte(edited))
	line := 1 + strings.Count(edited[:index], "\n")
	column := index - strings.LastIndex(edited[:index], "\n")
	want := fmt.Sprintf("numeric-contract.yaml:%d:%d: %s takes a number from %v to %v, not %q",
		line, column, row.key, row.low, row.high, amount)
	if err == nil || err.Error() != want {
		t.Fatalf("%s=%s: %v, want %s", row.prefix, amount, err, want)
	}
}

func TestLegacyFireNumericLimitsReportTheirOwnExactLocations(t *testing.T) {
	raw := legacyNumericFixture(fireRecipeFixture(t))
	requireLegacyNumericControl(t, raw)
	rows := []numericContract{
		{"radius: {value: ", "0.45", "value", .001, 10000},
		{"height: {value: ", "1.5", "value", .001, 10000},
		{"absorption: {value: ", "1", "value", 0, 10000},
		{"temperature: {value: ", "1750", "value", 1, 10000},
		{"edge-temperature: {value: ", "900", "value", 1, 10000},
		{"heat-release: {value: ", "40", "value", .001, 1e12},
		{"convective-fraction: ", "0.71", "convective-fraction", 1e-6, 1},
		{"soot-yield: ", "0.0024", "soot-yield", 1e-12, 1},
		{"combustion: {value: ", "21", "value", .001, 1e6},
		{"mass-extinction: {value: ", "10.6", "value", 0, 1e6},
		{"air-temperature: {value: ", "288", "value", 1, 10000},
		{"air-density: {value: ", "1.2", "value", 1e-9, 10000},
		{"height: {value: ", "10", "value", .001, 1e6},
		{"albedo: ", "0.3", "albedo", 0, 1},
		{"light-cells: ", "3", "light-cells", 3, 8},
	}
	for _, row := range rows {
		t.Run(row.prefix+row.old, func(t *testing.T) {
			refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
			refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
		})
	}
}

func TestLegacyFrontNumericLimitsReportTheirOwnExactLocations(t *testing.T) {
	raw := legacyNumericFixture(frontFixture(t))
	requireLegacyNumericControl(t, raw)
	rows := []numericContract{
		{"bearing: {value: ", "90", "value", 0, 360},
		{"speed: {value: ", "2", "value", .000001, 1000},
		{"starts: {value: ", "3", "value", 0, 86400},
		{"burn-duration: {value: ", "8", "value", .001, 86400},
		{"fade-time: {value: ", "0.2", "value", 0, 43200},
		{"group-width: {value: ", "12", "value", .01, 10000},
		{"ignition-height: {value: ", "1", "value", 0, 10000},
		{"smoke-radius: {value: ", "8", "value", .001, 10000},
	}
	for _, row := range rows {
		t.Run(row.prefix, func(t *testing.T) {
			refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
			refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
		})
	}
}

func TestLegacySparkNumericLimitsReportTheirOwnExactLocations(t *testing.T) {
	raw := legacyNumericFixture(strings.Replace(fireRecipeFixture(t), "      flame:", sparkBlock+"      flame:", 1))
	requireLegacyNumericControl(t, raw)
	rows := []numericContract{
		{"rate: {value: ", "100", "value", .001, 1e6},
		{"lifetime: {value: ", "3", "value", .001, 3600},
		{"radius: {value: ", "0.025", "value", .0001, 1},
		{"release-height: {value: ", "3", "value", 0, 10000},
		{"launch-speed: {value: ", "9", "value", 0, 1000},
		{"spread: {value: ", "2", "value", 0, 1000},
		{"updraft: {value: ", "8", "value", 0, 1000},
		{"drag-time: {value: ", "0.8", "value", .001, 1000},
		{"gravity: {value: ", "9.80665", "value", 0, 100},
		{"temperature: {value: ", "1650", "value", 1, 10000},
		{"end-temperature: {value: ", "900", "value", 1, 10000},
	}
	for _, row := range rows {
		t.Run(row.prefix, func(t *testing.T) {
			refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
			refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
		})
	}
}

func TestLegacyFuelNumericLimitsReportTheirOwnExactLocations(t *testing.T) {
	raw := legacyNumericFixture(bedFixture(t))
	requireLegacyNumericControl(t, raw)
	rows := []numericContract{
		{"east: ", "263812", "east", -1e8, 1e8},
		{"north: ", "2148748", "north", -1e8, 1e8},
		{"width: ", "32", "width", 8, 32},
		{"depth: ", "32", "depth", 8, 32},
		{"cell: {value: ", "0.5", "value", .25, 2},
		{"step: {value: ", "0.01", "value", .001, .1},
		{"ignition-radius: {value: ", "1.6", "value", .01, 100},
		{"ignition-temperature: {value: ", "1200", "value", 300, 3000},
		{"ambient-temperature: {value: ", "300", "value", 200, 400},
		{"fuel: ", "1", "fuel", 0, 1},
	}
	for _, row := range rows {
		t.Run(row.prefix, func(t *testing.T) {
			refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
			refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
		})
	}
}

func TestLegacyObservatoryAndWeatherLimitsReportTheirOwnExactLocations(t *testing.T) {
	raw := legacyNumericFixture(ecosystemFixture(t))
	requireLegacyNumericControl(t, raw)
	rows := []numericContract{
		{"diameter: {value: ", "24", "value", 1, 200},
		{"opening: {value: ", "35", "value", 5, 90},
		{"aperture: {value: ", "8", "value", .1, 120},
		{"azimuth: ", "180", "azimuth", 0, 360},
		{"altitude: ", "55", "altitude", 0, 90},
		{"from: {value: ", "270", "value", 0, 360},
		{"speed: {value: ", "2", "value", 0, 100},
		{"extinction: {value: ", "0.006", "value", 0, 100},
		{"depth: {value: ", "20", "value", .1, 1000},
		{"coverage: ", "0.4", "coverage", 0, 1},
		{"seed: ", "23", "seed", 0, 9007199254740992},
		{"clearance: {value: ", "40", "value", 0, 100000},
		{"count: ", "5000", "count", 1, 100000},
		{"above: ", "1150", "above", -10000, 10000},
		{"lon: ", "-155.2485", "lon", -180, 180},
	}
	for _, row := range rows {
		t.Run(row.prefix+row.old, func(t *testing.T) {
			refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
			refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
		})
	}
}

func TestLegacyLayoutNumericLimitsReportTheirOwnExactLocations(t *testing.T) {
	for _, pattern := range []string{"cluster", "line"} {
		block := fireLayoutBlock
		if pattern == "line" {
			block = strings.Replace(block, "pattern: cluster", "pattern: line", 1)
			block = strings.Replace(block, "radius: {value: 0.55", "length: {value: 4", 1)
		}
		raw := legacyNumericFixture(strings.Replace(fireRecipeFixture(t), "      flame:", block+"      flame:", 1))
		requireLegacyNumericControl(t, raw)
		rows := []numericContract{
			{"count: ", "7", "count", 2, 32},
			{"height-variation: ", "0.3", "height-variation", 0, .5},
		}
		if pattern == "cluster" {
			rows = append(rows, numericContract{"radius: {value: ", "0.55", "value", .001, 10000})
		}
		if pattern == "line" {
			rows = append(rows, numericContract{"length: {value: ", "4", "value", .001, 10000})
		}
		for _, row := range rows {
			t.Run(pattern+row.prefix, func(t *testing.T) {
				refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
				refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
			})
		}
	}
}

func TestLegacyMoonAndPlacementLimitsReportTheirOwnExactLocations(t *testing.T) {
	raw := legacyNumericFixture(fireRecipeFixture(t))
	raw = strings.Replace(raw, "  light: none", "  light: moon\n  moon: {azimuth: 123, altitude: 21}", 1)
	rows := []numericContract{
		{"azimuth: ", "123", "azimuth", 0, 360},
		{"altitude: ", "21", "altitude", -90, 90},
		{"  seed: ", "7", "seed", 0, 9007199254740992},
		{"        seed: ", "5", "seed", 0, 9007199254740992},
		{"        east: ", "263812", "east", 0, math.Inf(1)},
		{"        north: ", "2148748", "north", 0, math.Inf(1)},
	}
	requireLegacyNumericControl(t, raw)
	for _, row := range rows {
		t.Run(row.prefix, func(t *testing.T) {
			refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
			if !math.IsInf(row.high, 0) {
				refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
			}
		})
	}
}

func TestLegacyStandLimitsReportTheirOwnExactLocations(t *testing.T) {
	raw := legacyNumericFixture(ecosystemFixture(t))
	raw = strings.Replace(raw, "        above: 1150", "        above: 1150\n        below: 2000\n        alight: 0", 1)
	rows := []numericContract{
		{"        seed: ", "11", "seed", 0, 9007199254740992},
		{"        below: ", "2000", "below", -10000, 10000},
		{"        alight: ", "0", "alight", 0, 100000},
	}
	requireLegacyNumericControl(t, raw)
	for _, row := range rows {
		t.Run(row.prefix, func(t *testing.T) {
			refuseNumericEdge(t, raw, row, math.Nextafter(row.low, math.Inf(-1)))
			refuseNumericEdge(t, raw, row, math.Nextafter(row.high, math.Inf(1)))
		})
	}
}
