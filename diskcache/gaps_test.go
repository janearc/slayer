package diskcache

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// lockFolders makes every folder under the store's folder read-only and
// restores them when the test ends, so removing an entry fails.
func lockFolders(t *testing.T, store *Store) {
	t.Helper()
	var folders []string
	err := filepath.WalkDir(store.folder, func(path string, each fs.DirEntry, err error) error {
		if err == nil && each.IsDir() {
			folders = append(folders, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range folders {
		if err := os.Chmod(folder, 0o500); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, folder := range folders {
			_ = os.Chmod(folder, 0o700)
		}
	})
}

// TestWrittenSaysWhenThePartCannotBeCreated checks that a write into a folder
// that is not there is an error and leaves no file.
func TestWrittenSaysWhenThePartCannotBeCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing", "entry")
	if err := written(path, payload(10)); err == nil {
		t.Fatal("wrote into a folder that is not there")
	}
	left, err := os.ReadDir(dir)
	if err != nil || len(left) != 0 {
		t.Errorf("left behind %v, %v", left, err)
	}
	good := filepath.Join(dir, "entry")
	if err := written(good, payload(10)); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(good); err != nil || len(got) != 10 || got[9] != 9 {
		t.Errorf("read back %v, %v", got, err)
	}
}

// TestTrimmedSaysWhenTheFolderCannotBeRead checks that a folder that cannot
// be listed is an error from trimming, not a silent success.
func TestTrimmedSaysWhenTheFolderCannotBeRead(t *testing.T) {
	store, _ := testStore(t, "trim-unreadable")
	store.folder = store.folder + "/not-there/" + string([]byte{0})
	if err := store.trimmed(); err == nil {
		t.Fatal("trimming a folder that cannot be read said nothing")
	}
}

// TestTrimmedSaysWhenAnEntryCannotBeRemoved checks that an entry that must go
// but cannot is an error, and that it is still there.
func TestTrimmedSaysWhenAnEntryCannotBeRemoved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not bind the superuser")
	}
	store, ticking, keys := filled(t, "trim-locked", 6)
	ticking.at = ticking.at.Add(48 * time.Hour)
	lockFolders(t, store)
	if err := store.trimmed(); err == nil {
		t.Fatal("an entry that could not be removed was not reported")
	}
	if !there(store, keys[0]) {
		t.Error("the entry went despite the error")
	}
}

// TestClearSaysWhenAnEntryCannotBeRemoved checks that Clear stops at an entry
// it cannot remove, and says how many it had removed.
func TestClearSaysWhenAnEntryCannotBeRemoved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not bind the superuser")
	}
	store, _, keys := filled(t, "clear-locked", 3)
	lockFolders(t, store)
	removed, err := store.Clear()
	if err == nil || removed != 0 {
		t.Fatalf("removed %d, %v", removed, err)
	}
	if !there(store, keys[0]) {
		t.Error("the entry went despite the error")
	}
}
