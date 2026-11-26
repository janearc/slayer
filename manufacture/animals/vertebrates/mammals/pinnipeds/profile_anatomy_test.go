package pinnipeds

import (
	"bytes"
	"testing"
)

func testHarborLoft(book *Book) (Loft, error) {
	p, _ := book.Get("harbor-seal")
	return NewLoft(p, "hauled")
}
func testMonkLoft(book *Book) (Loft, error) {
	p, _ := book.Get("hawaiian-monk-seal")
	return NewLoft(p, "hauled")
}
func TestPinnipedProfilesDriveSealSkeletons(t *testing.T) {
	book, err := Read("pinnipeds.yaml", profileRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		thoracic, ribs int
	}{
		{"harbor-seal", 15, 30}, {"hawaiian-monk-seal", 15, 30},
	} {
		profile, found := book.Get(tc.name)
		if !found {
			t.Fatal("missing seal profile")
		}
		bones, err := newSealBones(profile)
		if err != nil {
			t.Fatal(err)
		}
		if bones.bones.Count("thoracic") != tc.thoracic || bones.bones.Count("rib-") != tc.ribs {
			t.Fatalf("%s: wrong skeleton counts", tc.name)
		}
	}
}

func TestPinnipedExternalProfileChangesOnlyItsOwnSkeleton(t *testing.T) {
	raw := bytes.Replace(profileRaw, []byte("thoracic: 15"), []byte("thoracic: 16"), 1)
	raw = bytes.Replace(raw, []byte(`thoracic: {basis: read, source: "Pierce, Clack and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755", value: 15}`), []byte(`thoracic: {basis: picked}`), 1)
	book, err := Read("changed.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := testHarborLoft(book)
	if err != nil {
		t.Fatal(err)
	}
	row := changed
	bones, err := newSealBones(row.form.profile)
	if err != nil {
		t.Fatal(err)
	}
	if got := bones.bones.Count("thoracic"); got != 16 {
		t.Fatalf("changed profile made %d thoracics", got)
	}
	monk, err := testMonkLoft(book)
	if err != nil {
		t.Fatal(err)
	}
	if monk.form.profile.Thoracic != 15 || harborSealForm.profile.Thoracic != 15 {
		t.Fatal("changed profile escaped its book")
	}
}

func TestPinnipedOutlineFollowsExternalProfile(t *testing.T) {
	raw := bytes.Replace(profileRaw, []byte("[.7500, .0690, .0790, .067]"), []byte("[.7500, .0720, .0850, .067]"), 1)
	if bytes.Equal(raw, profileRaw) {
		t.Fatal("head station fixture not found")
	}
	book, err := Read("changed.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := testHarborLoft(book)
	if err != nil {
		t.Fatal(err)
	}
	original, err := testHarborLoft(Default())
	if err != nil {
		t.Fatal(err)
	}
	row := changed
	old := original
	if row.form.outline.at(.75) == old.form.outline.at(.75) {
		t.Fatal("profile did not change the head")
	}
	bones, err := newSealBones(row.form.profile)
	if err != nil {
		t.Fatal(err)
	}
	held := sealHeldTrunk(bones, row.form.outline)
	if held.surface.open() != 0 {
		t.Fatal("changed outline did not make a closed trunk")
	}
	for i, influences := range held.holds {
		weight := 0.0
		for _, influence := range influences {
			weight += influence.Weight()
		}
		if !(weight > 0) {
			t.Fatalf("changed trunk point %d has no skeletal influence", i)
		}
	}
	a, err := original.Build(5)
	if err != nil {
		t.Fatal(err)
	}
	b, err := changed.Build(5)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"seal eye", "seal whisker"} {
		found := false
		for i, piece := range a.Parts {
			if piece.Material != nil && piece.Material.Name == name {
				if i >= len(b.Parts) || b.Parts[i].Material == nil || b.Parts[i].Material.Name != name {
					t.Fatalf("missing %s in changed seal", name)
				}
				if piece.At(0).Origin == b.Parts[i].At(0).Origin {
					t.Errorf("%s did not follow changed head", name)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %s in original seal", name)
		}
	}
}
