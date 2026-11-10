package publication

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestLocalCreateFollowsAStableLinkedParent needs real directory entries
// to establish the lexical name and physical target of an existing link.
func TestLocalCreateFollowsAStableLinkedParent(t *testing.T) {
	target, parent, fallback := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("TMPDIR", fallback)
	linked := filepath.Join(parent, "linked")
	if err := os.Symlink(target, linked); err != nil {
		t.Fatal(err)
	}
	opened := createdNative(t, tempSpec{parent: linked, pattern: ".linked-*"})
	if opened.err != nil || opened.file == nil || filepath.Dir(opened.name) != linked {
		t.Fatalf("linked create: %+v", opened)
	}
	file := nativeCreatedFile(t, opened.file)
	if file.Name() != opened.name {
		t.Fatal("reported name differs from the native handle's name")
	}
	held, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	basename := filepath.Base(opened.name)
	physicalPath := filepath.Join(target, basename)
	named, err := os.Stat(physicalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(held, named) {
		t.Fatal("linked parent did not create in its target")
	}
	assertNativeDirectory(t, fallback)
}

// TestLocalCreateLeavesADanglingParentUnresolved is the paired anti-test.
// A broken link cannot create its target.
// A broken link cannot fall back to another directory.
func TestLocalCreateLeavesADanglingParentUnresolved(t *testing.T) {
	parent, fallback := t.TempDir(), t.TempDir()
	t.Setenv("TMPDIR", fallback)
	target, linked := filepath.Join(parent, "absent"), filepath.Join(parent, "linked")
	if err := os.Symlink(target, linked); err != nil {
		t.Fatal(err)
	}
	opened := createdNative(t, tempSpec{parent: linked, pattern: ".linked-*"})
	if !errors.Is(opened.err, os.ErrNotExist) || opened.file != nil || opened.name != "" {
		t.Fatalf("dangling create: %+v, want error only", opened)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing target changed: %v", err)
	}
	still, err := os.Readlink(linked)
	if err != nil || still != target {
		t.Fatalf("parent link changed: %q, %v", still, err)
	}
	assertNativeDirectory(t, fallback)
}
