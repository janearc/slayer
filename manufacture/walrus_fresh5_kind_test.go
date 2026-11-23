package manufacture

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func TestWalrusSetKindIsTheWalrus(t *testing.T) {
	kind, err := KindNamed("walrus-fresh-5")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(17), "quick")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	original := testWalrusFresh5(t, 0, 0)
	want := walrusPartIndex(1, walrusHeadPartCount)
	if len(actual) != want || len(original) != want || made.PartCount() != want {
		t.Fatalf("parts: made %d, direct %d, inventory %d, want %d",
			len(actual), len(original), made.PartCount(), want)
	}
	for i := range original {
		if !reflect.DeepEqual(actual[i].Shape, original[i].Shape) ||
			!reflect.DeepEqual(actual[i].Material, original[i].Material) ||
			actual[i].At(0) != original[i].At(0) {
			t.Fatalf("geometry, look or placement changed at part %d", i)
		}
	}
	_, hideFaces, closed := testWalrusMesh(t, actual[walrusHidePart])
	if !closed {
		t.Fatal("the hide is open")
	}
	hide := actual[walrusHidePart].Material
	if hide.Name != "walrus hide" || hide.Colour != walrusCinnamon ||
		hide.Gloss != .13 || hide.Satin != .59 {
		t.Fatalf("the hide's material is %+v", hide)
	}
	faces := 0
	for _, part := range original {
		switch part.Shape.(type) {
		case *mesh.Mesh, *shape.Mesh:
			_, each, _ := testWalrusMesh(t, part)
			faces += len(each)
		}
	}
	if made.faces() != faces || faces <= len(hideFaces) {
		t.Fatalf("stored faces: got %d want %d", made.faces(), faces)
	}
}

func TestWalrusSetKindStaysInsideItsExtent(t *testing.T) {
	kind, err := KindNamed("walrus-fresh-5")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(17), "quick")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	extent := made.Bounds()
	for i, p := range actual {
		b := transformed(p.Shape.Bounds(), p.At(0))
		if b.Min.X < extent.Min.X || b.Min.Y < extent.Min.Y ||
			b.Min.Z < extent.Min.Z || b.Max.X > extent.Max.X ||
			b.Max.Y > extent.Max.Y || b.Max.Z > extent.Max.Z {
			t.Fatalf("part %d outside extent", i)
		}
	}
	place := maths.NewSubspace(maths.V(4, 5, 6), maths.Rotation(maths.Up, .4), 1)
	placed, err := made.Things(place)
	if err != nil {
		t.Fatal(err)
	}
	for i := range actual {
		if placed[i].At(0) != actual[i].At(0).Within(place) {
			t.Fatalf("part %d misplaced", i)
		}
	}
	if _, err = made.Things(maths.Subspace{Scale: -1}); err == nil {
		t.Fatal("accepted invalid placement")
	}
}
