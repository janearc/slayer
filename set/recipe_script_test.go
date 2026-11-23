package set_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// simplifiedStreet is the set that asks for the simplified script, and
// askedForScript the text of its mirror recipe that the tests change to see
// what is refused.
const (
	simplifiedStreet = "neon-street-simplified.yaml"
	askedForScript   = "script: simplified-chinese, carries: " +
		"[womens-shelter, dog-boutique], pins:"
	aStreetRecipe = "wet: {kind: street, "
)

// readSimplifiedStreet is the set that asks for the simplified script,
// with the first text it has of before changed to after.
func readSimplifiedStreet(t *testing.T, before, after string) (set.Set,
	error) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "sets", simplifiedStreet))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), before, after, 1)
	if before != after && text == string(raw) {
		t.Fatalf("%s no longer says %q", simplifiedStreet, before)
	}
	return set.Read(simplifiedStreet, []byte(text))
}

func TestARecipeAsksForTheSimplifiedScriptAndHerTrades(t *testing.T) {
	read, err := readSimplifiedStreet(t, askedForScript, askedForScript)
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string][]string{"gothic": nil, "concrete": nil,
		"mirror": {"womens-shelter", "dog-boutique"}}
	for name, carries := range asked {
		recipe, found := read.Recipe(name)
		if !found {
			t.Fatalf("no recipe %s", name)
		}
		if recipe.Script() != "simplified-chinese" ||
			!slices.Equal(recipe.Carries(), carries) {
			t.Errorf("%s says the script %q and carries %v", name,
				recipe.Script(), recipe.Carries())
		}
		if _, err := recipe.Construction(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	recipe, _ := read.Recipe("cyber")
	if recipe.Script() != "" || len(recipe.Carries()) != 0 {
		t.Errorf("the cyber recipe says %q and carries %v", recipe.Script(),
			recipe.Carries())
	}
}

func TestARecipeThatAsksForWhatTheSignsCannotKeepIsRefused(t *testing.T) {
	for _, changed := range []string{
		"script: klingon, pins:",
		"script: simplified-chinese, carries: [bank], pins:",
		"script: simplified-chinese, carries: [], pins:",
		"script: simplified-chinese, carries: [ramen, ramen], pins:"} {
		if _, err := readSimplifiedStreet(t, askedForScript,
			changed); err == nil {
			t.Errorf("a recipe that says %q was read", changed)
		}
	}
	if _, err := readSimplifiedStreet(t, aStreetRecipe, aStreetRecipe+
		"script: simplified-chinese, "); err == nil {
		t.Error("a street, which draws no signs, was given a script")
	}
}

func TestASetThatSaysNoScriptIsReadAsItWas(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "sets", "neon-street.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("neon-street.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cyber", "gothic", "concrete", "mirror"} {
		recipe, _ := read.Recipe(name)
		if recipe.Script() != "" || len(recipe.Carries()) != 0 {
			t.Errorf("%s says the script %q and carries %v", name,
				recipe.Script(), recipe.Carries())
		}
	}
}
