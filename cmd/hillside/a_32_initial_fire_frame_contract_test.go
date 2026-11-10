package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/flame"
)

func TestStagedSingletonFrontIncludesItsRequestedInitialTime(t *testing.T) {
	given := frontOnFixture(t)
	given.at = 3 * time.Second
	world, made, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if ready.live == nil || len(ready.live.fronts) != 1 || len(ready.live.beds) != 0 {
		t.Fatal("fixture lost its single front without a fuel bed")
	}
	defer ready.live.closeBeds()
	patches := 0
	for _, volume := range world.Volumes {
		if field, ok := volume.Medium.(*flame.Field); ok {
			patches += len(field.Tick(ready.at).(*flame.Field).Pieces())
		}
	}
	if patches != 4 || made.burning != 5 {
		t.Fatalf("initial three-second frame omitted its live front: %d patches, %d burning", patches, made.burning)
	}
}

func TestStagedSingletonFuelBedIncludesItsInitialSurfacesAndFlames(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "renderer: gpu-native", "renderer: cpu", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	world, made, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if ready.live == nil || len(ready.live.beds) != 1 || len(ready.live.fronts) != 0 {
		t.Fatal("fixture lost its single actual fuel bed without a front")
	}
	defer ready.live.closeBeds()
	surfaces, patches := 0, 0
	for _, item := range world.Things {
		if item.Material != nil && item.Material.Name == "surface fuel" {
			surfaces++
		}
	}
	for _, volume := range world.Volumes {
		if field, ok := volume.Medium.(*flame.Field); ok {
			patches += len(field.Pieces())
		}
	}
	if surfaces != 64 || patches == 0 || made.burning != patches+1 {
		t.Fatalf("initial frame omitted actual fuel: %d surfaces, %d flame patches, %d burning", surfaces, patches, made.burning)
	}
}
