package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// crowdSource is the built fixture with a crowd of lamps in its world.
func crowdSource(t *testing.T, field string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/built.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "  world:\n", "  world:\n"+
		"    lamp-fields:\n      crowd:\n"+field, 1)
	if text == string(raw) {
		t.Fatal("built fixture has no world block")
	}
	return "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json\n" + text
}

const crowdField = "        count: 500\n        seed: 7\n" +
	"        low: [-10, 0.5, -10]\n        high: [10, 4, 10]\n" +
	"        temperature: 3000\n        strength: 2\n        radius: 0.05\n" +
	"        source: eye\n"

func TestLampFieldScattersCountLampsInItsBoxTheSameEachTime(t *testing.T) {
	read := func() []set.Lamp {
		source := crowdSource(t, crowdField)
		made, err := set.Read("crowd.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		world, _ := made.World()
		var crowd []set.Lamp
		for _, lamp := range world.LampFields() {
			if strings.HasPrefix(lamp.Name(), "crowd-") {
				crowd = append(crowd, lamp)
			}
		}
		return crowd
	}
	lamps := read()
	if len(lamps) != 500 {
		t.Fatalf("%d lamps, want 500", len(lamps))
	}
	if lamps[0].Name() != "crowd-0" || lamps[499].Name() != "crowd-499" {
		t.Errorf("names %q .. %q", lamps[0].Name(), lamps[499].Name())
	}
	again := read()
	apart := false
	for index, lamp := range lamps {
		at := lamp.At()
		// the set's world flips y and z; the box is in the file's
		// terms, checked by the extent of the three axes together
		if at != again[index].At() {
			t.Fatalf("lamp %d moved between reads", index)
		}
		if index > 0 && at != lamps[0].At() {
			apart = true
		}
		heat, strength := lamp.Shine()
		if heat != 3000 || strength != 2 {
			t.Fatalf("lamp %d shines %v %v", index, heat, strength)
		}
	}
	if !apart {
		t.Error("every lamp is in one place")
	}
}

// anti: a different seed is a different crowd, and a field that leaves a
// word out, or whose high is below its low, is refused by name.
func TestLampFieldSeedsAndRefusals(t *testing.T) {
	aSource := crowdSource(t, crowdField)
	a, err := set.Read("a.yaml", []byte(aSource))
	if err != nil {
		t.Fatal(err)
	}
	bSource := crowdSource(t,
		strings.Replace(crowdField, "seed: 7", "seed: 8", 1))
	b, err := set.Read("b.yaml", []byte(bSource))
	if err != nil {
		t.Fatal(err)
	}
	wa, _ := a.World()
	wb, _ := b.World()
	if wa.LampFields()[0].At() == wb.LampFields()[0].At() {
		t.Error("two seeds drew the same first lamp")
	}
	for name, bad := range map[string]string{
		"no seed":    strings.Replace(crowdField, "        seed: 7\n", "", 1),
		"high below": strings.Replace(crowdField, "high: [10, 4, 10]", "high: [-20, 4, 10]", 1),
		"no count":   strings.Replace(crowdField, "        count: 500\n", "", 1),
		"half count": strings.Replace(crowdField, "count: 500", "count: 2.5", 1),
	} {
		source := crowdSource(t, bad)
		if _, err := set.Read("bad.yaml", []byte(source)); err == nil {
			t.Errorf("%s was read", name)
		}
	}
}
