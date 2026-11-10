package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestPlanetOptionalModelsRefuseIncompleteAndInconsistentInputs(t *testing.T) {
	storm := "{seed: 1, turbulence: .2, eddies: 3, stretch: 2, edge: 4"
	weather := "{cells: 32, day: 1000, drag: 100, relax: 100, " +
		"storms: {rate: .001, wide: 2, strength: .1}, seed: 1, moment: 0, every: 1}"
	for _, test := range []struct{ block, want string }{
		{"xrays: {times: -1, of: jupiter, band: 3}", "takes a number"},
		{"xrays: {times: 1, of: jupiter}", "x-rays say"},
		{"sky: {azimuth: -1, altitude: 4, distance: 100}", "takes a number"},
		{"sky: {azimuth: 1, altitude: 4}", "sky says"},
		{"storms: {seed: -1}", "takes a number"},
		{"storms: {seed: 1}", "storms say"},
		{"storms: " + storm + ", churn: {column: absent, strength: .1}}", "churn column"},
		{"storms: " + storm + ", jets: {shear: -1, every: 1}}", "takes a number"},
		{"storms: " + storm + ", jets: {shear: 1}}", "jets say"},
		{"storms: " + storm + ", swarm: {seed: -1}}", "takes a number"},
		{"storms: " + storm + ", swarm: {seed: 1}}", "a swarm says"},
		{"storms: " + storm + ", spots: {latitude: 0}}", "spots is a list"},
		{"storms: " + storm + ", spots: [{latitude: 91}]}", "takes a number"},
		{"heat: {tops: -1, hottest: 500}", "takes a number"},
		{"heat: {tops: 300}", "heat says"},
		{"heat: {tops: 500, hottest: 300}", "no cooler than"},
		{"cooking: {column: clear, time: 0}", "takes a number"},
		{"cooking: {column: clear}", "cooking says"},
		{"airglow: {radiance: -1, band: 1}", "takes a number"},
		{"airglow: {radiance: 1}", "airglow says"},
		{"weather: " + strings.Replace(weather, "cells: 32", "cells: 0", 1), "takes a number"},
		{"weather: " + strings.Replace(weather, "every: 1", "", 1), "weather says"},
		{"weather: " + strings.Replace(weather, "cells: 32", "cells: 31", 1), "power of two"},
	} {
		t.Run(test.block, func(t *testing.T) {
			raw := strings.Replace(giant, "    from_star:", "    "+test.block+"\n    from_star:", 1)
			_, err := set.Read("planet-contract.yaml", []byte(raw))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("want %q, got %v", test.want, err)
			}
		})
	}
}

func TestRingSubmodelsRefuseMissingRolesAndReversedSizes(t *testing.T) {
	for _, test := range []struct{ block, want string }{
		{"moonlets: {seed: -1}", "takes a number"},
		{"moonlets: {seed: 1}", "moonlets say"},
		{"moonlets: {seed: 1, count: 4, smallest: 10, largest: 2, slope: 2, opens: 1}", "largest is below"},
		{"grains: {seed: -1}", "takes a number"},
		{"grains: {seed: 1}", "grains say"},
		{"grains: {pattern: cells, seed: 1, smallest: 10, largest: 2, slope: 2, shear: 1, clumping: .2}", "largest is below"},
	} {
		raw := strings.Replace(ringed(), "made_of: {ice: 1}", "made_of: {ice: 1}, "+test.block, 1)
		_, err := set.Read("ring-contract.yaml", []byte(raw))
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: want %q, got %v", test.block, test.want, err)
		}
	}
	if shear, every, has := (set.Storms{}).Jets(); has || shear != 0 || every != 0 {
		t.Fatal("unspecified jets acquired values")
	}
	if seed, count, small, large, slope, has := (set.Storms{}).Swarm(); has || seed != 0 || count != 0 || small != 0 || large != 0 || slope != 0 {
		t.Fatal("unspecified swarm acquired values")
	}
}
