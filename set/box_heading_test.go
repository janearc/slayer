package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// boxHeadingWorld is the small built world with a heading said for its
// lawn, and the world read from it, or the error the reading gave.
func boxHeadingWorld(t *testing.T, heading string) (set.World, error) {
	t.Helper()
	source := strings.Replace(built(t), "        size: [4, 5, 6]\n",
		"        size: [4, 5, 6]\n        heading: "+heading+"\n", 1)
	made, err := set.Read("box-heading.yaml", []byte(source))
	if err != nil {
		return set.World{}, err
	}
	world, _ := made.World()
	return world, nil
}

func TestBoxHeadingIsReadAndOtherBoxesSayNone(t *testing.T) {
	world, err := boxHeadingWorld(t, "{value: 30, unit: deg}")
	if err != nil {
		t.Fatal(err)
	}
	boxes := world.Boxes()
	if angle, said := boxes[0].Heading(); !said || angle != 30 {
		t.Errorf("the lawn's heading = %g, %t, want 30, true", angle, said)
	}
	if angle, said := boxes[1].Heading(); said || angle != 0 {
		t.Errorf("the pane's heading = %g, %t, want none", angle, said)
	}
}

func TestBoxHeadingOfNoughtIsSaid(t *testing.T) {
	world, err := boxHeadingWorld(t, "{value: 0, unit: deg}")
	if err != nil {
		t.Fatal(err)
	}
	if angle, said := world.Boxes()[0].Heading(); !said || angle != 0 {
		t.Errorf("an explicit nought = %g, %t, want 0, true", angle, said)
	}
}

func TestBoxHeadingRefusesWhatIsNotADegreeTurn(t *testing.T) {
	for _, bad := range []string{
		"{value: 361, unit: deg}", "{value: -1, unit: deg}",
		"{value: 1, unit: rad}", "30"} {
		if _, err := boxHeadingWorld(t, bad); err == nil {
			t.Errorf("heading %s was read, want it refused", bad)
		}
	}
}
