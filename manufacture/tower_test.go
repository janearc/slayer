package manufacture

import (
	"errors"
	"testing"
)

func TestTheTowerKindIsAPlainBlockOfGlassGridAndPlantWithoutARecipe(t *testing.T) {
	made := mustMake(t, "tower", 1)
	if got := len(made.slots()); got < 4 {
		t.Errorf("%d materials, want glass, grid, plant and podium or "+
			"crown at least", got)
	}
	if bounds := made.Bounds(); bounds.Min.Y < -epsilon || bounds.Max.Y < 100 {
		t.Errorf("a plain tower spans %v to %v up", bounds.Min.Y,
			bounds.Max.Y)
	}
}

func TestATowerNeedsASownSeed(t *testing.T) {
	if _, err := (towerRow{}).made(seed{}); !errors.Is(err, errGood) {
		t.Errorf("an unsown seed made a tower: %v", err)
	}
}

func TestATowerRecipeKeepsItsPinsAndRefusesThoseItCannot(t *testing.T) {
	yes := true
	row, err := Tower(TowerSpec{Irregularity: 1, Plan: "star", Shafts: 2,
		Span: "skybridge", Setbacks: &yes})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := makeOf(row, sow(3)); err != nil {
		t.Errorf("a pinned tower was not made: %v", err)
	}
	for name, spec := range map[string]TowerSpec{
		"irregularity over one": {Irregularity: 2},
		"unknown plan":          {Plan: "dome"},
		"too many shafts":       {Shafts: 99},
	} {
		if _, err := Tower(spec); err == nil {
			t.Errorf("%s: a recipe was accepted", name)
		}
	}
}

func TestATowerOfDrawnIrregularityDiffersFromSeedToSeed(t *testing.T) {
	row, err := Tower(TowerSpec{Drawn: true})
	if err != nil {
		t.Fatal(err)
	}
	heights := map[float64]bool{}
	for number := range uint64(12) {
		made, err := makeOf(row, sow(number))
		if err != nil {
			t.Fatal(err)
		}
		heights[made.Bounds().Max.Y] = true
	}
	if len(heights) < 3 {
		t.Errorf("twelve towers of drawn irregularity had %d heights",
			len(heights))
	}
}
