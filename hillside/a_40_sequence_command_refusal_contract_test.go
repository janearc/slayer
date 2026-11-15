package hillside

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSequenceReadKeepsTheActualMissingPlanFileCause(t *testing.T) {
	control, _ := sequenceFixture(t)
	if _, err := readSequence(control); err != nil {
		t.Fatal("real valid plan control failed", err)
	}
	path := filepath.Join(t.TempDir(), "missing-plan.yaml")
	_, err := readSequence(path)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), path) {
		t.Fatalf("actual plan-open failure became a schema or decode failure: %v", err)
	}
}
