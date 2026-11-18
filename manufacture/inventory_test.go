package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestPartCountRetainsPanelsThroughSharingAndPlacement(t *testing.T) {
	assembly := &instrumentAssembly{fine: cut{around: 12},
		object: good{dressed: groundLooks()}}
	assembly.add(hexagonalPrimary{rings: 2, diameter: 6, fill: .9,
		depth: .1, thickness: .1}, maths.Local(), spaceMirror)
	assembly.box(1, 1, 1, maths.V(8, 0, 0), spaceStructure)
	// Two hexagonal rings contain 6 + 12 panels, beside one box.
	if got := assembly.object.PartCount(); got != 19 {
		t.Fatalf("before sharing: %d parts, want 19", got)
	}
	if err := assembly.shareComponents(); err != nil {
		t.Fatal(err)
	}
	if got := assembly.object.PartCount(); got != 19 {
		t.Fatalf("sharing lost panels: %d", got)
	}
	// The rendering group supersedes any retained primitive in its slot.
	assembly.object.parts = append(assembly.object.parts,
		part{shape: boxShape(1, 1, 1), of: spaceStructure})
	if got := assembly.object.PartCount(); got != 19 {
		t.Fatalf("a shadowed primitive was counted twice: %d", got)
	}
	assembly.object.parts = nil
	moved, err := assembly.object.placed(maths.NewSubspace(
		maths.V(4, 5, 6), maths.Identity, 2))
	if err != nil {
		t.Fatal(err)
	}
	joined, err := gather(assembly.object, moved)
	if err != nil {
		t.Fatal(err)
	}
	if got := joined.PartCount(); got != 38 {
		t.Fatalf("placement or composition changed inventory: %d", got)
	}
}

// anti-test of TestPartCountRetainsPanelsThroughSharingAndPlacement
func TestPartCountDoesNotCountTrianglesOrMaterialBatches(t *testing.T) {
	for _, around := range []int{12, 48} {
		assembly := &instrumentAssembly{fine: cut{around: around},
			object: good{dressed: groundLooks()}}
		assembly.box(1, 1, 1, maths.V(0, 0, 0), spaceStructure)
		assembly.box(1, 1, 1, maths.V(3, 0, 0), spaceStructure)
		things, err := assembly.object.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		if len(things) != 1 || assembly.object.PartCount() != 2 {
			t.Fatal("parts were confused with material batches")
		}
	}
}

func TestPartCountIncludesCompoundMirrorForms(t *testing.T) {
	forms := []struct {
		shape form
		want  int
	}{
		{annularPrimary{count: 36, hole: 1}, 36},
		{owlSegmented{count: 3048, hole: 5}, 3048},
		{mirrorDish{radius: 1, depth: .1}, 1},
		{mirrorDish{radius: 1, depth: .1, segmented: true}, 19},
	}
	for _, row := range forms {
		made := good{parts: []part{{shape: row.shape}}}
		if got := made.PartCount(); got != row.want {
			t.Fatalf("%T: %d parts, want %d", row.shape, got, row.want)
		}
	}
	if got := (good{}).PartCount(); got != 0 {
		t.Fatalf("empty specimen has %d parts", got)
	}
}

func TestPartCountMultipliesAnArrayInventory(t *testing.T) {
	for _, pair := range [][2]string{{"sma-antenna", "sma-array"}, {"geodss", "geodss-site"}} {
		one, err := KindNamed(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		many, err := KindNamed(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		prototype, err := Make(one, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		array, err := Make(many, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		copies := 8
		if pair[1] == "geodss-site" {
			copies = 3
		}
		parts := prototype.PartCount()
		if pair[1] == "geodss-site" {
			// The standalone kind includes one ancillary building per dome;
			// the site recipe explicitly omits that building.
			parts--
		}
		if got, want := array.PartCount(), copies*parts; got != want {
			t.Fatalf("%s: %d parts, want %d", pair[1], got, want)
		}
	}
}
