//go:build draw && darwin && arm64

package manufacture

import (
	"strings"
	"testing"
)

func TestConiferReferenceAlternatesAreNarrow(t *testing.T) {
	names := []string{"spruce", "norway", "coast-redwood", "giant-sequoia"}
	var rows []referenceRecord
	for _, line := range strings.Split(strings.TrimSpace(coniferAlternates), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		row, err := parseReferenceRecord(line)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if err := validateReferencePopulation(rows, names, referenceCounts{Cases: 32, Geometries: 32}); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if !referenceAccepted(row.name, row.seed, row.level, row.expected, "original") ||
			referenceAccepted(row.name, row.seed, row.level, strings.Repeat("0", 64), "original") ||
			referenceAccepted("unlisted", row.seed, row.level, row.expected, "original") ||
			referenceAccepted(row.name, 99, row.level, row.expected, "original") ||
			referenceAccepted(row.name, row.seed, "unlisted", row.expected, "original") {
			t.Fatalf("alternate did not constrain %s %s %s", row.name, row.seedText, row.level)
		}
	}
}
