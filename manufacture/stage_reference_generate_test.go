//go:build darwin && arm64

package manufacture

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestWriteStageReference writes the stage lights' reference rows when
// STAGE_REFERENCE_OUT names a file, and is skipped otherwise. The rows
// are an authored record: regenerate them only when the stage lights'
// authored geometry deliberately changes, and say so in the change.
func TestWriteStageReference(t *testing.T) {
	path := os.Getenv("STAGE_REFERENCE_OUT")
	if path == "" {
		t.Skip("set STAGE_REFERENCE_OUT to write the rows")
	}
	var names []string
	for name := range stageRows() {
		names = append(names, name)
	}
	sort.Strings(names)
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
					fmt.Fprintf(&out, "%s %d %s REFUSED %s\n", name,
						seed, level, err.Error())
					continue
				}
				fmt.Fprintf(&out, "%s %d %s %s\n", name, seed, level,
					referenceDigest(t, made))
			}
		}
	}
	if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
