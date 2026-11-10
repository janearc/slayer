package publication

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// ownerReadWritePermissions is defined by the OS owner-read and owner-write bits.
const ownerReadWritePermissions os.FileMode = syscall.S_IRUSR | syscall.S_IWUSR

// noPermissionBits is defined as the empty set of file permission bits.
const noPermissionBits os.FileMode = 0

// fileStart is defined as the byte offset of the beginning of a file.
const fileStart int64 = 0

// noDirectoryEntries is defined as the cardinality of an empty directory.
const noDirectoryEntries = 0

// createdNative keeps cleanup even if the result breaks its contract.
// All requested and fallback directories belong to the enclosing test.
func createdNative(t *testing.T, request tempSpec) openedTemp {
	t.Helper()
	opened := (localFilesystem{}).CreateTemp(request)
	if opened.file != nil {
		owner := nativeHandleCleanup{reporter: t, close: opened.file.Close}
		t.Cleanup(owner.finishIfOpen)
	}
	return opened
}

// TestLocalCreateTransfersAPrivateSameParentFile needs native files to
// establish name placement, mode and ownership. No flush or live paths.
func TestLocalCreateTransfersAPrivateSameParentFile(t *testing.T) {
	parent, fallback := t.TempDir(), t.TempDir()
	t.Setenv("TMPDIR", fallback)
	sibling := nativeSibling(t, parent)
	opened := createdNative(t, tempSpec{parent: parent, pattern: ".snapshot-*.tmp"})
	if opened.err != nil || opened.file == nil || opened.name == "" {
		t.Fatalf("create: %+v, want an owned handle and name", opened)
	}
	name := filepath.Base(opened.name)
	if filepath.Dir(opened.name) != parent ||
		!strings.HasPrefix(name, ".snapshot-") || !strings.HasSuffix(name, ".tmp") {
		t.Fatalf("temporary name: %q", opened.name)
	}
	file := nativeCreatedFile(t, opened.file)
	if file.Name() != opened.name {
		t.Fatalf("handle names %q, result names %q", file.Name(), opened.name)
	}
	written := opened.file.Write(writeSpec{body: []byte("owned")})
	if written.err != nil || written.written != len("owned") {
		t.Fatalf("transferred write: %+v", written)
	}
	assertNativeHandleBytes(t, file, []byte("owned"))
	if err := opened.file.Close(); err != nil {
		t.Fatal(err)
	}
	assertNativeSibling(t, sibling)
	assertNativeDirectory(t, fallback)
}

// nativeCreatedFile uses the open handle. No umask assumption is needed.
// No permission changes are needed to inspect the new file.
func nativeCreatedFile(t *testing.T, opened temporary) *os.File {
	t.Helper()
	local, ok := opened.(localTemporary)
	if !ok || local.file == nil {
		t.Fatalf("transferred handle: %T, want localTemporary", opened)
	}
	info, err := local.file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Size() != noWrittenBytes ||
		info.Mode().Perm()&^ownerReadWritePermissions != noPermissionBits {
		t.Fatalf("fresh file: mode %v, size %d", info.Mode(), info.Size())
	}
	return local.file
}

// nativeSibling retains the creation handle rather than assuming umask
// permits reopening it later. The new temporary must leave its bytes alone.
func nativeSibling(t *testing.T, parent string) *os.File {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(parent, "keep"),
		os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	owner := nativeHandleCleanup{reporter: t, close: file.Close}
	t.Cleanup(owner.finish)
	if _, err := file.Write([]byte("sibling")); err != nil {
		t.Fatal(err)
	}
	return file
}

// assertNativeHandleBytes checks fixture bytes through an owned open handle.
func assertNativeHandleBytes(t *testing.T, file *os.File, want []byte) {
	t.Helper()
	got := make([]byte, len(want))
	count, err := file.ReadAt(got, fileStart)
	if err != nil || count != len(want) || !bytes.Equal(got, want) {
		t.Fatalf("read: %x, count %d, error %v; want %x", got, count, err, want)
	}
}

// assertNativeSibling checks the directory entry too: an old held inode
// alone could hide removal or replacement of the sibling's name.
func assertNativeSibling(t *testing.T, file *os.File) {
	t.Helper()
	held, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	named, err := os.Stat(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(held, named) {
		t.Fatal("sibling directory entry was replaced")
	}
	assertNativeHandleBytes(t, file, []byte("sibling"))
}

// TestLocalCreateFailureTransfersNothing is the anti-test: invalid parents
// and patterns cannot fall back. They cannot create parents or transfer
// partial ownership.
func TestLocalCreateFailureTransfersNothing(t *testing.T) {
	for _, name := range []string{"empty", "relative", "unclean", "missing", "pattern"} {
		witness := nativeCreateRefusal{name: name}
		t.Run(name, witness.check)
	}
}

// nativeCreateRefusal names the same authored parent or pattern failure.
type nativeCreateRefusal struct {
	name string
}

// check keeps the exact refusal cause and owned-directory assertions together.
func (witness nativeCreateRefusal) check(t *testing.T) {
	name := witness.name
	parent, fallback := t.TempDir(), t.TempDir()
	t.Setenv("TMPDIR", fallback)
	request := tempSpec{parent: parent, pattern: ".snapshot-*"}
	cause := os.ErrInvalid
	switch name {
	case "empty":
		request.parent = ""
	case "relative":
		request.parent = "relative"
	case "unclean":
		request.parent += "/./"
	case "missing":
		request.parent = filepath.Join(parent, "missing")
		cause = os.ErrNotExist
	case "pattern":
		request.pattern = "bad/name-*"
		cause = nil
	}
	opened := createdNative(t, request)
	if opened.err == nil || opened.file != nil || opened.name != "" {
		t.Errorf("refused create: %+v, want only an error", opened)
	}
	if cause != nil && !errors.Is(opened.err, cause) {
		t.Errorf("cause: %v, want %v", opened.err, cause)
	}
	if name == "pattern" {
		var pathError *os.PathError
		if !errors.As(opened.err, &pathError) || pathError.Op != "createtemp" {
			t.Errorf("pattern error: %v, want native create error", opened.err)
		}
	}
	assertNativeDirectory(t, parent)
	assertNativeDirectory(t, fallback)
}

// assertNativeDirectory refuses unexpected files, including created parents.
func assertNativeDirectory(t *testing.T, parent string) {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != noDirectoryEntries {
		t.Fatalf("%s holds %v, want an empty fixture directory", parent, entries)
	}
}
