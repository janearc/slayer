package assets

import (
	"strings"
	"testing"
)

func TestGuidanceUsesPinnedSchemaAndRefusesUnknownFields(t *testing.T) {
	value, err := Guidance("film", "wide")
	if err != nil || !strings.Contains(value, "above 0 and below 180") {
		t.Fatalf("schema field guidance: %q, %v", value, err)
	}
	if _, err := Guidance("film", "invented"); err == nil {
		t.Fatal("unknown field should not acquire invented guidance")
	}
}
