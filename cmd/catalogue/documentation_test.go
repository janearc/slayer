package main

import (
	"fmt"
	"strings"
	"testing"
)

// Every registered kind must have a description and notes that contain
// actual words. The menu already checks that no kind is omitted.
func TestCatalogueRefusesUndocumentedKinds(t *testing.T) {
	if _, err := readMenu(); err != nil {
		t.Fatalf("the current catalogue must document every kind: %v", err)
	}
	for _, field := range []string{"about", "notes"} {
		t.Run(field, func(t *testing.T) {
			raw := "entries:\n" + fmt.Sprintf(oneEntry, "webb", "quick")
			original := map[string]string{
				"about": "about: A specimen for the tests.",
				"notes": "notes: Authored for the tests.",
			}[field]
			raw = strings.Replace(raw, original, field+": '   '", 1)
			_, err := parseMenu("undocumented.yaml", []byte(raw))
			if err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("blank %s was accepted or not identified: %v", field, err)
			}
		})
	}
}
