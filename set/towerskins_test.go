package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestATowerRecipeReadsItsSkinAndLayersAndConstructs(t *testing.T) {
	source := towerSet(`      irregularity: 0
      pins:
        plan: lobed
        skin: granite
        style: ribbon
        colour: red
        use: home
        cyber: true
        planted: false
        spiral: false
        storeys: 40
        segments: 3
        width: {value: 30, unit: m}
`)
	made, err := set.Read("skins.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := made.Recipe("landmark")
	if !found {
		t.Fatal("no recipe")
	}
	if _, err := recipe.Construction(); err != nil {
		t.Error(err)
	}
}

func TestATowerRecipeRefusesASkinOrAUseOrAStyleThereIsNot(t *testing.T) {
	for name, pin := range map[string]string{"skin": "skin: chrome",
		"use": "use: factory", "style": "style: arched",
		"storeys": "storeys: 0", "cyber": "cyber: maybe"} {
		source := towerSet(
			"      pins:\n        " + pin + "\n")
		_, err := set.Read("skins.yaml", []byte(source))
		if err == nil || !strings.Contains(err.Error(), strings.Split(pin,
			":")[0]) {
			t.Errorf("%s: %q was read: %v", name, pin, err)
		}
	}
}

func TestATowerRecipeWithAColourThatIsNotOneOfTheSkinsIsRefusedAtReading(
	t *testing.T) {
	source := towerSet(`      pins:
        skin: mirror
        colour: chartreuse
`)
	_, err := set.Read("skins.yaml", []byte(source))
	if err == nil || !strings.Contains(err.Error(), "chartreuse") {
		t.Errorf("a colour no mirror has was accepted: %v", err)
	}
}
