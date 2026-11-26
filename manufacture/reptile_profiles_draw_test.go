//go:build draw && darwin && arm64

package manufacture

import "testing"

// Captured independently before the profile adapters were introduced.
func TestReptileProfilesRetainTheExistingStudies(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	book, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, study := range []struct {
		species, name string
		count         int
		digest        string
	}{
		{"pliosaur", "baseline", 55, "5cc6fa48e40746953b88acac1dae4e17ddc024937fe81627a3563707ebfc68dc"},
		{"pliosaur", "continuous", 19, "38817309fb37f1bc064af7a7a99c15330131ec5dd2542c46d6257b31da5134ac"},
		{"purussaurus", "study", 15, "c41c5140441f7ee8109e6cf6834a210a361bafb6b2d08c996b774d2e30934918"},
		{"hatzegopteryx", "composite", 16, "fd96b1dd609ad5cb5cd053cbf5c4a19d7f2e4c25d3290e37e7f3fc772729a85b"},
		{"hatzegopteryx", "cage", 23, "6d2209cd0865a1ab701a4ad6d87718015fe25ad3fcd7f16f518f9010f980b803"},
		{"hatzegopteryx", "density", 1, "f3025031ecc36384365f0e5de82830f5d9dc68b4dda8a223077187d7f3cc6b39"},
	} {
		t.Run(study.species+"-"+study.name, func(t *testing.T) {
			row, err := book.NewReptile(study.species, study.name)
			if err != nil {
				t.Fatal(err)
			}
			made, err := Make(row, Sow(1))
			if err != nil {
				t.Fatal(err)
			}
			parts := made.native
			if got := bioformMesherDigest(parts); len(parts) != study.count || got != study.digest {
				t.Fatalf("%d parts and %s", len(parts), got)
			}
		})
	}
}
