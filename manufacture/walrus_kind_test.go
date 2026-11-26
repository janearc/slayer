package manufacture

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestWalrusSetKindIsTheWalrus(t *testing.T) {
	kind, err := referencePinnipedKind("walrus-fresh-5")
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
	original := testWalrusStudy(t, 0, 0)
	want := 9
	if len(actual) != want || len(original) != want || made.PartCount() != want {
		t.Fatalf("parts: made %d, direct %d, inventory %d, want %d",
			len(actual), len(original), made.PartCount(), want)
	}
	for i := range original {
		if !reflect.DeepEqual(actual[i].Shape, original[i].Shape) || actual[i].At(0) != original[i].At(0) {
			t.Fatalf("geometry or placement changed at part %d", i)
		}
		checkWalrusMaterial(t, i, actual[i], original[i])
	}
	_, hideFaces, closed := testWalrusMesh(t, actual[0])
	if !closed {
		t.Fatal("the hide is open")
	}
	hide := actual[0].Material
	if hide.Name != "walrus hide" || hide.Colour != maths.RGB(.25, .150, .085) ||
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
	kind, err := referencePinnipedKind("walrus-fresh-5")
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

// checkWalrusMaterial compares every material field and samples the actual
// texture. A texture's private rig may have a differently populated cache
// even when its rendered appearance is identical.
func checkWalrusMaterial(t *testing.T, index int, got, want thing.Thing) {
	t.Helper()
	if got.Material == nil || want.Material == nil {
		if got.Material != want.Material {
			t.Fatalf("part %d material presence changed", index)
		}
		return
	}
	a, b := *got.Material, *want.Material
	gotTexture, wantTexture := a.Texture, b.Texture
	a.Texture, b.Texture = nil, nil
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("part %d material fields changed", index)
	}
	if gotTexture == nil || wantTexture == nil {
		if gotTexture != nil || wantTexture != nil {
			t.Fatalf("part %d texture presence changed", index)
		}
		return
	}
	check := func(p maths.Vec) {
		if a, b := gotTexture.At(p), wantTexture.At(p); a != b {
			t.Fatalf("part %d texture at %v: got %v, want %v", index, p, a, b)
		}
	}
	// Include surface positions and a three-dimensional sweep covering the
	// head, shoulders, flank, underside and tail in the walrus rest space.
	switch got.Shape.(type) {
	case *mesh.Mesh, *shape.Mesh:
		points, _, _ := testWalrusMesh(t, got)
		step := max(1, len(points)/512)
		for i := 0; i < len(points); i += step {
			check(points[i])
		}
	}
	for x := -9; x <= 9; x++ {
		for y := -2; y <= 3; y++ {
			for z := -3; z <= 3; z++ {
				check(maths.V(float64(x)*.2, float64(y)*.2, float64(z)*.2))
			}
		}
	}
}

func testWalrusMesh(t *testing.T, part thing.Thing) ([]maths.Vec, [][3]int, bool) {
	t.Helper()
	switch made := part.Shape.(type) {
	case *mesh.Mesh:
		return made.Points(), made.Faces(), made.Closed()
	case *shape.Mesh:
		return made.Vertices, made.Faces, made.Closed()
	}
	t.Fatalf("part %q is not a mesh", part.Material.Name)
	return nil, nil, false
}

// testWalrusStudy is the walrus at yaw and phase, failing the test on an
// error.
func testWalrusStudy(t *testing.T, yaw, phase float64) []thing.Thing {
	t.Helper()
	parts, err := WalrusStudy(yaw, phase)
	if err != nil {
		t.Fatal(err)
	}
	return parts
}
