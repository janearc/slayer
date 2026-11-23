package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// lashingsSetFile is the lashings set the key tests change, from the set
// directory.
const lashingsSetFile = "../sets/lashings-street-10.yaml"

// lashingsRecipe is the recipe line of that set the tests change.
const lashingsRecipe = "      density: 1.0\n"

// lashingsSetWith is the lashings set with its density line replaced.
func lashingsSetWith(t *testing.T, lines string) (set.Set, error) {
	t.Helper()
	text, err := os.ReadFile(lashingsSetFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), lashingsRecipe) {
		t.Fatalf("the lashings set has no %q to change", lashingsRecipe)
	}
	edited := strings.Replace(string(text), lashingsRecipe, lines, 1)
	return set.Read(lashingsSetFile, []byte(edited))
}

func TestALashedStreetRecipeSaysDensityStreetLengthAndSign(t *testing.T) {
	made, err := lashingsSetWith(t, "      density: 0.4\n"+
		"      street: {value: 14, unit: m}\n"+
		"      length: {value: 60, unit: m}\n"+
		"      sign: human-scale-2m\n")
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := made.Recipe("hung")
	if !found || recipe.Kind() != "lashed-street" {
		t.Errorf("the recipe is %v, found %v", recipe.Kind(), found)
	}
	if _, err := recipe.Construction(); err != nil {
		t.Error(err)
	}
}

func TestALashedStreetRecipeSaysItsStyle(t *testing.T) {
	for _, style := range []string{"festival", "lantern-rows"} {
		made, err := lashingsSetWith(t, "      style: "+style+"\n")
		if err != nil {
			t.Fatal(err)
		}
		recipe, _ := made.Recipe("hung")
		if _, err := recipe.Construction(); err != nil {
			t.Errorf("style %s: %v", style, err)
		}
	}
}

func TestADensityOfNoughtIsASaidDensity(t *testing.T) {
	if _, err := lashingsSetWith(t, "      density: 0\n"); err != nil {
		t.Error(err)
	}
}

// Anti: a density out of range or no number, a street or length with no
// unit or below one, and a sign that is no kind of part, are refused.
func TestALashedStreetRecipeRefusals(t *testing.T) {
	cases := map[string]struct{ lines, want string }{
		"density over one": {"      density: 1.5\n", "density"},
		"density below":    {"      density: -0.1\n", "density"},
		"density a word":   {"      density: thick\n", "density"},
		"street no unit":   {"      street: {value: 12}\n", "street"},
		"length too short": {"      length: {value: 0.5, unit: m}\n",
			"length"},
		"style unknown": {"      style: bunting\n", "style"},
		"sign unknown":  {"      sign: nothing-of-the-kind\n", "sign"},
		"sign itself":   {"      sign: lashed-street\n", "lashed street"},
		"sign cluster":  {"      sign: scavenged-cluster\n", "lashed street"}}
	for name, one := range cases {
		_, err := lashingsSetWith(t, one.lines)
		mustHold(t, name, err, one.want)
	}
}

// schemaLine is the line of the lashings set that names its schema.
const schemaLine = "$schema: https://github.com/janearc/slayer/schemas/" +
	"ecosystem-v2.json\n"

// Anti: the keys of a lashed street are refused by a recipe of any other
// kind. A set that names its schema has the key refused by the schema
// before the reader is asked, naming the density; a set that names none
// is not checked by a schema, and the reader refuses it in its own words.
// The lashings set says setting: space, a key of version 2 that a set with
// no schema is refused for, so that line goes with the schema line.
func TestLashingKeysOnAnotherKindAreRefused(t *testing.T) {
	text, err := os.ReadFile(lashingsSetFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), schemaLine) {
		t.Fatalf("the lashings set does not say %q", schemaLine)
	}
	other := strings.Replace(string(text), "kind: lashed-street",
		"kind: yurt", 1)
	_, err = set.Read(lashingsSetFile, []byte(other))
	mustHold(t, "density on a yurt", err,
		"additional properties 'density' not allowed")
	bare := strings.Replace(other, schemaLine, "", 1)
	bare = strings.Replace(bare, "  setting: space\n", "", 1)
	_, err = set.Read(lashingsSetFile, []byte(bare))
	mustHold(t, "density on a yurt, no schema", err, "not a lashed street")
}
