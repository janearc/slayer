package main

import (
	"os"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// the shoebox's set reads and builds: its boxes in their materials, its
// four bulbs, the night's flood and saturn's sun, and saturn.
func TestTheShoeboxIsBuilt(t *testing.T) {
	written, err := os.ReadFile("../../sets/shoebox.yaml")
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("shoebox.yaml", written)
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	boxes := map[string]int{}
	for _, each := range world.Things {
		if each.Material != nil {
			boxes[each.Material.Name]++
		}
	}
	for material, want := range map[string]int{"night-grass": 1,
		"concrete": 1, "corten": 1, "garage-door": 1, "asphalt": 1,
		"lit-window": 7} {
		if boxes[material] != want {
			t.Errorf("%d boxes of %s, not %d", boxes[material], material,
				want)
		}
	}
	if lamps := len(world.Lamps); lamps != 6 {
		t.Errorf("%d lamps, not six", lamps)
	}
	if _, err := read.Shot("front", places{}.where); err != nil {
		t.Error(err)
	}
}

// -at reads a name and three numbers, and says the places back.
func TestAPlaceIsANameAndThreeNumbers(t *testing.T) {
	at := places{}
	if err := at.Set("the box = 1, -2.5,3"); err != nil {
		t.Fatal(err)
	}
	if spot, found := at.where("the box "); !found ||
		spot != maths.V(1, -2.5, 3) || at.String() != "map[the box :{1 -2.5 3}]" {
		t.Errorf("the box is at %v, %v; said back as %s", spot, found,
			at.String())
	}
	if _, found := at.where("the shed"); found {
		t.Error("the shed was found")
	}
	for _, said := range []string{"the box", "the box=1,2",
		"the box=1,2,3,4", "the box=1,two,3"} {
		if err := at.Set(said); err == nil {
			t.Errorf("%q was taken", said)
		}
	}
}
