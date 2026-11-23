package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// streetSet is a set with a street recipe, whose recipe block is given.
func streetSet(recipe string) string {
	return strings.Replace(towerSet(recipe), "kind: tower", "kind: street",
		1)
}

func TestAStreetRecipeReadsItsWetnessAndLengthAndConstructs(t *testing.T) {
	source := streetSet(`      wetness: 0.8
      length: {value: 24, unit: m}
`)
	made, err := set.Read("street.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := made.Recipe("landmark")
	if !found || recipe.Kind() != "street" {
		t.Fatalf("recipe %+v", recipe)
	}
	if _, err := recipe.Construction(); err != nil {
		t.Error(err)
	}
}

func TestAStreetRecipeThatCannotBeKeptIsRefused(t *testing.T) {
	for name, recipe := range map[string]string{
		"wetness over one":  "      wetness: 1.5\n",
		"negative wetness":  "      wetness: -0.1\n",
		"length too short":  "      length: {value: 2, unit: m}\n",
		"length too long":   "      length: {value: 400, unit: m}\n",
		"length without it": "      length: 24\n",
		"pins on a street":  "      pins: {plan: y}\n",
		"depth on a street": "      depth: 1\n",
	} {
		source := streetSet(recipe)
		if _, err := set.Read("street.yaml", []byte(source)); err == nil {
			t.Errorf("%s: the recipe was accepted", name)
		}
	}
}

func TestOnlyAStreetIsWetOrLong(t *testing.T) {
	for _, recipe := range []string{"      wetness: 0.5\n",
		"      length: {value: 24, unit: m}\n"} {
		source := bridgeSet(recipe)
		if _, err := set.Read("bridge.yaml", []byte(source)); err == nil {
			t.Errorf("a bridge was given %q", recipe)
		}
	}
}
