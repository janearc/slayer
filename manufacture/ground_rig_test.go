package manufacture_test

import (
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

// Splitting for animation must keep the exact existing inventory, including
// the shell and Rubin's service wing; no cutaway or replacement model.
func TestGroundRigPreservesFactoryInventory(t *testing.T) {
	faces := func(parts []thing.Thing) int {
		n := 0
		for _, part := range parts {
			m, ok := part.Shape.(*mesh.Mesh)
			if !ok {
				t.Fatalf("unexpected rig shape %T", part.Shape)
			}
			n += m.FaceCount()
		}
		return n
	}
	for _, name := range []string{"keck-i", "keck-ii", "subaru", "rubin", "elt"} {
		rig, err := manufacture.MakeGroundRig(name, "quick")
		if err != nil {
			t.Fatal(err)
		}
		kind, err := manufacture.KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(5), "quick")
		if err != nil {
			t.Fatal(err)
		}
		original, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		if faces(original) != faces(rig.Fixed)+faces(rig.Azimuth)+faces(rig.Optical)+faces(rig.Enclosure) {
			t.Fatalf("%s lost or duplicated geometry", name)
		}
		if len(rig.Fixed) == 0 || len(rig.Azimuth) == 0 || len(rig.Optical) == 0 || rig.Pivot.Y <= 0 {
			t.Fatalf("%s missing mechanical role", name)
		}
		found := false
		for _, part := range rig.Optical {
			if part.Material.Name == "primary mirror" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s has no moving primary", name)
		}
	}
}
