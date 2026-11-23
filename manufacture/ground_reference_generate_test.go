//go:build darwin && arm64

package manufacture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteGroundReference captures only the three ground instrument records.
// Set GROUND_REFERENCE_OUT to a scratch directory after independent checks;
// review the changed rows before replacing any committed record.
func TestWriteGroundReference(t *testing.T) {
	destination := os.Getenv("GROUND_REFERENCE_OUT")
	if destination == "" {
		t.Skip("set GROUND_REFERENCE_OUT to capture ground instrument records")
	}
	families := map[string][]string{
		"ground": {"palomar", "keck-i", "keck-ii", "subaru"},
		"survey": {"rubin"}, "large": {"elt"},
	}
	for family, names := range families {
		var out strings.Builder
		for _, name := range names {
			kind, err := KindNamed(name)
			if err != nil {
				t.Fatal(err)
			}
			for _, seed := range []uint64{5, 11} {
				for _, level := range Complexities() {
					made, err := MakeWithComplexity(kind, Sow(seed), level)
					if err != nil {
						t.Fatal(err)
					}
					fmt.Fprintf(&out, "%s %d %s %s\n", name, seed, level,
						referenceDigest(t, made))
				}
			}
		}
		path := filepath.Join(destination, family+"-reference-arm64.txt")
		text := out.String()
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
