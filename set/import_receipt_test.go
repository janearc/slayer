package set_test

import (
	"testing"

	"github.com/janearc/slayer/set"
)

func TestImportedReceiptGetsAuthoredRightsAndMaterialProvenance(t *testing.T) {
	read, err := set.Read(importedPath(t), []byte(importedSet))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	part := world.Imports()[0]
	if part.RequestedHeightMetres() != 2 || part.License() != "CC0-1.0" ||
		part.Source() != "Self-authored CC0 mannequin" {
		t.Fatalf("import receipt provenance = %q %q %g", part.Source(),
			part.License(), part.RequestedHeightMetres())
	}
	if _, has := part.SelectedScene(); has {
		t.Fatal("implicit one-scene choice was recorded as authored selection")
	}
	materials := world.Materials()
	if len(materials) != 1 || materials[0].Name() != "matte" ||
		materials[0].Source() != "eye" {
		t.Fatalf("native material provenance = %#v", materials)
	}
}
