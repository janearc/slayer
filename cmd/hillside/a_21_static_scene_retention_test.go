package main

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// Combine real shared mist and moonlight with a real changing fire. A
// sequence must preserve static scene contributions while rebuilding fire.
func TestRetainedFramesKeepActualStaticMistAndMoonlight(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	weather := `  weather:
    mist:
      kind: prescribed-mist
      extinction: {value: 0.006, unit: m^-1}
      depth: {value: 20, unit: m}
      coverage: 1
      seed: 23
  terrain:`
	text := strings.Replace(string(raw), "  terrain:", weather, 1)
	text = strings.Replace(text, "  light: none", "  light: moon\n  moon:\n    azimuth: 90\n    altitude: 10", 1)
	text = strings.Replace(text, "renderer: gpu-native", "renderer: cpu", 1)
	text = strings.Replace(text, "        count: 12", "        count: 12\n        atmosphere: {weather: mist, extent: canopy-and-ground, fade-at-boundary: true}", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	initial, _, prepared, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.live == nil || len(prepared.live.fires) != 1 {
		t.Fatal("checked scene did not retain its actual fire")
	}
	defer prepared.live.closeBeds()
	var mist *thing.Volume
	for _, volume := range initial.Volumes {
		if _, ok := volume.Medium.(*medium.Grove); ok {
			if mist != nil {
				t.Fatal("fixture did not stage one shared static mist")
			}
			mist = volume
		}
	}
	if mist == nil {
		t.Fatal("initial scene lost its explicit static mist control")
	}
	staticLamps := func(world *thing.World) []thing.Lamp {
		var kept []thing.Lamp
		for _, lamp := range world.Lamps {
			if _, dynamic := lamp.(emit.Volume); !dynamic {
				kept = append(kept, lamp)
			}
		}
		return kept
	}
	moon := staticLamps(initial)
	if len(moon) != 1 || len(initial.Lamps) <= len(moon) {
		t.Fatal("initial scene needs real moonlight and real dynamic fire lamps")
	}
	probe := maths.V(10, 2, 5)
	wantDirection, wantDistance, wantLight := moon[0].Light(probe, maths.YAxis)
	if !(math.IsInf(wantDistance, 1) && wantLight.Luminance() > 0) {
		t.Fatal("moonlight control illuminates no actual point")
	}
	for _, at := range []time.Duration{0, time.Second, 2 * time.Second} {
		world, _, err := prepared.live.frameAt(at)
		if err != nil {
			t.Fatal(err)
		}
		if len(world.Volumes) != len(initial.Volumes) {
			t.Fatalf("at %s: %d media, want the original %d static and dynamic media", at, len(world.Volumes), len(initial.Volumes))
		}
		found := 0
		for _, volume := range world.Volumes {
			if grove, ok := volume.Medium.(*medium.Grove); ok {
				found++
				if grove.Bounds() != mist.Medium.Bounds() || volume.Material == nil ||
					volume.Material.Name != mist.Material.Name || volume.Material.Colour != mist.Material.Colour {
					t.Fatalf("at %s: static mist changed while retaining fire", at)
				}
			}
		}
		if found != 1 {
			t.Fatalf("at %s: retained %d static mist fields, want one", at, found)
		}
		lamps := staticLamps(world)
		if len(lamps) != len(moon) {
			t.Fatalf("at %s: lost actual static moonlight", at)
		}
		direction, distance, light := lamps[0].Light(probe, maths.YAxis)
		if gap := direction.Sub(wantDirection).Len(); !(gap <= 1e-12) {
			t.Fatalf("at %s: moonlight direction moved %g", at, gap)
		}
		if !(math.IsInf(distance, 1)) {
			t.Fatalf("at %s: moonlight distance changed", at)
		}
		for _, pair := range [][2]float64{{light.R, wantLight.R}, {light.G, wantLight.G}, {light.B, wantLight.B}} {
			if !(math.Abs(pair[0]-pair[1]) <= 1e-12*math.Max(1e-30, math.Abs(pair[1]))) {
				t.Fatalf("at %s: static moonlight strength changed", at)
			}
		}
	}
}

// These real legacy kinds sample their lamps during construction. Their
// refusal must come from retention, with generic alight disabled so that
// a different failure cannot conceal the lost unsupported-lighting cause.
func TestRetentionRefusesActualLegacyFireLampKinds(t *testing.T) {
	for _, kind := range []string{"spruce", "campfire", "bonfire"} {
		t.Run(kind, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "kind: spruce", "kind: "+kind)
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(raw), "alight: true", "alight: false", 1)
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			_, _, prepared, err := staged(given)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.live == nil || prepared.live.made.burning != 0 || len(prepared.live.fires) != 0 {
				t.Fatal("fixture accidentally activated the separate legacy-alight refusal")
			}
			defer prepared.live.closeBeds()
			_, _, err = prepared.live.frameAt(time.Second)
			if kind == "spruce" {
				if err != nil {
					t.Fatalf("ordinary static scene should retain its frame: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(),
				"legacy fire lamps are sampled at construction; use fire recipes") {
				t.Fatalf("actual %s lost its retained-lighting refusal: %v", kind, err)
			}
		})
	}
}
