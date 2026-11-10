package publication

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assertNativeEntryNames checks all parent entries, including leaked temporaries.
func assertNativeEntryNames(t *testing.T, parent string, want []string) {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		t.Fatalf("parent entries: %v, want %v", entries, want)
	}
	for index, entry := range entries {
		if entry.Name() != want[index] {
			t.Fatalf("entry %d: %s, want %s", index, entry.Name(), want[index])
		}
	}
}

// TestNativePublicationReplacesWholeSnapshots needs the complete local adapter
// to inspect actual output, retained readers and cleanup. No flush or live root.
func TestNativePublicationReplacesWholeSnapshots(t *testing.T) {
	examples := payloadExamples(t)
	for _, sample := range examples.NativePublication.cases() {
		t.Run(sample.name, sample.checkPublication)
	}
}

// checkPublication keeps old-reader, sibling, mode and namespace witnesses.
func (sample nativePayloadCase) checkPublication(t *testing.T) {
	parent := t.TempDir()
	old := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "snapshot", body: []byte("before"),
	})
	sibling := nativeSibling(t, parent)
	request := replacement{parent: parent, name: "snapshot", body: sample.body}
	result := (publisher{filesystem: localFilesystem{}}).replace(request)
	if result.stage != publicationVisible || result.err != nil {
		t.Fatalf("publication: %+v", result)
	}
	destination := filepath.Join(parent, request.name)
	assertNativeBytes(t, destination, sample.body)
	assertNativeHandleBytes(t, old, []byte("before"))
	assertNativeSibling(t, sibling)
	assertNativePublishedMode(t, destination)
	assertNativeEntryNames(t, parent, []string{"keep", "snapshot"})
}

// assertNativePublishedMode never changes a result's permissions to pass a check.
func assertNativePublishedMode(t *testing.T, name string) {
	t.Helper()
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() ||
		info.Mode().Perm()&^ownerReadWritePermissions != noPermissionBits {
		t.Fatalf("published mode: %v", info.Mode())
	}
}

// TestNativePublicationRefusalKeepsTheDirectory is the paired anti-test.
// This pre-change refusal does not support a general rollback claim.
func TestNativePublicationRefusalKeepsTheDirectory(t *testing.T) {
	parent := t.TempDir()
	destination := filepath.Join(parent, "directory")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	child, sibling := nativeSibling(t, destination), nativeSibling(t, parent)
	identity, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	request := replacement{parent: parent, name: "directory", body: []byte("refuse")}
	result := (publisher{filesystem: localFilesystem{}}).replace(request)
	if result.stage != publicationRenameUncertain {
		t.Fatalf("refusal stage: %+v", result)
	}
	var refusal *os.LinkError
	if !errors.As(result.err, &refusal) || refusal.Op != "rename" ||
		refusal.New != destination || filepath.Dir(refusal.Old) != parent ||
		!strings.HasPrefix(filepath.Base(refusal.Old), ".directory-") {
		t.Fatalf("rename cause: %v", result.err)
	}
	assertNativeSameEntry(t, destination, identity)
	assertNativeSibling(t, child)
	assertNativeSibling(t, sibling)
	assertNativeEntryNames(t, parent, []string{"directory", "keep"})
}
