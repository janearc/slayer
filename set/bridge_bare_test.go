package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// bridgeSetFile is the bridge set the bare tests change, from the set
// directory.
const bridgeSetFile = "../sets/bridge-of-buildings.yaml"

// bridgeRecipe is the recipe of that set, whose depth line the tests change.
const bridgeRecipe = "      depth: 2\n"

// bridgeSetWith is the bridge set with its depth line replaced.
func bridgeSetWith(t *testing.T, line string) error {
	t.Helper()
	text, err := os.ReadFile(bridgeSetFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), bridgeRecipe) {
		t.Fatalf("the bridge set has no %q to change", bridgeRecipe)
	}
	edited := strings.Replace(string(text), bridgeRecipe, line, 1)
	_, err = set.Read(bridgeSetFile, []byte(edited))
	return err
}

// A bridge recipe that is bare and says no depth is read.
func TestABareBridgeRecipeIsRead(t *testing.T) {
	if err := bridgeSetWith(t, "      bare: true\n"); err != nil {
		t.Error(err)
	}
}

// Anti: a bare bridge that says a depth is refused, and so is a bare that
// is not true or false; a bridge that is not bare keeps its depth.
func TestABareBridgeRecipeRefusals(t *testing.T) {
	mustHold(t, "bare and depth", bridgeSetWith(t,
		"      bare: true\n      depth: 2\n"), "no depth")
	mustHold(t, "bare not a truth", bridgeSetWith(t, "      bare: maybe\n"),
		"bare")
	if err := bridgeSetWith(t, bridgeRecipe); err != nil {
		t.Errorf("a bridge not bare: %v", err)
	}
}
