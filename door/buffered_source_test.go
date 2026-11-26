package door

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrozenJobUsesOriginalDirectoryForRelativeInputs(t *testing.T) {
	originalDir, frozenDir := t.TempDir(), t.TempDir()
	original := filepath.Join(originalDir, "job.yaml")
	frozen := filepath.Join(frozenDir, "snapshot.yaml")
	raw := []byte("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: scene.yaml}\nfilm: {width: 16, height: 12}\n")
	if err := os.WriteFile(frozen, raw, 0600); err != nil {
		t.Fatal(err)
	}
	wanted, err := jobOfSource(frozen, original)
	if err != nil {
		t.Fatal(err)
	}
	if path, _ := wanted.Scene().Set(); path != filepath.Join(originalDir, "scene.yaml") {
		t.Fatalf("relative set resolved to %q", path)
	}
	if wanted.File() != original {
		t.Fatalf("job source = %q, want %q", wanted.File(), original)
	}
}

func TestFrozenSetUsesBufferInsteadOfDisk(t *testing.T) {
	raw, err := os.ReadFile("../ferruccio/examples/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), "  seed: 5", "  seed: 9", 1)
	if changed == string(raw) {
		t.Fatal("fixture did not change")
	}
	original := filepath.Join(t.TempDir(), "scene.yaml")
	if err := os.WriteFile(original, raw, 0600); err != nil {
		t.Fatal(err)
	}
	frozen := filepath.Join(t.TempDir(), "snapshot.yaml")
	if err := os.WriteFile(frozen, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	wanted, err := jobOfSource(frozen, original)
	if err != nil {
		t.Fatal(err)
	}
	read, has := wanted.ImportedSet()
	if !has || read.Seed() != 9 {
		t.Fatalf("buffer was not retained: %t seed %d", has, read.Seed())
	}
}

func TestFrozenSequencePlanIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.yaml")
	if err := os.WriteFile(path, []byte("$schema: https://github.com/janearc/slayer/schemas/sequence-v1.json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := jobOfSource(path, "/original/plan.yaml"); err == nil || !strings.Contains(err.Error(), "sequence plan") {
		t.Fatalf("sequence plan accepted: %v", err)
	}
}
