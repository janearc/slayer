package preview

import "testing"

func TestCatalogueProfilesExposeCheckedFishAndPlants(t *testing.T) {
	profiles, err := CatalogueProfiles()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, profile := range profiles {
		if profile.Name == "pacific-bluefin-tuna" && profile.Kind == "animal" && profile.Group == "Fish" && profile.About != "" && profile.Spec != nil {
			seen["fish"] = true
		}
		if profile.Name == "spruce" && profile.Kind == "plant" && profile.About != "" && profile.Spec != nil {
			seen["plant"] = true
		}
	}
	if !seen["fish"] || !seen["plant"] {
		t.Fatalf("missing checked catalogue profiles: %v", seen)
	}
}

func TestCatalogueProfilesReturnsIndependentSpecs(t *testing.T) {
	first, err := CatalogueProfiles()
	if err != nil || len(first) == 0 {
		t.Fatalf("first profiles: %v", err)
	}
	original := first[0].Spec.Content[1].Value
	first[0].Spec.Content[1].Value = "changed"
	second, err := CatalogueProfiles()
	if err != nil || second[0].Spec.Content[1].Value != original {
		t.Fatal("a caller changed the shared catalogue")
	}
}
