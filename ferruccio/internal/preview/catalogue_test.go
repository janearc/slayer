package preview

import (
	"slices"
	"testing"
)

// TestSpecimenMenuIncludesSubaruWithoutWildcardSelectors protects single renders.
func TestSpecimenMenuIncludesSubaruWithoutWildcardSelectors(t *testing.T) {
	names, err := Specimens()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(names, "subaru") {
		t.Fatal("pinned menu lacks Subaru")
	}
	for _, name := range []string{"all", "*", "", "../subaru", "--help"} {
		if slices.Contains(names, name) {
			t.Fatalf("menu contains wildcard or unsafe name %q", name)
		}
	}
}
