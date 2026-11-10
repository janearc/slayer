//go:build darwin

package publication

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// ownerDirectoryPermissions are the OS owner's read, write and search bits.
const ownerDirectoryPermissions = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR

// TestPreparedNativeCompletes checks the complete adapter on private files.
// It establishes namespace visibility without a root or exclusion claim.
func TestPreparedNativeCompletes(t *testing.T) {
	parent := t.TempDir()
	old := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "state", body: []byte("old state")})
	namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "intent", body: []byte("observing")})
	request := preparedPublication{parent: parent,
		intent: []byte("prepared"), state: []byte("new state")}
	got := (publisher{filesystem: localFilesystem{}}).complete(request)
	if got.stage != completionComplete || got.err != nil {
		t.Fatalf("native completion: %+v", got)
	}
	assertNativeBytes(t, filepath.Join(parent, "state"), request.state)
	assertNativeHandleBytes(t, old, []byte("old state"))
	assertNativePublishedMode(t, filepath.Join(parent, "state"))
	assertNativeEntryNames(t, parent, []string{"state"})
}

// TestPreparedNativeStateRefusalRetainsIntent is the paired obstruction test.
// This known rename refusal does not establish rollback after other failures.
func TestPreparedNativeStateRefusalRetainsIntent(t *testing.T) {
	parent := t.TempDir()
	state := filepath.Join(parent, "state")
	if err := os.Mkdir(state, ownerDirectoryPermissions); err != nil {
		t.Fatal(err)
	}
	child := nativeSibling(t, state)
	identity, err := os.Stat(state)
	if err != nil {
		t.Fatal(err)
	}
	namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "intent", body: []byte("observing")})
	request := preparedPublication{parent: parent,
		intent: []byte("prepared"), state: []byte("new state")}
	got := (publisher{filesystem: localFilesystem{}}).complete(request)
	if got.stage != completionStateFailed || got.err == nil {
		t.Fatalf("native state refusal: %+v", got)
	}
	assertNativeBytes(t, filepath.Join(parent, "intent"), request.intent)
	assertNativeSameEntry(t, state, identity)
	assertNativeSibling(t, child)
	assertNativeEntryNames(t, parent, []string{"intent", "state"})
}
