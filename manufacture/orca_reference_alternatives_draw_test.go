//go:build draw && darwin && arm64

package manufacture

import (
	"os"
	"strings"
	"testing"
)

func TestOrcaReferenceAlternatesAreNarrow(t *testing.T) {
	names := []string{"gargoyle-orca-concrete", "gargoyle-orca-granite", "gargoyle-cornice-concrete", "gargoyle-cornice-granite"}
	var rows []referenceRecord
	for _, line := range strings.Split(strings.TrimSpace(orcaAlternates), "\n") {
		row, err := parseReferenceRecord(line)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if err := validateReferencePopulation(rows, names, referenceCounts{Cases: 32, Geometries: 32}); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile("testdata/historical-public-partial.txt")
	if err != nil {
		t.Fatal(err)
	}
	originals := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(original)), "\n") {
		row, err := parseReferenceRecord(line)
		if err != nil {
			t.Fatal(err)
		}
		originals[row.name+" "+row.seedText+" "+row.level] = row.expected
	}
	for _, row := range rows {
		key := row.name + " " + row.seedText + " " + row.level
		old := originals[key]
		if old == "" || old == row.expected ||
			!referenceAccepted(row.name, row.seed, row.level, old, old) ||
			!referenceAccepted(row.name, row.seed, row.level, row.expected, old) ||
			referenceAccepted(row.name, row.seed, row.level, strings.Repeat("0", 64), old) ||
			referenceAccepted("unlisted", row.seed, row.level, row.expected, old) ||
			referenceAccepted(row.name, 99, row.level, row.expected, old) ||
			referenceAccepted(row.name, row.seed, "unlisted", row.expected, old) {
			t.Fatalf("alternate did not preserve and constrain %s", key)
		}
	}
}
