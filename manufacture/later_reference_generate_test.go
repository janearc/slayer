//go:build darwin && arm64

package manufacture

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// recordedKinds is every kind some geometry reference file already holds a
// row for.
func recordedKinds(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("testdata/*-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	recorded := map[string]bool{}
	for _, file := range files {
		if strings.HasPrefix(filepath.Base(file), "monolithic") {
			continue
		}
		text, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(text), "\n") {
			if fields := strings.Fields(line); len(fields) > 0 {
				recorded[fields[0]] = true
			}
		}
	}
	return recorded
}

// TestWriteLaterReference writes the reference rows of every kind that has
// none, when LATER_REFERENCE_OUT names a file, and is skipped otherwise, as
// the trucks' are. It never writes a row for a kind that has one, so it
// cannot change a record: a record changes only by a deliberate change of
// authored geometry, said in the change.
func TestWriteLaterReference(t *testing.T) {
	path := os.Getenv("LATER_REFERENCE_OUT")
	if path == "" {
		t.Skip("set LATER_REFERENCE_OUT to write the rows")
	}
	recorded := recordedKinds(t)
	var names []string
	for _, name := range KindNames() {
		if !recorded[name] {
			names = append(names, name)
		}
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
