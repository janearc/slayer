//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Check the original seeded fire geometry without depending on the unrelated
// historical catalogue population count.
func TestRingFireGeometryReference(t *testing.T) {
	file, err := os.Open("testdata/geometry-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	seen := map[string]bool{}
	for scan.Scan() {
		row := strings.Fields(scan.Text())
		if len(row) < 4 || (row[0] != "campfire" && row[0] != "bonfire") || row[2] != "quick" {
			continue
		}
		key := row[0] + "/" + row[1]
		if seen[key] {
			t.Fatalf("duplicate fire geometry %s", key)
		}
		seen[key] = true
		n, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) { referenceSpecimen(t, row[0], n, row[2], row[3], "") })
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"campfire", "bonfire"} {
		for _, seed := range []string{"5", "11"} {
			if !seen[name+"/"+seed] {
				t.Fatalf("missing reference %s/%s", name, seed)
			}
		}
	}
	if len(seen) != 4 {
		t.Fatalf("unexpected fire references: %v", seen)
	}
}
