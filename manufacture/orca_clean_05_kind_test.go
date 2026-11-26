package manufacture

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestOrcaKindPreservesApprovedMeshAndPlacement(t *testing.T) {
	kind, err := defaultWhaleProfiles.NewOrca("swim", "")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(17), "quick")
	if err != nil {
		t.Fatal(err)
	}
	at := maths.NewSubspace(maths.V(4, 5, 6), maths.Identity, 1)
	got, err := made.Things(at)
	if err != nil {
		t.Fatal(err)
	}
	want, err := orcaSwimming(0, maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || made.PartCount() != 1 || got[0].At(0) != at {
		t.Fatalf("wrong manufactured placement or count: %+v", got)
	}
	a, aok := got[0].Shape.(*mesh.Mesh)
	b, bok := want[0].Shape.(*mesh.Mesh)
	if !aok || !bok || !reflect.DeepEqual(a.Points(), b.Points()) ||
		!reflect.DeepEqual(a.Faces(), b.Faces()) || a.Bounds() != made.Bounds() {
		t.Fatal("manufacture did not retain the approved indexed skin")
	}
	for _, p := range []maths.Vec{maths.V(1.55, .16, .5), maths.V(0, -.5, 0), maths.V(0, 0, .5)} {
		if got[0].Material.Texture.At(p) != want[0].Material.Texture.At(p) {
			t.Fatalf("paint differs at %v", p)
		}
	}
	if _, err := made.Things(maths.Subspace{Origin: maths.V(0, 0, 0), Scale: -1}); err == nil {
		t.Fatal("accepted invalid placement")
	}
	combined, err := gather(made, made)
	if err != nil {
		t.Fatal(err)
	}
	combined, err = combined.placed(at)
	if err != nil {
		t.Fatal(err)
	}
	placed, err := combined.Things(maths.Local())
	if err != nil || len(placed) != 2 || combined.PartCount() != 2 ||
		placed[0].At(0) != at || placed[1].At(0) != at {
		t.Fatalf("gathered native surfaces lost their placement: %v, %v", placed, err)
	}
}
