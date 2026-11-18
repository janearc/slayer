package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// TestAnImportNeedsItsRightsDeclaration prevents unattributed scene admission.
func TestAnImportNeedsItsRightsDeclaration(t *testing.T) {
	text := strings.Replace(importedSet, "        license: CC0-1.0\n", "", 1)
	_, err := set.Read(importedPath(t), []byte(text))
	if err == nil || !strings.Contains(err.Error(), "license") {
		t.Fatalf("wanted a missing license refusal, got %v", err)
	}
}

// TestAnImportRefusesBlankRights keeps an empty label from posing as provenance.
func TestAnImportRefusesBlankRights(t *testing.T) {
	text := strings.Replace(importedSet, "license: CC0-1.0", "license: '   '", 1)
	_, err := set.Read(importedPath(t), []byte(text))
	if err == nil || !strings.Contains(err.Error(), "license") {
		t.Fatalf("wanted a blank license refusal, got %v", err)
	}
}
