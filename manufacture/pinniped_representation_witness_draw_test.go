//go:build draw && darwin && arm64

package manufacture

import "testing"

// Captured from the legacy registry before these representations were
// made profile-driven, at seed 5 and quick detail on Darwin/ARM64.
func TestPinnipedRepresentationBaselineWitness(t *testing.T) {
	book, err := ReadPinnipedProfiles("pinnipeds.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ legacy, species, representation, digest string }{
		{"harbor-seal-bones", "harbor-seal", "bones", "f47b1f0ef902735c418d353bf0cb4ec017dd41c2dac4667a6cd9708c08820d54"},
		{"walrus-fresh-5-skeleton", "walrus", "bones", "e4aaed629cf680b50b3e514cc3849d79ad2bf4ec7bd4ea197cad701cb16f046e"},
		{"walrus-fresh-5-volumes", "walrus", "volumes", "751493516dcc248eb94b8420e7bcb0a8f3ad81cae8a1d3b3c20885ae680b470e"},
	} {
		t.Run(tc.legacy, func(t *testing.T) {
			newRow, err := book.NewPinnipedRepresentation(tc.species, "", tc.representation)
			if err != nil {
				t.Fatal(err)
			}
			made, err := MakeWithComplexity(newRow, Sow(5), "quick")
			if err != nil {
				t.Fatal(err)
			}
			if got := pinnipedWitness(t, made); got != tc.digest {
				t.Fatalf("got %s, want pre-profile %s", got, tc.digest)
			}
		})
	}
}
