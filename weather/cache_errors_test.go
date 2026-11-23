package weather

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckpointRefusesTruncatedWrongAndMismatchedFiles(t *testing.T) {
	made := small(t, 5)
	dir := t.TempDir()
	if err := made.save(dir); err != nil {
		t.Fatal(err)
	}
	originalPath := filepath.Join(dir, name(0))
	original, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	wrong, padding := bytes.Clone(original), strings.Repeat("x", len(magic))
	copy(wrong, padding)
	count := bytes.Clone(original)
	count[len(magic)+16] ^= 1
	for _, test := range []struct {
		name string
		data []byte
	}{{"short-head", original[:2]}, {"wrong-magic", wrong}, {"wrong-grid", count}, {"short-body", original[:len(original)-1]}} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "checkpoint")
			if err := os.WriteFile(path, test.data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := made.load(path); err == nil {
				t.Fatal("invalid checkpoint accepted")
			}
			if made.Clock() != 0 {
				t.Fatal("failed load changed clock")
			}
		})
	}
	if err := made.load(filepath.Join(dir, "absent")); err == nil {
		t.Fatal("missing checkpoint")
	}
	if err := made.resume(filepath.Join(dir, "absent"), 1); err == nil {
		t.Fatal("missing shelf")
	}
	if err := made.save(filepath.Join(dir, "absent")); err == nil {
		t.Fatal("save to missing shelf")
	}
	folderPath := filepath.Join(dir, name(3600))
	if err := os.Mkdir(folderPath, 0700); err != nil {
		t.Fatal(err)
	}
	made.Run(3600)
	if err := made.save(dir); err == nil {
		t.Fatal("checkpoint silently overwrote directory")
	}
}

func TestReachReportsAReferencedBrokenCheckpointAndBlockedShelf(t *testing.T) {
	made := small(t, 5)
	dir := t.TempDir()
	shelf := filepath.Join(dir, made.key(false))
	if err := os.MkdirAll(shelf, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note(shelf, 10), []byte("absent.wx"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := made.Reach(dir, 10, 10, false); err == nil {
		t.Fatal("missing referenced checkpoint")
	}
	brokenPath := filepath.Join(shelf, name(20))
	if err := os.WriteFile(brokenPath, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := made.Reach(dir, 30, 10, false); err == nil {
		t.Fatal("broken resumed checkpoint")
	}
	file := filepath.Join(t.TempDir(), "held")
	if err := os.WriteFile(file, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := made.Reach(file, 10, 10, false); err == nil {
		t.Fatal("blocked cache root")
	}
}
