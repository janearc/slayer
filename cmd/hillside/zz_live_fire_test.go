package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLiveFireFramesReuseTheStaticSceneryAndDoNotAccumulate(t *testing.T) {
	given := fireOnFixture(t)
	_, _, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if ready.live == nil {
		t.Fatal("staging did not retain its animation state")
	}
	a, _, err := ready.live.frameAt(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := ready.live.frameAt(4 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	again, _, err := ready.live.frameAt(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Things) == 0 || len(a.Things) != len(b.Things) || len(a.Things) != len(again.Things) ||
		a.Things[0] != b.Things[0] || a.Things[0] != again.Things[0] {
		t.Fatal("static geometry was rebuilt or accumulated")
	}
	if len(a.Volumes) != 2 || len(b.Volumes) != 2 || len(again.Volumes) != 2 {
		t.Fatal("fire volumes accumulated")
	}
	if a.Volumes[0] != b.Volumes[0] {
		t.Fatal("immutable fire field was rebuilt")
	}
	if len(a.Lamps) == 0 || len(again.Lamps) != len(a.Lamps) {
		t.Fatal("fire lights were lost")
	}
	for i := range a.Lamps {
		if !reflect.DeepEqual(a.Lamps[i], again.Lamps[i]) {
			t.Fatal("same time changed live fire light")
		}
	}
	if a == b || a == again {
		t.Fatal("frames share mutable world containers")
	}
}

func TestLiveFireFramesPreserveUnrelatedLightsAndCallerSlices(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "light: none", "light: moon\n  moon: {azimuth: 115, altitude: 28, disc: false}", 1)
	if err = os.WriteFile(given.set, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	a, _, err := ready.live.frameAt(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lights := len(a.Lamps)
	first := a.Things[0]
	a.Things[0] = nil
	a.Lamps = nil
	a.Volumes = nil
	b, _, err := ready.live.frameAt(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if b.Things[0] != first || len(b.Lamps) != lights || len(b.Volumes) != 2 {
		t.Fatal("caller changed the retained scene")
	}
	if len(b.Fields) != 0 {
		t.Fatal("unrequested fields appeared")
	}
}
