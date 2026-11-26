//go:build draw && darwin && arm64

package manufacture

import "testing"

// Captured from the original registered studies before profile selection,
// at seed 407 and quick detail on Darwin/ARM64.
func TestPinnipedStudyBaselineWitness(t *testing.T) {
	book, err := ReadPinnipedProfiles("pinnipeds.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ legacy, study, digest string }{
		{"harbor-seal", "detailed-wet", "719f8882473e206ac8c2e22435df867a4fded47f5cf59420cd40cff35ac58a99"},
		{"harbor-seal-dry", "detailed-dry", "708b5b9700200b44b359c0ad5ba75b4d86b2aa0cdd50fdca3240823afb57cab9"},
		{"bay-seal", "bay", "3363f6352f2b45695ef07c22ba4eacab21f54a8465aaaa7baf701e10df7a9d93"},
	} {
		t.Run(tc.legacy, func(t *testing.T) {
			newRow, err := book.NewPinnipedStudy("harbor-seal", tc.study)
			if err != nil {
				t.Fatal(err)
			}
			made, err := MakeWithComplexity(newRow, Sow(407), "quick")
			if err != nil {
				t.Fatal(err)
			}
			if got := pinnipedWitness(t, made); got != tc.digest {
				t.Fatalf("got %s, want pre-profile %s", got, tc.digest)
			}
		})
	}
}
