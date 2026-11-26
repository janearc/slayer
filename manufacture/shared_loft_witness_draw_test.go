//go:build draw && darwin && arm64

package manufacture

import "testing"

// The Alfa and coral exercise the same loft and crease shading as the
// animal studies; these record the pre-extraction geometry on Darwin/ARM64.
func TestSharedLoftKeepsPreExtractionGeometry(t *testing.T) {
	alfa, err := KindNamed(alfaKind)
	if err != nil {
		t.Fatal(err)
	}
	coral, err := Coral(DefaultCoral())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, digest string
		kind         Kind
		seed         uint64
	}{
		{"alfa", "b1f2ecf88428ea2baba545d6adb71a9c3f7b7a089ed354505c71ed2e9f0ff10a", alfa, 3},
		{"coral", "cb6daf5ca985a99f08bf6aac44a1ca5e99c98f5310b5c47bea64c713200d46d4", coral, 14},
	} {
		t.Run(tc.name, func(t *testing.T) {
			made, err := MakeWithComplexity(tc.kind, Sow(tc.seed), "quick")
			if err != nil {
				t.Fatal(err)
			}
			if got := referenceDigest(t, made); got != tc.digest {
				t.Fatalf("got %s, want pre-extraction %s", got, tc.digest)
			}
		})
	}
}
