package cities

import (
	"slices"
	"testing"
)

// endSeeds is how many seeds the end-colour tests draw a terrace from.
const endSeeds = 30

func TestTheRunOfATerraceNamesTheColoursOfItsFirstAndLastBay(t *testing.T) {
	for seed := range Seed(endSeeds) {
		run, err := TerraceRun("portofino", seed)
		if err != nil {
			t.Fatal(err)
		}
		plan := drawerOf(t, "portofino", seed).row()
		if run.First != plan.colours[0].Name {
			t.Errorf("seed %d: first bay %q, plan says %q", seed, run.First,
				plan.colours[0].Name)
		}
		if run.Last != plan.colours[len(plan.colours)-1].Name {
			t.Errorf("seed %d: last bay %q, plan says %q", seed, run.Last,
				plan.colours[len(plan.colours)-1].Name)
		}
	}
}

func TestTheEndColoursOfARunAreColoursOfTheCulturesPalette(t *testing.T) {
	names := namesOf(drawerOf(t, "portofino", 1).options("palette"))
	for seed := range Seed(endSeeds) {
		run, err := TerraceRun("portofino", seed)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(names, run.First) ||
			!slices.Contains(names, run.Last) {
			t.Errorf("seed %d: ends %q and %q are not among %v", seed,
				run.First, run.Last, names)
		}
	}
}

func TestARunSaysWhetherItsCultureKeepsNeighboursApart(t *testing.T) {
	for _, culture := range []string{"portofino", "groningen"} {
		run, err := TerraceRun(culture, 3)
		if err != nil {
			t.Fatal(err)
		}
		if !run.Differ {
			t.Errorf("%s says no two neighbours share a colour, and its "+
				"run does not", culture)
		}
	}
}

func TestTheRoomOfARunIsTheSameWhateverItsEndColoursAreAskedFor(t *testing.T) {
	run, err := TerraceRun("portofino", 5)
	if err != nil {
		t.Fatal(err)
	}
	plan := drawerOf(t, "portofino", 5).row()
	if want := float64(plan.bays) * plan.pitch; run.Length != want {
		t.Errorf("a run is %g m long, its plan says %g", run.Length, want)
	}
}

func TestARunOfACultureThatIsNotKnownIsRefused(t *testing.T) {
	if _, err := TerraceRun("atlantis", 1); err == nil {
		t.Error("a run of a culture with no file was drawn")
	}
}
