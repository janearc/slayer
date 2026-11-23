package session

import (
	"testing"

	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// sparingStaging is a stage for the set of warmSetText, whose camera is
// the one of lookingNorth.
func sparingStaging(t *testing.T) *staging {
	t.Helper()
	positions := readPositionSet(t, warmSetText())
	return warmStaging(t, positions)
}

// TestANamedTowerOutOfViewIsMadePartByPartWithFewerFacesThanWhole checks
// that where scrooge is asked a named tower behind the camera is made part
// by part, and comes out with fewer faces than the same tower made whole.
func TestANamedTowerOutOfViewIsMadePartByPartWithFewerFacesThanWhole(
	t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	stage := sparingStaging(t)
	parts, sparing, err := stage.sparingly("tower", "", "quick", 1,
		"far tower", standingAt(behind))
	if err != nil || !sparing {
		t.Fatalf("a named tower: sparing %v, %v; want part by part",
			sparing, err)
	}
	whole, err := stage.stockOf("tower", "", "quick", 1, "far tower")
	if err != nil {
		t.Fatal(err)
	}
	if parts.faces == 0 || parts.faces >= whole.faces {
		t.Errorf("part by part %d faces, whole %d", parts.faces, whole.faces)
	}
}

// TestAThingThatIsNotSparedIsLeftToTheStock is the anti-test of the rule:
// with scrooge not asked, with no name, or of a kind not drawn part by
// part, the stock makes it whole as it did.
func TestAThingThatIsNotSparedIsLeftToTheStock(t *testing.T) {
	for _, each := range []struct {
		about, kind, name, asked string
	}{
		{"scrooge not asked", "tower", "far tower", ""},
		{"no name", "tower", "", "1"},
		{"a kind not drawn part by part", "spruce", "first spruce", "1"},
	} {
		t.Setenv(scrooge.Variable, each.asked)
		stage := sparingStaging(t)
		_, sparing, err := stage.sparingly(each.kind, "", "quick", 1,
			each.name, standingAt(behind))
		if err != nil || sparing {
			t.Errorf("%s: sparing %v, %v; want it left to the stock",
				each.about, sparing, err)
		}
	}
}
