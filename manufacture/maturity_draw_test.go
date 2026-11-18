//go:build draw

package manufacture

import (
	"slices"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/research"
)

// TestMaturityReportCoversTheWholeTable makes every kind at every detail,
// which takes seconds for the plants, and checks the report against the
// table: one row a kind in KindNames' order, every row with a family and
// the quick detail, every measured number a sourced one from research,
// and the one kind built of the most meshes counted as it was by hand.
func TestMaturityReportCoversTheWholeTable(t *testing.T) {
	report, err := MaturityReport()
	if err != nil {
		t.Fatal(err)
	}
	measured, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, one := range report {
		names = append(names, one.Name)
		if one.Family == "other" || !slices.Contains(one.Details, "quick") {
			t.Errorf("%s: family %q details %v", one.Name, one.Family, one.Details)
		}
		if one.Parts == 0 && one.MeshMembers == 0 {
			t.Errorf("%s has no geometry", one.Name)
		}
		for _, name := range one.Measured {
			number, err := measured.Get(name)
			if err != nil || number.Kind != numbers.Measured || number.From == nil {
				t.Errorf("%s: %s is listed as measured: %v", one.Name, name, err)
			}
		}
		if one.Name == "owl" && (one.Parts != 0 || one.MeshMembers != 526 ||
			one.MeshFaces != 160680 || one.Pieces != 3788) {
			t.Errorf("owl: parts %d members %d faces %d pieces %d", one.Parts, one.MeshMembers, one.MeshFaces, one.Pieces)
		}
	}
	if !slices.Equal(names, KindNames()) {
		t.Fatalf("report names %v, kinds %v", names, KindNames())
	}
}
