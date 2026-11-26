//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// TestHistoricalPlantRecordMatrix preserves the directly registered specimens
// captured before plant names moved into checked profile books.
func TestHistoricalPlantRecordMatrix(t *testing.T) {
	file, err := os.Open("testdata/historical-plants-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	seen := map[string]bool{}
	for scan.Scan() {
		row, err := parseReferenceRecord(scan.Text())
		if err != nil || row.expected == "REFUSED" {
			t.Fatalf("invalid historical plant record %q: %v", scan.Text(), err)
		}
		key := row.name + " " + row.seedText + " " + row.level
		if seen[key] {
			t.Fatalf("duplicate historical plant record %s", key)
		}
		seen[key] = true
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	names, err := os.ReadFile("testdata/historical-plants-names.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range strings.Fields(string(names)) {
		for _, seed := range []string{"5", "11"} {
			for _, detail := range []string{"hull", "clumps", "quick", "demo"} {
				key := strings.Join([]string{name, seed, detail}, " ")
				if !seen[key] {
					t.Errorf("missing historical plant record %s", key)
				}
			}
		}
	}
	if len(strings.Fields(string(names))) != 12 || len(seen) != 96 {
		t.Fatalf("historical plant population: got %d names and %d records", len(strings.Fields(string(names))), len(seen))
	}
}
