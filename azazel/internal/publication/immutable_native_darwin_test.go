package publication

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestImmutableNativePublishesCompleteBytes joins writing and real exclusion.
func TestImmutableNativePublishesCompleteBytes(t *testing.T) {
	parent := t.TempDir()
	sibling := nativeSibling(t, parent)
	body := payloadExamples(t).NativePublication.Binary
	request := installationSpec{parent: parent, name: "result", body: body}
	got := (immutablePublisher{filesystem: localFilesystem{}}).publish(request)
	if got.stage != publicationVisible || got.err != nil {
		t.Fatalf("native immutable publication: %+v", got)
	}
	assertNativeBytes(t, filepath.Join(parent, request.name), body)
	assertNativePublishedMode(t, filepath.Join(parent, request.name))
	assertNativeSibling(t, sibling)
	assertNativeEntryNames(t, parent, []string{"keep", "result"})
}

// TestImmutableNativeKeepsAnIncumbent retains its inode, old reader and bytes.
func TestImmutableNativeKeepsAnIncumbent(t *testing.T) {
	parent := t.TempDir()
	incumbent := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "result", body: []byte("first result"),
	})
	identity := nativeFileIdentity(t, incumbent)
	sibling := nativeSibling(t, parent)
	request := installationSpec{parent: parent, name: "result",
		body: payloadExamples(t).NativePublication.Binary}
	got := (immutablePublisher{filesystem: localFilesystem{}}).publish(request)
	if got.stage != publicationExists || !errors.Is(got.err, os.ErrExist) {
		t.Fatalf("native incumbent refusal: %+v", got)
	}
	assertNativeSameEntry(t, incumbent.Name(), identity)
	assertNativeHandleBytes(t, incumbent, []byte("first result"))
	assertNativeBytes(t, incumbent.Name(), []byte("first result"))
	assertNativeSibling(t, sibling)
	assertNativeEntryNames(t, parent, []string{"keep", "result"})
}
