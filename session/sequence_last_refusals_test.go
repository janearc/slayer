package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSequenceCannotCreateAnOutputUnderAReadOnlyParent(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(parent, 0700); err != nil {
			t.Error(err)
		}
	})
	output := filepath.Join(parent, "frames")
	_, err := sequenceFolder(output, sequenceState{Complete: map[int]sequenceEntry{}}, false)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("uncreatable output: %v", err)
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output appeared after refusal: %v", err)
	}
}
