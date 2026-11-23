package manufacture

import "testing"

// The village kinds in the manufactory: named, levelled as buildings,
// stood on terrain house by house, and what they refuse. Written
// 2026-11-20 beside the code and not yet run.

// TestVillageKindsAreBuildings checks that both kinds are in the table,
// levelled part by part on the buildings' ladder, slope aware, and stood
// on terrain house by house.
func TestVillageKindsAreBuildings(t *testing.T) {
	for _, name := range []string{VillageHouseKind, VillageKind} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		if !LevelsParts(row) || ladderName(row) != buildingLadder {
			t.Errorf("%s is not levelled as a building", name)
		}
		if !SlopeAware(name) || !StandsOnTerrain(row) {
			t.Errorf("%s is not stood on the ground under it", name)
		}
	}
}

// TestVillageHouseIsLit checks that a house made at the demo level carries
// lights: its stair door's lamp and its lit rooms.
func TestVillageHouseIsLit(t *testing.T) {
	row, err := KindNamed(VillageHouseKind)
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(row, sow(2), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(made.lights) == 0 {
		t.Error("a village house at the demo level has no light")
	}
}

// TestVillageRefusesNegativeParcel is an anti-test: a parcel below nought
// and one too small for a plot are refused when the recipe is made, and
// a level that is not a building's is refused when the kind is.
func TestVillageRefusesNegativeParcel(t *testing.T) {
	for _, spec := range []VillageSpec{{Width: -1}, {Width: 2, Depth: 2}} {
		if _, err := Village(spec); err == nil {
			t.Errorf("a village of %+v is not refused", spec)
		}
	}
	row, err := KindNamed(VillageKind)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MakeWithComplexity(row, sow(1), "clumps"); err == nil {
		t.Error("a village at the clumps level is not refused")
	}
}
