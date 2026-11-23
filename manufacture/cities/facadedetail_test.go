package cities

import "testing"

// detailSeeds are the seeds the detail tests draw, enough that a door and
// a shutter are among them.
const detailSeeds = 6

// terraceAt is a terrace of a culture at a detail level, drawn, or the
// test's end.
func terraceAt(t *testing.T, culture string, seed Seed,
	level Detail) []Rendered {
	t.Helper()
	design, err := terraceAtDetail(culture, seed, nil, level)
	if err != nil {
		t.Fatalf("%s seed %d at %s: %v", culture, seed, level, err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestAFacadeIsDrawnInMorePartsTheCloserTheLevel(t *testing.T) {
	for seed := range Seed(detailSeeds) {
		hull := len(terraceAt(t, "portofino", seed, "hull"))
		quick := len(terraceAt(t, "portofino", seed, "quick"))
		demo := len(terraceAt(t, "portofino", seed, "demo"))
		if !(hull < quick && quick < demo) {
			t.Errorf("seed %d: %d parts at the hull, %d quick, %d demo", seed,
				hull, quick, demo)
		}
	}
}

func TestAtTheHullAFacadeIsTheWallAloneWithNoOpeningDrawn(t *testing.T) {
	for _, part := range []string{"pane", "frame", "shutter", "light",
		"muntin", "slat", "stile", "sill", "head", "cornice"} {
		for seed := range Seed(detailSeeds) {
			if got := named(terraceAt(t, "portofino", seed, "hull"),
				part); len(got) != 0 {
				t.Errorf("seed %d: %d %s at the hull", seed, len(got), part)
			}
		}
	}
}

func TestTheQuickLevelIsTheTerraceAsItWasBeforeLevelsWereGiven(t *testing.T) {
	for _, culture := range []string{"groningen", "portofino", "midwest"} {
		old, err := Terrace(culture, 3)
		if err != nil {
			t.Fatal(err)
		}
		before, err := old.Rendered("quick")
		if err != nil {
			t.Fatal(err)
		}
		if digest(before) != digest(terraceAt(t, culture, 3, "quick")) {
			t.Errorf("%s: the quick level changed", culture)
		}
	}
}

func TestAtTheDemoLevelEveryPartOfAFacadeIsDrawnAndEachIsClosed(t *testing.T) {
	found := map[string]int{}
	for seed := range Seed(detailSeeds) {
		for _, one := range terraceAt(t, "portofino", seed, "demo") {
			found[one.Mesh.Name]++
			if !closed(one.Mesh) {
				t.Fatalf("seed %d: %s is not closed", seed, one.Mesh.Name)
			}
		}
	}
	for _, part := range []string{"light", "muntin", "sill", "head", "slat",
		"stile", "cornice", "door leaf", "door panel", "handle", "step"} {
		if found[part] == 0 {
			t.Errorf("no %s at the demo level", part)
		}
	}
	if found["pane"] != 0 {
		t.Errorf("%d whole panes at the demo level, where lights stand",
			found["pane"])
	}
}

func TestAShutterAtTheDemoLevelIsAFrameWithAMidRailAndManyTiltedSlats(
	t *testing.T) {
	fine := drawerOf(t, "portofino", 1).finish()
	drawn := terraceAt(t, "portofino", 3, "demo")
	panels := len(named(drawn, "rail")) / 3
	if panels == 0 || len(named(drawn, "rail"))%3 != 0 {
		rails := len(named(drawn, "rail"))
		t.Fatalf("%d rails, three to a shutter", rails)
	}
	slats := named(drawn, "slat")
	if len(slats) < 20*panels {
		t.Errorf("%d slats for %d shutters, about thirty each wanted",
			len(slats), panels)
	}
	for _, slat := range slats {
		low, high := reach(slat)
		if !(high.Z-low.Z > 2*fine.SlatLeaf) || !closed(slat.Mesh) {
			t.Fatalf("a slat standing %v deep, not tilted", high.Z-low.Z)
		}
	}
}

func TestAWindowAtTheDemoLevelHasTheLightsTheCultureSays(t *testing.T) {
	fine := drawerOf(t, "portofino", 1).finish()
	drawn := terraceAt(t, "portofino", 2, "demo")
	windows := len(named(drawn, "sill"))
	if windows == 0 {
		t.Fatal("no window with a sill")
	}
	if got, want := len(named(drawn, "light")),
		windows*fine.LightsAcross*fine.LightsUp; got < want {
		t.Errorf("%d lights for %d windows, at least %d wanted", got, windows,
			want)
	}
}

func TestACondominiumIsDrawnAtTheLevelItIsGiven(t *testing.T) {
	design, err := CondominiumAt(4, nil, "demo")
	if err != nil {
		t.Fatal(err)
	}
	demo, err := design.Rendered("demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(named(demo, "light")) == 0 || len(named(demo, "cornice")) == 0 {
		t.Error("a condominium at the demo level has no lights or cornice")
	}
}
