package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeScene writes a minimal set file below directory.
func writeScene(t *testing.T, directory, name string) {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("scene: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestLibraryShowsBothScenesThatShareAFileName checks that neither is hidden
// and that each carries enough of its path to be told apart.
func TestLibraryShowsBothScenesThatShareAFileName(t *testing.T) {
	directory := t.TempDir()
	writeScene(t, directory, "a/plain.yaml")
	writeScene(t, directory, "b/plain.yaml")
	entries := readSceneLibrary(directory)
	if len(entries) != 2 {
		t.Fatalf("library has %d entries; want both scenes: %+v", len(entries), entries)
	}
	if entries[0].label == entries[1].label {
		t.Fatalf("labels %q and %q cannot be told apart", entries[0].label, entries[1].label)
	}
	for _, entry := range entries {
		if !strings.Contains(entry.label, filepath.ToSlash(entry.name)) {
			t.Fatalf("label %q does not show its path %q", entry.label, entry.name)
		}
	}
}

// TestLibraryKeepsPlainLabelsWhenNamesDiffer is the anti-test: scenes with
// different names keep their plain labels, with no path added.
func TestLibraryKeepsPlainLabelsWhenNamesDiffer(t *testing.T) {
	directory := t.TempDir()
	writeScene(t, directory, "a/first.yaml")
	writeScene(t, directory, "b/second.yaml")
	for _, entry := range readSceneLibrary(directory) {
		if strings.Contains(entry.label, "/") {
			t.Fatalf("label %q carries a path it does not need", entry.label)
		}
	}
}
