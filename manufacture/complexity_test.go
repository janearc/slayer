package manufacture

import (
	"testing"
)

func TestCoarseTreesHaveFewerFacesAndDemoHasSeparateBlades(t *testing.T) {
	row, _ := testKind("spruce")
	counts, leafFaces := map[string]int{}, map[string]int{}
	for _, detail := range Complexities() {
		made, err := MakeWithComplexity(row, Sow(11), detail)
		if err != nil {
			t.Fatal(err)
		}
		counts[detail] = made.faces()
		if group := made.groups[leaf]; group != nil {
			leafFaces[detail] = group.StoredFaces()
		}
	}
	// Demo's separate blades live in its leaf prototypes: each blade is
	// faces of its own, so demo keeps more leaf faces than quick's
	// shared tetrahedra, and plenty of them.
	if leafFaces["demo"] <= leafFaces["quick"] || leafFaces["demo"] < 1000 {
		t.Errorf("demo has no separate blades: %d leaf faces to quick's %d",
			leafFaces["demo"], leafFaces["quick"])
	}
	if counts["hull"] >= counts["clumps"] || counts["clumps"] >= counts["quick"] {
		t.Errorf("coarse representations did not reduce faces: %v", counts)
	}
}

func TestComplexityRefusesUnknownLevelsAndUnsownSeeds(t *testing.T) {
	row, _ := testKind("spruce")
	if _, err := MakeWithComplexity(row, Sow(1), "needlez"); err == nil {
		t.Error("accepted an unknown level")
	}
	for _, detail := range Complexities() {
		if _, err := MakeWithComplexity(row, seed{}, detail); err == nil {
			t.Errorf("%s accepted an unsown seed", detail)
		}
	}
}
