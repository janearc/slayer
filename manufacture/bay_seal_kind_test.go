package manufacture

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestBaySealKindOriginalSkinAndPlacement(t *testing.T) {
	kind, err := KindNamed("bay-seal")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(407), "quick")
	if err != nil {
		t.Fatal(err)
	}
	at := maths.NewSubspace(maths.V(4, 5, 6), maths.Identity, 1)
	got, err := made.Things(at)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 33 || made.PartCount() != 33 || got[0].At(0) != at {
		t.Fatalf("wrong parts or placement: %d", len(got))
	}
	rig := NewSealRig()
	want, err := rig.Skin(SealRigPose{Yaw: .18, Pitch: -.20, HeadPitch: -.86, Fore: [2]float64{-.26, -.20}})
	if err != nil {
		t.Fatal(err)
	}
	skin, ok := got[0].Shape.(*mesh.Mesh)
	if !ok || !reflect.DeepEqual(skin.Points(), want.Points()) || !reflect.DeepEqual(skin.Faces(), want.Faces()) {
		t.Fatal("original posed continuous skin changed")
	}
	coat := got[0].Material
	if coat.Name != "wet dark harbor seal" || coat.Colour != maths.RGB(.068, .063, .055) || coat.Gloss != .42 || coat.Satin != .33 || coat.Index != 1.43 || coat.Texture == nil {
		t.Fatal("original wet coat changed")
	}
	// Check retained paint at separated body points.
	original := baySealWetPaint{spots: HarborSealCoat(407)}
	for _, p := range []maths.Vec{maths.V(0, .2, 0), maths.V(-.4, .1, .12), maths.V(.81, .059, .02)} {
		if coat.Texture.At(p) != original.At(p) {
			t.Fatalf("paint differs at %v", p)
		}
	}
	if got[1].Material.Name != "harbor seal eye" || got[17].Material.Name != "harbor seal eye" {
		t.Fatal("corneas not present")
	}
	if got[2].Material.Name != "wet muzzle whiskers" {
		t.Fatal("whiskers not present")
	}
	if _, err := made.Things(maths.Subspace{Scale: -1}); err == nil {
		t.Fatal("accepted invalid placement")
	}
	together, err := gather(made, made)
	if err != nil {
		t.Fatal(err)
	}
	together, err = together.placed(at)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := together.Things(maths.Local())
	if err != nil || len(parts) != 66 || parts[0].At(0) != at || parts[33].At(0) != at {
		t.Fatalf("assembly placement: %v", err)
	}
}

func TestBaySealExtentContainsAllStaticPieces(t *testing.T) {
	kind, err := KindNamed("bay-seal")
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
	extent := made.Bounds()
	for i, p := range parts {
		b := transformed(p.Shape.Bounds(), p.At(0))
		if b.Min.X < extent.Min.X || b.Min.Y < extent.Min.Y || b.Min.Z < extent.Min.Z || b.Max.X > extent.Max.X || b.Max.Y > extent.Max.Y || b.Max.Z > extent.Max.Z {
			t.Fatalf("part %d lies outside static placement extent: %+v versus %+v", i, b, extent)
		}
	}
}
