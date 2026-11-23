package stage_test

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// lawnOf is the thing the built world's lawn box becomes, with a heading
// said for it, or none when the text is empty.
func lawnOf(t *testing.T, heading string) *thing.Thing {
	t.Helper()
	raw, err := os.ReadFile("../set/testdata/built.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	if heading != "" {
		source = strings.Replace(source, "        size: [4, 5, 6]\n",
			"        size: [4, 5, 6]\n        heading: "+heading+"\n", 1)
	}
	made, err := set.Read("box-heading.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := made.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range world.Things {
		if each.Material != nil && each.Material.Name == "grass" {
			return each
		}
	}
	t.Fatal("the built world has no lawn")
	return nil
}

func TestBoxHeadingTurnsItsWidthClockwiseSeenFromAbove(t *testing.T) {
	// north is minus z, so a width that was east and is turned a quarter
	// clockwise lies south, which is plus z; a half turn lies west.
	cases := []struct {
		heading string
		want    maths.Vec
	}{
		{"{value: 90, unit: deg}", maths.V(0, 0, 1)},
		{"{value: 180, unit: deg}", maths.V(-1, 0, 0)},
		{"{value: 270, unit: deg}", maths.V(0, 0, -1)},
	}
	for _, each := range cases {
		turn := lawnOf(t, each.heading).Motion.At(0).Orientation
		got := turn.Out(maths.XAxis)
		if got.Sub(each.want).Len() > 1e-9 {
			t.Errorf("heading %s turns east to %v, want %v", each.heading,
				got, each.want)
		}
	}
}

func TestBoxWithNoHeadingIsNotTurned(t *testing.T) {
	turn := lawnOf(t, "").Motion.At(0).Orientation
	if turn != (maths.Orientation{}) {
		t.Errorf("a box with no heading is turned: %+v", turn)
	}
	if math.IsNaN(turn.Out(maths.XAxis).X) {
		t.Error("an unturned box has no east")
	}
}
