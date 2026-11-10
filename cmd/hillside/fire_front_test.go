package main

import (
	"github.com/janearc/slayer/flame"
	"os"
	"strings"
	"testing"
	"time"
)

func frontOnFixture(t *testing.T) asked {
	t.Helper()
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	text = strings.Replace(text, "    stands:\n", "    stands:\n      front fuel: {kind: fern, count: 4, seed: 17}\n", 1)
	text = strings.Replace(text, "  film:", `  fire-fronts:
    a passing front:
      stand: front fuel
      recipe: wood fire
      from: the tree
      bearing: {value: 90, unit: deg}
      speed: {value: 1000, unit: m/s}
      starts: {value: 2, unit: s}
      burn-duration: {value: 2, unit: s}
      fade-time: {value: 0.2, unit: s}
      group-width: {value: 100, unit: m}
      ignition-height: {value: 0, unit: m}
      smoke-radius: {value: 8, unit: m}
  film:`, 1)
	if !strings.Contains(text, "front fuel:") {
		t.Fatal("test fixture has no stand section")
	}
	if err = os.WriteFile(given.set, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	return given
}

func TestLiveFrontIgnitesPlacedFuelAndRetainsItsTimePlan(t *testing.T) {
	given := frontOnFixture(t)
	_, _, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	before, base, err := ready.live.frameAt(0)
	if err != nil {
		t.Fatal(err)
	}
	during, made, err := ready.live.frameAt(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	after, _, err := ready.live.frameAt(8 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	replay, _, err := ready.live.frameAt(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	patches := 0
	for _, volume := range during.Volumes {
		if field, ok := volume.Medium.(*flame.Field); ok {
			patches += len(field.Tick(3 * time.Second).(*flame.Field).Pieces())
		}
	}
	if patches != 4 {
		t.Fatalf("front ignited %d targets, want 4", patches)
	}
	if len(before.Volumes) != 2 || len(after.Volumes) != 2 || len(during.Volumes) <= 2 || len(replay.Volumes) != len(during.Volumes) {
		t.Fatal("front volumes did not follow ignition windows")
	}
	if made.burning != base.burning+4 || len(during.Lamps) <= len(before.Lamps) {
		t.Fatal("front lost burning count or source light")
	}
	if len(during.Things) != len(before.Things) || during.Things[0] != before.Things[0] {
		t.Fatal("front rebuilt or removed fuel geometry")
	}
}

func TestLiveFrontRefusesTooManyConcurrentGroupsBeforeRendering(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "kind: fern, count: 4", "kind: fern, count: 100", 1)
	text = strings.Replace(text, "group-width: {value: 100", "group-width: {value: 0.01", 1)
	if err = os.WriteFile(given.set, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = staged(given); err == nil || !strings.Contains(err.Error(), "volume") {
		t.Fatalf("excessive front accepted: %v", err)
	}
}
