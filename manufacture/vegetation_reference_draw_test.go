//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestValleyOakGeometryReference checks the eight existing recorded cases
// without changing the separate whole-catalogue population gate.
func TestValleyOakGeometryReference(t *testing.T) {
	file, err := os.Open("testdata/vegetation-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	seen := make(map[string]bool)
	for scan.Scan() {
		row := strings.Fields(scan.Text())
		if len(row) == 0 || row[0] != "valley-oak" {
			continue
		}
		if len(row) != 4 {
			t.Fatalf("invalid valley oak reference: %q", scan.Text())
		}
		key := row[1] + "_" + row[2]
		if seen[key] {
			t.Fatalf("duplicate valley oak reference: %s", key)
		}
		seen[key] = true
		number, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) { referenceSpecimen(t, row[0], number, row[2], row[3], "") })
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	for _, seed := range []string{"5", "11"} {
		for _, detail := range []string{"hull", "clumps", "quick", "demo"} {
			if !seen[seed+"_"+detail] {
				t.Fatalf("missing valley oak reference: %s_%s", seed, detail)
			}
		}
	}
	if len(seen) != 8 {
		t.Fatalf("got %d valley oak references, want 8", len(seen))
	}
}
