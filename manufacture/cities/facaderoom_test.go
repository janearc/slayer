package cities

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestAtTheDemoLevelEveryWindowHasARoomBehindClearGlass(t *testing.T) {
	drawn := terraceAt(t, "portofino", 3, "demo")
	windows := len(named(drawn, "sill"))
	for _, part := range []string{"floor", "ceiling", "back wall"} {
		if got := len(named(drawn, part)); got < windows {
			t.Errorf("%d %s for %d windows", got, part, windows)
		}
	}
	if got := len(named(drawn, "side wall")); got < 2*windows {
		t.Errorf("%d side walls for %d windows", got, windows)
	}
	for _, light := range named(drawn, "light") {
		if light.Material.Through != 1 || light.Material.Gloss != 1 {
			t.Fatalf("a light of %+v is not clear and mirroring",
				light.Material)
		}
		if light.Material.Glow != (maths.Light{}) {
			t.Fatal("a light of the glass itself glows at the demo level")
		}
	}
}

func TestALitWindowHasALampThatGlowsAndADarkOneHasNone(t *testing.T) {
	lamps, curtains := 0, 0
	for seed := range Seed(detailSeeds) {
		drawn := terraceAt(t, "portofino", seed, "demo")
		for _, lamp := range named(drawn, "lamp") {
			lamps++
			if lamp.Material.Glow == (maths.Light{}) {
				t.Fatalf("seed %d: a lamp that does not glow", seed)
			}
			if lamp.Lamp == nil || !(lamp.Lamp.Heat > 0 && lamp.Lamp.Luminance > 0) {
				t.Fatalf("seed %d: a lamp that is no light: %+v", seed,
					lamp.Lamp)
			}
		}
		curtains += len(named(drawn, "curtain"))
		windows := len(named(drawn, "sill"))
		if len(named(drawn, "lamp")) > windows {
			lamps := len(named(drawn, "lamp"))
			t.Errorf("seed %d: %d lamps in %d windows", seed,
				lamps, windows)
		}
	}
	if lamps == 0 || curtains == 0 {
		t.Errorf("%d lamps and %d curtains over %d terraces", lamps,
			curtains, detailSeeds)
	}
}

func TestBelowTheDemoLevelThereIsNoRoomAndALitPaneStillGlows(t *testing.T) {
	for _, level := range []Detail{"hull", "quick"} {
		drawn := terraceAt(t, "portofino", 3, level)
		for _, part := range []string{"floor", "ceiling", "back wall",
			"lamp", "curtain"} {
			if got := len(named(drawn, part)); got != 0 {
				t.Errorf("%d %s at %s", got, part, level)
			}
		}
	}
	glowing := 0
	for _, pane := range named(terraceAt(t, "portofino", 3, "quick"),
		"pane") {
		if pane.Material.Glow != (maths.Light{}) {
			glowing++
		}
	}
	if glowing == 0 {
		t.Error("no pane glows at the quick level")
	}
}

func TestNoWallStandsInARoom(t *testing.T) {
	drawn := terraceAt(t, "portofino", 3, "demo")
	for _, back := range named(drawn, "back wall") {
		low, high := reach(back)
		middle := low.Add(high).Scale(1 / float64(half))
		for _, wall := range named(drawn, "wall") {
			lowWall, highWall := reach(wall)
			if middle.X > lowWall.X+tolerance &&
				middle.X < highWall.X-tolerance &&
				middle.Y > lowWall.Y+tolerance &&
				middle.Y < highWall.Y-tolerance &&
				middle.Z > lowWall.Z+tolerance &&
				middle.Z < highWall.Z-tolerance {
				t.Fatalf("a wall from %v to %v stands in a room at %v",
					lowWall, highWall, middle)
			}
		}
	}
}

func TestEveryPartOfARoomIsClosedAndAShallowHouseHasNoRoom(t *testing.T) {
	for _, one := range terraceAt(t, "portofino", 3, "demo") {
		if !closed(one.Mesh) {
			t.Fatalf("%s is not closed", one.Mesh.Name)
		}
	}
	facade := facade{Recess: 0.15, Finish: finish{RoomDepth: 3, BackWall: 0.6}}
	if got := facade.cavityFor(0.5); got != 0 {
		t.Errorf("a house 0.5 m deep has rooms %v deep", got)
	}
	if got := facade.cavityFor(10); got != 3 {
		t.Errorf("a house 10 m deep has rooms %v deep, not 3", got)
	}
}
