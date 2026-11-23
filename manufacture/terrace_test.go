package manufacture

import (
	"errors"
	"testing"
)

func TestEveryTerraceKindIsMadeOfManyMaterialsAndStandsOnTheGround(t *testing.T) {
	for _, name := range []string{"terrace-groningen", "terrace-portofino",
		"terrace-midwest", "condominium"} {
		made := mustMake(t, name, 1)
		if got := len(made.slots()); got < 5 {
			t.Errorf("%s: %d materials, want walls, roof, frames, glass "+
				"and doors at least", name, got)
		}
		if bounds := made.Bounds(); bounds.Min.Y < -epsilon ||
			bounds.Max.Y < 6 {
			t.Errorf("%s spans %v to %v up", name, bounds.Min.Y,
				bounds.Max.Y)
		}
	}
}

func TestATerraceNeedsASownSeed(t *testing.T) {
	for _, row := range []maker{terraceRow{culture: "groningen"},
		condominiumRow{}} {
		if _, err := row.made(seed{}); !errors.Is(err, errGood) {
			t.Errorf("an unsown seed made a terrace: %v", err)
		}
	}
}

func TestATerraceIsMadeAtEveryDetailLevelAndNoOther(t *testing.T) {
	row := terraceRow{culture: "portofino"}
	for _, level := range []string{"hull", "clumps", "quick", "demo"} {
		if _, err := row.madeAt(sow(1), level); err != nil {
			t.Errorf("%s: %v", level, err)
		}
	}
}

func TestATerraceOfNoCultureIsRefused(t *testing.T) {
	if _, err := (terraceRow{culture: "atlantis"}).made(sow(1)); err == nil {
		t.Error("a terrace of a culture that is not one was made")
	}
}
