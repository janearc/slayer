package publication

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// nativeFixtureSpec names a tiny, explicitly owned fixture and its bytes.
type nativeFixtureSpec struct {
	parent string
	name   string
	body   []byte
}

// namedNativeFixture retains its creation handle for data inspection even
// when umask would prevent reopening. Cleanup also tolerates an earlier close.
func namedNativeFixture(t *testing.T, request nativeFixtureSpec) *os.File {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(request.parent, request.name),
		os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	owner := nativeHandleCleanup{reporter: t, close: file.Close}
	t.Cleanup(owner.finishIfOpen)
	if _, err := file.Write(request.body); err != nil {
		t.Fatal(err)
	}
	assertNativeHandleBytes(t, file, request.body)
	return file
}

// nativeFileIdentity captures an owned inode before the operation changes names.
func nativeFileIdentity(t *testing.T, file *os.File) os.FileInfo {
	t.Helper()
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	return info
}

// assertNativeSameEntry compares an entry with a captured identity, not its name.
func assertNativeSameEntry(t *testing.T, name string, want os.FileInfo) {
	t.Helper()
	got, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(got, want) {
		t.Fatalf("%s names another inode", name)
	}
}

// TestLocalRenameReplacesTheEntryAndRetainsReaders needs actual filesystem
// entries to observe replacement and an old reader. No flush or live paths.
func TestLocalRenameReplacesTheEntryAndRetainsReaders(t *testing.T) {
	examples := payloadExamples(t)
	parent := t.TempDir()
	source := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "source", body: examples.RenameSource,
	})
	identity := nativeFileIdentity(t, source)
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	destination := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "destination", body: []byte("before"),
	})
	sibling := nativeSibling(t, parent)
	request := renameSpec{from: source.Name(), to: destination.Name()}
	if err := (localFilesystem{}).Rename(request); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(request.from); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source name remains: %v", err)
	}
	assertNativeSameEntry(t, request.to, identity)
	assertNativeHandleBytes(t, destination, []byte("before"))
	assertNativeSibling(t, sibling)
}

// assertNativeRenameError retains the native error and both exact path names.
func assertNativeRenameError(t *testing.T, request renameSpec, err error) {
	t.Helper()
	var refusal *os.LinkError
	if !errors.As(err, &refusal) || refusal.Op != "rename" ||
		refusal.Old != request.from || refusal.New != request.to {
		t.Fatalf("rename error: %v, want native error for %+v", err, request)
	}
}

// TestLocalRenameRefusalsKeepTheirFixtures is the anti-test. These refusals
// happen before any change. Other rename errors may still have uncertain
// effects.
func TestLocalRenameRefusalsKeepTheirFixtures(t *testing.T) {
	t.Run("missing source", testNativeRenameMissingSource)
	t.Run("directory destination", testNativeRenameDirectoryDestination)
}

// testNativeRenameMissingSource keeps an existing destination and sibling.
func testNativeRenameMissingSource(t *testing.T) {
	parent := t.TempDir()
	destination := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "destination", body: []byte("before"),
	})
	identity := nativeFileIdentity(t, destination)
	sibling := nativeSibling(t, parent)
	request := renameSpec{from: filepath.Join(parent, "absent"), to: destination.Name()}
	err := (localFilesystem{}).Rename(request)
	assertNativeRenameError(t, request, err)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing source error: %v", err)
	}
	assertNativeSameEntry(t, request.to, identity)
	assertNativeHandleBytes(t, destination, []byte("before"))
	assertNativeSibling(t, sibling)
}

// testNativeRenameDirectoryDestination never replaces or recursively clears it.
func testNativeRenameDirectoryDestination(t *testing.T) {
	parent := t.TempDir()
	source := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "source", body: []byte("new"),
	})
	identity := nativeFileIdentity(t, source)
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(parent, "directory")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	sibling := nativeSibling(t, destination)
	directory, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	request := renameSpec{from: source.Name(), to: destination}
	assertNativeRenameError(t, request, (localFilesystem{}).Rename(request))
	assertNativeSameEntry(t, request.from, identity)
	assertNativeSameEntry(t, request.to, directory)
	assertNativeSibling(t, sibling)
}
