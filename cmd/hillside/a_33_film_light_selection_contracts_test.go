package main

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestFilmKeepsMissingAndNarrowExplicitFieldOfViewDistinct(t *testing.T) {
	for _, check := range []struct {
		name, option string
		want         float64
	}{
		{"missing", "", 55}, {"narrow", "0.01", .01},
		{"one-degree", "1", 1}, {"above-one", "1.0000000000000002", 1.0000000000000002},
		{"wide", "179", 179},
	} {
		t.Run(check.name, func(t *testing.T) {
			given := mustParse(t)
			was, now := "", ""
			if check.option != "" {
				was, now = "    width: 24", "    wide: "+check.option+"\n    width: 24"
			}
			given.set = onFixture(t, was, now)
			read := readBoundarySet(t, given.set)
			got, err := filmed(read, given)
			if err != nil || got.wide != check.want {
				t.Fatalf("actual %s field of view =%g, %v; want %g degrees", check.name, got.wide, err, check.want)
			}
		})
	}
}

// The selected lighting mode remains authoritative. Sun coordinates may be
// present for another mode; absent coordinates retain the legacy daylight.
func TestSceneLightingKeepsModeAndOptionalSunCoordinatesSeparate(t *testing.T) {
	for _, check := range []struct {
		name, mode, sun string
		primary         float64
	}{
		{"legacy-day", "sun", "", 1.5},
		{"night-with-day-sun-metadata", "moon", "{azimuth: 180, altitude: 35}", .03},
		{"night-with-set-sun-metadata", "moon", "{azimuth: 180, altitude: -3}", .03},
	} {
		t.Run(check.name, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "", "")
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(raw), "light: moon", "light: "+check.mode, 1)
			text = strings.Replace(text, "alight: true", "alight: false", 1)
			if check.sun != "" {
				text = strings.Replace(text, "  terrain:", "  sun: "+check.sun+"\n  terrain:", 1)
			}
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			world, _, ready, err := staged(given)
			if err != nil {
				t.Fatalf("valid selected lighting was refused: %v", err)
			}
			defer ready.live.closeBeds()
			if len(world.Lamps) != 2 {
				t.Fatal("legacy light selection lost its primary or fill")
			}
			way, distance, light := world.Lamps[0].Light(maths.Vec{}, maths.YAxis)
			if !(way.Sub(maths.V(-.5, .8, .45).Unit()).Len() <= 1e-12 &&
				math.Abs(distance-4000) <= 1e-9 &&
				math.Abs(light.R-check.primary) <= 1e-12 &&
				math.Abs(light.G-check.primary) <= 1e-12 &&
				math.Abs(light.B-check.primary) <= 1e-12) {
				t.Fatalf("selected %s was replaced by a different sun: %v, %g, %v", check.mode, way, distance, light)
			}
		})
	}
}

func TestExplicitDaylightKeepsLowSunAcceptanceAndItsChosenFill(t *testing.T) {
	for _, check := range []struct {
		name, written string
		altitude      float64
		refused       bool
	}{
		{"horizon", "0", 0, true}, {"low", "0.1", .1, false},
		{"one-degree", "1", 1, false}, {"high", "35", 35, false},
	} {
		t.Run(check.name, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "light: moon", "light: sun\n  sun: {azimuth: 90, altitude: "+check.written+"}")
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(raw), "alight: true", "alight: false", 1)
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			world, made, ready, err := staged(given)
			if check.refused {
				if err == nil || !strings.Contains(err.Error(), "daylight sun must be above the horizon") {
					t.Fatalf("actual horizon input lost its daylight refusal: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid low sun was refused: %v", err)
			}
			defer ready.live.closeBeds()
			if len(world.Lamps) != 2 || made.lamps != 2 {
				t.Fatal("explicit day lost its two actual distant sources")
			}
			radians := check.altitude * math.Pi / 180
			wantWays := []maths.Vec{
				maths.V(math.Cos(radians), math.Sin(radians), 0),
				maths.V(.6, .5, -.3).Unit(),
			}
			for i, power := range []float64{1.5, .375} {
				way, distance, light := world.Lamps[i].Light(maths.Vec{}, maths.YAxis)
				if !(way.Sub(wantWays[i]).Len() <= 1e-12 && math.IsInf(distance, 1) &&
					math.Abs(light.R-power) <= 1e-12 && math.Abs(light.G-power) <= 1e-12 &&
					math.Abs(light.B-power) <= 1e-12) {
					t.Fatalf("explicit sun source %d changed its actual direction or chosen power: %v, %g, %v", i, way, distance, light)
				}
			}
		})
	}
}
