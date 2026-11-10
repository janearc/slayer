package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestPlanetOptionalModelsReachTheirPublicViews(t *testing.T) {
	extra := `    weather: {cells: 32, day: 1000, drag: 100, relax: 200, storms: {rate: .001, wide: 2, strength: .1}, seed: 7, moment: 300, every: 60}
    heat: {tops: 200, hottest: 800}
    cooking: {column: hazed, time: 100}
    airglow: {radiance: .2, band: 30}
    sky: {azimuth: 120, altitude: 40, distance: 1e9}
    xrays: {times: 2, of: jupiter, band: 20}
    ringlets: {seed: 5, strength: .4, width: 1000}
    storms:
      seed: 3
      turbulence: .2
      eddies: 10
      stretch: 2
      edge: 4
      folds: 2
      jets: {shear: 30, every: 15}
      swarm: {seed: 11, count: 9, smallest: 1, largest: 3, slope: 2}
      marks: {cap: hazed, clearing: clear}
      churn: {column: hazed, strength: .3}
      spots: [{latitude: -20, longitude: 10, radius: 3, turn: 1}]
`
	source := strings.Replace(giant, "    from_star:", extra+"    from_star:", 1)
	read, err := set.Read("models.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	planet, has := read.Planet()
	if !has {
		t.Fatal("planet absent")
	}
	weather, has := planet.Weather()
	if !has || weather.Grid() != 32 || weather.Seed() != 7 {
		t.Fatal("weather grid or seed")
	}
	if day, drag, relax := weather.Turning(); day != 1000 || drag != 100 || relax != 200 {
		t.Fatal("weather times")
	}
	if rate, wide, strength := weather.Storms(); rate != .001 || wide != 2 || strength != .1 {
		t.Fatal("storm units")
	}
	if moment, every := weather.Moment(); moment != 300 || every != 60 {
		t.Fatal("cache times")
	}
	if top, hottest, has := planet.Heat(); !has || top != 200 || hottest != 800 {
		t.Fatal("heat")
	}
	if column, seconds, has := planet.Cooking(); !has || column != "hazed" || seconds != 100 {
		t.Fatal("cooking")
	}
	if power, band, fronts, has := planet.Airglow(); !has || power != .2 || band != 30 || fronts != .2 {
		t.Fatal("airglow defaults")
	}
	if az, alt, away, has := planet.InSky(); !has || az != 120 || alt != 40 || away != 1e9 {
		t.Fatal("sky placement")
	}
	if watts, band, has := planet.Xrays(); !has || watts != 2e9 || band != 20 {
		t.Fatal("x-ray units")
	}
	if seed, strength, width, has := planet.Ringlets(); !has || seed != 5 || strength != .4 || width != 1000 {
		t.Fatal("ringlets")
	}
	storms, has := planet.Stormy()
	if !has {
		t.Fatal("storms absent")
	}
	if seed, turb, eddies, stretch, edge, folds := storms.Flow(); seed != 3 || turb != .2 || eddies != 10 || stretch != 2 || edge != 4 || folds != 2 {
		t.Fatal("storm flow")
	}
	if shear, every, has := storms.Jets(); !has || shear != 30 || every != 15 {
		t.Fatal("jets")
	}
	if seed, count, small, large, slope, has := storms.Swarm(); !has || seed != 11 || count != 9 || small != 1 || large != 3 || slope != 2 {
		t.Fatal("swarm")
	}
	if cap, clearing := storms.Marks(); cap != "hazed" || clearing != "clear" {
		t.Fatal("marks")
	}
	if column, strength := storms.Churn(); column != "hazed" || strength != .3 {
		t.Fatal("churn")
	}
	spots := storms.Spots()
	if len(spots) != 1 {
		t.Fatal("spot count")
	}
	if lat, lon, radius, turn := spots[0].Where(); lat != -20 || lon != 10 || radius != 3 || turn != 1 {
		t.Fatal("spot coordinates")
	}
	spots[0] = set.Spot{}
	if lat, _, _, _ := storms.Spots()[0].Where(); lat != -20 {
		t.Fatal("caller changed internal spots")
	}
}

func TestRingOptionalModelsKeepAllUnitsAndDefaults(t *testing.T) {
	models := "moonlets: {seed: 7, count: 9, smallest: 2, largest: 20, slope: 3, opens: 10}, grains: {pattern: cells, seed: 11, smallest: 1, largest: 50, slope: 2, shear: 3, clumping: .4}"
	source := strings.Replace(ringed(), "made_of: {ice: 1}", "made_of: {ice: 1}, "+models, 1)
	read, err := set.Read("ring-models.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := read.Planet()
	rings := body.Rings()
	var found bool
	for _, ring := range rings {
		if seed, count, small, large, slope, opens, has := ring.Moonlets(); has {
			found = true
			if seed != 7 || count != 9 || small != 2 || large != 20 || slope != 3 || opens != 10 {
				t.Fatal("moonlet units")
			}
			grains, has := ring.Grained()
			if !has {
				t.Fatal("grains absent")
			}
			if name, seed, small, large, slope, shear, clump := grains.Clumps(); name != "cells" || seed != 11 || small != 1 || large != 50 || slope != 2 || shear != 3 || clump != .4 {
				t.Fatal("grain units")
			}
		}
	}
	if !found {
		t.Fatal("ring models absent")
	}
	if _, has := (set.Ring{}).Grained(); has {
		t.Fatal("default ring has grains")
	}
	if _, _, _, _, _, _, has := (set.Ring{}).Moonlets(); has {
		t.Fatal("default ring has moonlets")
	}
	plain, err := set.Read("plain.yaml", []byte(giant))
	if err != nil {
		t.Fatal(err)
	}
	planet, _ := plain.Planet()
	if _, has := planet.Weather(); has {
		t.Fatal("default weather")
	}
	if _, _, has := planet.Heat(); has {
		t.Fatal("default heat")
	}
	if _, _, has := planet.Cooking(); has {
		t.Fatal("default cooking")
	}
	if _, _, _, has := planet.Airglow(); has {
		t.Fatal("default airglow")
	}
	if _, _, has := planet.Xrays(); has {
		t.Fatal("default x-rays")
	}
	if _, _, _, has := planet.Ringlets(); has {
		t.Fatal("default ringlets")
	}
	if _, _, _, has := planet.InSky(); has {
		t.Fatal("default sky")
	}
	if _, has := planet.Stormy(); has {
		t.Fatal("default storms")
	}
}
