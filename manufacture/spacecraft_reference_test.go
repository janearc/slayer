//go:build darwin && arm64

package manufacture

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestSpacecraftAppearance holds only the three spacecraft families to
// their authored records. Explicit capture writes a separate file for review.
func TestSpacecraftAppearance(t *testing.T) {
	var rows strings.Builder
	for _, name := range []string{"hubble", "space-observatory", "spitzer", "webb"} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, seed := range []uint64{5, 11} {
			for _, detail := range Complexities() {
				made, err := MakeWithComplexity(row, Sow(seed), detail)
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(&rows, "%s %d %s %s\n", name, seed,
					detail, referenceDigest(t, made))
			}
		}
	}
	if out := os.Getenv("SPACECRAFT_REFERENCE_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(rows.String()), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile("testdata/spacecraft-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != rows.String() {
		t.Fatal("spacecraft appearance differs from reviewed record")
	}
}
