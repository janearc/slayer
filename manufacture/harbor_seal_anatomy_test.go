package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"testing"
)

func TestHarborSealStudiesMakeTheWholeAnimal(t *testing.T) {
	for _, name := range []string{"harbor-seal", "harbor-seal-dry"} {
		kind, err := referencePinnipedKind(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := MakeWithComplexity(kind, Sow(407), "quick")
		if err != nil {
			t.Fatal(err)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		if len(parts) < 6 {
			t.Fatalf("%s: %d parts", name, len(parts))
		}
		skin, ok := parts[0].Shape.(*mesh.Mesh)
		if !ok || !skin.Closed() {
			t.Fatalf("%s: the first part is not a closed skin", name)
		}
		if parts[0].Material.Name != "harbor seal skin" || parts[1].Material.Name != "harbor seal hair" {
			t.Fatalf("%s: skin and hair are not the first two parts", name)
		}
	}
	wet, _ := referencePinnipedKind("harbor-seal")
	dry, _ := referencePinnipedKind("harbor-seal-dry")
	if wet == dry {
		t.Fatal("wet and dry are one kind")
	}
	if _, err := KindNamed("harbor-seal-fur"); err == nil {
		t.Fatal("found a kind nobody made")
	}
}

// Both looks are made at the seeds the catalogue and the set tests draw
// from, not only the one above: a nostril shut as far as a look can shut it
// is a ring the mesh takes, at every seed.
func TestAShutHarborSealIsMadeAtEverySeed(t *testing.T) {
	for _, name := range []string{"harbor-seal", "harbor-seal-dry"} {
		kind, err := referencePinnipedKind(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, number := range []uint64{0, 1, 2, 407} {
			if _, err := MakeWithComplexity(kind, Sow(number),
				"quick"); err != nil {
				t.Errorf("%s at seed %d: %v", name, number, err)
			}
		}
	}
}
