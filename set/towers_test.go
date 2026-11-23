package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// towerSet is a set with a tower recipe, whose recipe block is given.
func towerSet(recipe string) string {
	return `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: tower recipe
  setting: space
  light: none
  film: {width: 64, height: 48, rays: 1, wide: 40, exposure: 1}
  cameras:
    far:
      location: {at: [0, -300, 2]}
      orientation: {look_at: [0, 0, 60]}
  recipes:
    landmark:
      kind: tower
` + recipe + `  set:
    things:
      the tower:
        recipe: landmark
        seed: 1
        complexity: hull
        east: 0
        north: 0
        height: {value: 0, unit: m}
  world:
    unit: 1
`
}

func TestATowerRecipeReadsItsIrregularityAndPinsAndConstructs(t *testing.T) {
	source := towerSet(`      irregularity: 0.8
      pins:
        plan: "y"
        shafts: 1
        setbacks: true
        span: none
`)
	made, err := set.Read("tower.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := made.Recipe("landmark")
	if !found || recipe.Kind() != "tower" {
		t.Fatalf("recipe %+v", recipe)
	}
	if _, err := recipe.Construction(); err != nil {
		t.Error(err)
	}
}

// bridgeSet is a set with a bridge recipe, whose recipe block is given.
func bridgeSet(recipe string) string {
	return strings.Replace(towerSet(recipe), "kind: tower", "kind: bridge",
		1)
}

func TestABridgeRecipeReadsItsSpanAndDepthAndConstructs(t *testing.T) {
	source := bridgeSet(`      irregularity: 0.6
      span: {value: 600, unit: m}
      depth: 1
`)
	made, err := set.Read("bridge.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := made.Recipe("landmark")
	if !found || recipe.Kind() != "bridge" {
		t.Fatalf("recipe %+v", recipe)
	}
	if _, err := recipe.Construction(); err != nil {
		t.Error(err)
	}
}

func TestABridgeRecipeThatCannotBeKeptIsRefused(t *testing.T) {
	for name, recipe := range map[string]string{
		"irregularity over one": "      irregularity: 1.5\n",
		"negative span":         "      span: {value: -5, unit: m}\n",
		"depth too deep":        "      depth: 40\n",
		"pins on a bridge":      "      pins: {plan: y}\n",
	} {
		source := bridgeSet(recipe)
		if _, err := set.Read("bridge.yaml", []byte(source)); err == nil {
			t.Errorf("%s: the recipe was accepted", name)
		}
	}
	deep := towerSet(
		"      depth: 1\n")
	if _, err := set.Read("tower.yaml", []byte(deep)); err == nil {
		t.Error("a tower was given a depth")
	}
}

func TestATowerRecipeMayDrawItsIrregularity(t *testing.T) {
	drawn := towerSet(
		"      irregularity: drawn\n")
	if _, err := set.Read("tower.yaml", []byte(drawn)); err != nil {
		t.Error(err)
	}
}

func TestATowerRecipeThatCannotBeKeptIsRefused(t *testing.T) {
	for name, recipe := range map[string]string{
		"irregularity over one":  "      irregularity: 1.5\n",
		"unknown plan":           "      pins: {plan: dome}\n",
		"pin that is not a bool": "      pins: {taper: sometimes}\n",
		"too many shafts":        "      pins: {shafts: 40}\n",
	} {
		source := towerSet(recipe)
		_, err := set.Read("tower.yaml", []byte(source))
		if err == nil {
			t.Errorf("%s: the recipe was accepted", name)
		}
	}
	other := strings.Replace(towerSet("      irregularity: 0.5\n"),
		"kind: tower", "kind: yurt", 1)
	if _, err := set.Read("tower.yaml", []byte(other)); err == nil {
		t.Error("a yurt was given an irregularity")
	}
}
