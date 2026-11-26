package manufacture

import "testing"

func TestSealBonesKindMakesOneClosedSolid(t *testing.T) {
	kind, err := referencePinnipedKind("harbor-seal-bones")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(407), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if made.PartCount() != 1 {
		t.Fatalf("%d parts, want the skeleton as one", made.PartCount())
	}
}

func TestSealBonesKindNeedsASeed(t *testing.T) {
	if _, err := (sealKind{form: sealBones}).made(seed{}); err == nil {
		t.Fatal("an unsown seed made a skeleton")
	}
}
