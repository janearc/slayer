package main

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestMacrostructureCatalogueRegistersAllFiveRecipes(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range menu.Entries {
		if !strings.HasPrefix(entry.Kind, "macro-") && !strings.HasPrefix(entry.Kind, "landform-") {
			continue
		}
		count++
		if _, err := manufacture.KindNamed(entry.Kind); err != nil {
			t.Fatal(err)
		}
		if len(entry.Details) != 3 {
			t.Fatal("macro recipe detail registration", entry)
		}
	}
	if count != 5 {
		t.Fatal("macrostructure catalogue recipe count", count)
	}
}
