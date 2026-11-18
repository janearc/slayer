package main

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

// The reusable families and full ten-thousand-building city are discoverable
// through the existing catalogue, with only their supported representations.
func TestCityCatalogueEntriesResolveToRegisteredRecipes(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range menu.Entries {
		if !strings.HasPrefix(entry.Kind, "city-") {
			continue
		}
		count++
		if _, err := manufacture.KindNamed(entry.Kind); err != nil {
			t.Fatal(err)
		}
		if len(entry.Details) != 2 || entry.Details[0] != "hull" || entry.Details[1] != "quick" {
			t.Fatalf("unsupported city catalogue detail: %v", entry.Details)
		}
	}
	if count != 7 {
		t.Fatalf("city catalogue entries: %d", count)
	}
}
