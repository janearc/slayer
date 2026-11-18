package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestSpitzerHasThePublishedSolarCellInventory(t *testing.T) {
	row, err := manufacture.KindNamed("spitzer")
	if err != nil {
		t.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(1), detail)
		if err != nil {
			t.Fatal(err)
		}
		if made.PartCount() < 600 {
			t.Fatal("Spitzer remains below the requested part floor", made.PartCount())
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		cells := 0
		for _, part := range parts {
			if part.Material.Name != "solar array" {
				continue
			}
			cells += connectedPanels(part.Shape.(*mesh.Mesh))
			if part.Material.Metal == nil || math.Abs(part.Material.Metal.N[0]-3.878) > 1e-12 {
				t.Fatal("solar cells are not using the published GaAs optics")
			}
		}
		if cells != 2*392 {
			t.Fatalf("%s: %d distinct cells, want 784", detail, cells)
		}
	}
}

// anti-test of TestSpitzerHasThePublishedSolarCellInventory
func TestSpitzerAliasHasTheSameInventory(t *testing.T) {
	var counts []int
	for _, name := range []string{"spitzer", "space-observatory"} {
		row, err := manufacture.KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := manufacture.Make(row, manufacture.Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		counts = append(counts, made.PartCount())
	}
	if counts[0] != counts[1] || counts[0] < 600 {
		t.Fatal("alias bypassed the matured solar assembly", counts)
	}
}
