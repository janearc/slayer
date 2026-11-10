package publication

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestLocalRemoveErasesOnlyTheNamedEntry needs real entries to test
// removal and sibling survival. No recursive operation or flush is requested.
func TestLocalRemoveErasesOnlyTheNamedEntry(t *testing.T) {
	parent := t.TempDir()
	temporary := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "temporary", body: []byte("owned"),
	})
	if err := temporary.Close(); err != nil {
		t.Fatal(err)
	}
	sibling := nativeSibling(t, parent)
	if err := (localFilesystem{}).Remove(pathSpec{path: temporary.Name()}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(temporary.Name()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary name remains: %v", err)
	}
	assertNativeSibling(t, sibling)
}

// assertNativeRemoveError retains the operation's error and exact path.
func assertNativeRemoveError(t *testing.T, request pathSpec, err error) {
	t.Helper()
	var refusal *os.PathError
	if !errors.As(err, &refusal) || refusal.Op != "remove" || refusal.Path != request.path {
		t.Fatalf("remove error: %v, want native error for %q", err, request.path)
	}
}

// TestLocalRemoveRefusesMissingAndNonemptyNames is the anti-test.
// Absence cannot succeed.
// A directory's child cannot be recursively erased.
func TestLocalRemoveRefusesMissingAndNonemptyNames(t *testing.T) {
	t.Run("missing", testNativeRemoveMissing)
	t.Run("nonempty directory", testNativeRemoveNonemptyDirectory)
}

// testNativeRemoveMissing preserves both the cause and an unrelated entry.
func testNativeRemoveMissing(t *testing.T) {
	parent := t.TempDir()
	sibling := nativeSibling(t, parent)
	request := pathSpec{path: filepath.Join(parent, "absent")}
	err := (localFilesystem{}).Remove(request)
	assertNativeRemoveError(t, request, err)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing remove: %v", err)
	}
	assertNativeSibling(t, sibling)
}

// testNativeRemoveNonemptyDirectory catches recursive cleanup substitutions.
func testNativeRemoveNonemptyDirectory(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "directory")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	sibling := nativeSibling(t, directory)
	identity, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	request := pathSpec{path: directory}
	assertNativeRemoveError(t, request, (localFilesystem{}).Remove(request))
	assertNativeSameEntry(t, directory, identity)
	assertNativeSibling(t, sibling)
}
