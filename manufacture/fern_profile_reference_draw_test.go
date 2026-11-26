//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestFernProfileGeometryReference checks the two existing fern quick
// references independently of the whole-catalogue population check.
func TestFernProfileGeometryReference(t *testing.T) {
	file, err := os.Open("testdata/geometry-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	seen := map[string]bool{}
	for scan.Scan() {
		row := strings.Fields(scan.Text())
		if len(row) < 4 || row[0] != "fern" || row[2] != "quick" {
			continue
		}
		if seen[row[1]] {
			t.Fatalf("duplicate fern seed %s", row[1])
		}
		seen[row[1]] = true
		n, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(row[1], func(t *testing.T) { referenceSpecimen(t, row[0], n, row[2], row[3], "") })
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || !seen["5"] || !seen["11"] {
		t.Fatalf("fern reference seeds: %v", seen)
	}
}
