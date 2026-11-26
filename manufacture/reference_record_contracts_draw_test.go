//go:build draw && darwin && arm64

package manufacture

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestReferenceParserPreservesAndRequiresTheFullRefusal(t *testing.T) {
	row, err := parseReferenceRecord("example 5 quick REFUSED no  mesh remains")
	if err != nil || row.expected != "REFUSED" || row.reason != "no  mesh remains" {
		t.Fatalf("refusal: %+v, %v", row, err)
	}
	for _, bad := range []string{
		"example 5 quick REFUSED", "example 5 quick REFUSED ",
		"example 7 quick " + strings.Repeat("a", 64),
		"example 5 unknown " + strings.Repeat("a", 64),
		"example 5 quick xyz", "example  5 quick " + strings.Repeat("a", 64),
	} {
		if _, err := parseReferenceRecord(bad); err == nil {
			t.Errorf("accepted invalid record %q", bad)
		}
	}
}

func TestReferencePopulationRejectsMissingDuplicateAndUnknownCases(t *testing.T) {
	var names []string
	var rows []referenceRecord
	for i := range 197 {
		name := fmt.Sprintf("kind-%d", i)
		names = append(names, name)
		for _, seed := range []string{"5", "11"} {
			for _, level := range []string{"hull", "clumps", "quick", "demo"} {
				rows = append(rows, referenceRecord{name: name, seedText: seed, level: level, expected: strings.Repeat("a", 64)})
			}
		}
	}
	counts := referenceCounts{Cases: 1576, Geometries: 1576}
	if err := validateReferencePopulation(rows, names, counts); err != nil {
		t.Fatal(err)
	}
	// The validator follows the supplied catalogue as future kinds are added.
	if err := validateReferencePopulation(rows[:8], names[:1], referenceCounts{Cases: 8, Geometries: 8}); err != nil {
		t.Fatalf("valid smaller catalogue: %v", err)
	}
	duplicate := slices.Clone(rows)
	duplicate[0] = duplicate[1]
	unknown := slices.Clone(rows)
	unknown[0].name = "not-in-catalogue"
	missing := slices.Clone(rows[:len(rows)-1])
	for label, candidate := range map[string][]referenceRecord{"duplicate": duplicate, "unknown": unknown, "missing": missing} {
		if err := validateReferencePopulation(candidate, names, counts); err == nil {
			t.Errorf("accepted %s case", label)
		}
	}
	duplicateNames := slices.Clone(names)
	duplicateNames[0] = duplicateNames[1]
	if err := validateReferencePopulation(rows, duplicateNames, counts); err == nil {
		t.Fatal("accepted a duplicate catalogue name")
	}
}
