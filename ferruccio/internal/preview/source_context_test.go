package preview

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestCopySourceManifestPreservesNearestBytesAndRefusesOverwrite ensures relocated
// scenes retain their input context without changing a preexisting destination.
func TestCopySourceManifestPreservesNearestBytesAndRefusesOverwrite(t *testing.T) {
	original := t.TempDir()
	near := filepath.Join(original, "scenes")
	if err := os.Mkdir(near, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("# retain exact authored metadata\r\nversion: 1\r\nsources: []\r\n")
	for path, data := range map[string][]byte{
		filepath.Join(original, "sources.yaml"): []byte("outer must not win\n"),
		filepath.Join(near, "sources.yaml"):     manifest,
		filepath.Join(near, "scene.yaml"):       []byte("scene: {}\n"),
		filepath.Join(near, "dataset.bin"):      []byte("dataset must stay here"),
	} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	destination := t.TempDir()
	source := filepath.Join(near, "scene.yaml")
	for attempt := 0; attempt < 2; attempt++ {
		if err := CopySourceManifest(source, destination); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(destination, "sources.yaml")
	got, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(got, manifest) {
		t.Fatalf("manifest bytes changed: %q, %v", got, err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 1 {
		t.Fatalf("copied more than metadata: %v, %v", entries, err)
	}
	changed := []byte("existing private context\n")
	if err := os.WriteFile(output, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := CopySourceManifest(source, destination); err == nil {
		t.Fatal("overwrote different destination manifest")
	}
	got, err = os.ReadFile(output)
	if err != nil || !bytes.Equal(got, changed) {
		t.Fatalf("existing manifest changed: %q, %v", got, err)
	}
}

// TestCopySourceManifestRejectsOversizedAndSymlinkMetadata bounds private copies.
func TestCopySourceManifestRejectsOversizedAndSymlinkMetadata(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		original, destination := t.TempDir(), t.TempDir()
		manifest := filepath.Join(original, "sources.yaml")
		if oversized {
			if err := os.WriteFile(manifest, make([]byte, sourceManifestLimit+1), 0600); err != nil {
				t.Fatal(err)
			}
		} else {
			target := filepath.Join(original, "metadata.yaml")
			if err := os.WriteFile(target, []byte("sources: []\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, manifest); err != nil {
				t.Fatal(err)
			}
		}
		if err := CopySourceManifest(filepath.Join(original, "scene.yaml"), destination); err == nil {
			t.Fatalf("accepted unsupported manifest, oversized=%v", oversized)
		}
		entries, err := os.ReadDir(destination)
		if err != nil || len(entries) != 0 {
			t.Fatalf("refused manifest left artifacts: %v %v", entries, err)
		}
	}
}
