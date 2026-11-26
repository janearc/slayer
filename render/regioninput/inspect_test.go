package regioninput

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectHashesNamedSnapshotFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := Inspect(root, []string{"one"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "one" || entries[0].Bytes != 5 || entries[0].SHA256 != sha256.Sum256([]byte("hello")) {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestInspectRejectsEscapeDuplicateAndSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("one", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, names := range [][]string{{"../one"}, {"one", "one"}, {"link"}} {
		if _, err := Inspect(root, names); err == nil {
			t.Fatalf("accepted %v", names)
		}
	}
}
