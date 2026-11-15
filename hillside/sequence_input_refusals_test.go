package hillside

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSequenceFolderDoesNotClaimProgressItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0700); err != nil {
			t.Error(err)
		}
	})
	_, err := sequenceFolder(dir, sequenceState{Complete: map[int]sequenceEntry{}}, true)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("read-only output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sequence.json")); !os.IsNotExist(err) {
		t.Fatalf("progress written after refusal: %v", err)
	}
}
