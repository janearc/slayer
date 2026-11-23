package diskcache

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// strangerBytes is a stranger's size: more than the cap, so a store
// that counted one would be over its cap and trim to get under it.
const strangerBytes = testCap + entryBytes

// The fan folder a test puts a key's file in when it is the wrong one.
const (
	wrongFan = "zz"
	otherFan = "yy"
)

// planted is a store of three entries, the clock moved past heldFor so
// any of them may be trimmed, and a stranger made as told, last used
// before them all. It says where the stranger is.
func planted(t *testing.T, name string, plant func(*Store, []Key) string) (
	*Store, *clock, []Key, string) {
	t.Helper()
	store, ticking, keys := filled(t, name, 3)
	ticking.at = ticking.at.Add(heldFor)
	return store, ticking, keys, plant(store, keys)
}

// stranger writes a file of strangerBytes at a path, in folders made,
// last used when the clock started, and says the path.
func stranger(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload(strangerBytes), fileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, started, started); err != nil {
		t.Fatal(err)
	}
	return path
}

// kept is whether a stranger is still there, and its bytes unchanged.
func kept(path string) bool {
	raw, err := os.ReadFile(path)
	return err == nil && bytes.Equal(raw, payload(strangerBytes))
}

// survives holds a stranger to the whole rule: not counted by Usage,
// not removed by a Put that trims the store, not removed by Clear. The
// strangers made are named by the paths the test gives.
func survives(t *testing.T, name string, plant func(*Store, []Key) string,
	targets ...string) {
	t.Helper()
	store, _, keys, path := planted(t, name, plant)
	usage, err := store.Usage()
	if err != nil || usage.Entries() != len(keys) ||
		usage.Bytes() >= strangerBytes {
		t.Errorf("a stranger was counted: %+v, %v", usage, err)
	}
	if err := store.Put(NewKey(store.kind).Text("i", "new"),
		payload(entryBytes)); err != nil {
		t.Fatal(err)
	}
	if there(store, keys[0]) || !kept(path) {
		t.Fatalf("trimming: oldest entry there %v, stranger kept %v; "+
			"want the entry gone and the stranger kept", there(store,
			keys[0]), kept(path))
	}
	removed, err := store.Clear()
	if err != nil || removed != len(keys) || !kept(path) {
		t.Fatalf("Clear removed %d, stranger kept %v, %v", removed,
			kept(path), err)
	}
	for _, target := range targets {
		if !kept(target) {
			t.Fatalf("a link's target %s was removed", target)
		}
	}
}

// The anti-test: a file at the root of the store's folder, whatever its
// name, is never listed, counted or removed.
func TestAStrangerAtTheRootSurvives(t *testing.T) {
	survives(t, "stranger-root", func(store *Store, _ []Key) string {
		return stranger(t, filepath.Join(store.folder, "holiday.jpg"))
	})
}

// The anti-test: a file in a fan folder whose name is not a key's is
// never listed, counted or removed.
func TestAStrangerOfAnotherNameSurvives(t *testing.T) {
	survives(t, "stranger-name", func(store *Store, keys []Key) string {
		name := keys[0].Name()
		return stranger(t, filepath.Join(store.folder, name[:fanDigits],
			"photograph.jpg"))
	})
}

// wrongFolder is the fan folder a key's name does not belong in.
func wrongFolder(key Key) string {
	if key.Name()[:fanDigits] == wrongFan {
		return otherFan
	}
	return wrongFan
}

// The anti-test: a file with a key's name in a fan folder that is not
// the name's first digits is never listed, counted or removed.
func TestAStrangerInTheWrongFanFolderSurvives(t *testing.T) {
	survives(t, "stranger-fan", func(store *Store, keys []Key) string {
		key := NewKey(store.kind).Text("i", "elsewhere")
		elsewhere := filepath.Join(store.folder, wrongFolder(key),
			key.Name())
		return stranger(t, elsewhere)
	})
}

// The anti-test: a file in a folder below a fan folder, even one with
// a key's name in its right fan folder, is never listed, counted or
// removed.
func TestAStrangerBelowAFanFolderSurvives(t *testing.T) {
	survives(t, "stranger-deep", func(store *Store, keys []Key) string {
		name := keys[0].Name()
		return stranger(t, filepath.Join(store.folder, name[:fanDigits],
			"album", name))
	})
}

// The anti-test: a symbolic link in a fan folder with a key's name,
// pointing at a file outside the store, is never listed, counted or
// removed, and its target is untouched.
func TestAStrangerLinkSurvivesAndSoDoesItsTarget(t *testing.T) {
	photograph := filepath.Join(t.TempDir(), "photograph.jpg")
	target := stranger(t, photograph)
	survives(t, "stranger-link", func(store *Store, _ []Key) string {
		key := NewKey(store.kind).Text("i", "linked")
		link := filepath.Join(store.folder, key.Name()[:fanDigits],
			key.Name())
		if err := os.MkdirAll(filepath.Dir(link), folderMode); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		return link
	}, target)
}

// The anti-test: a stale file that ends in .part but is not a part of
// any entry, at the root or in a fan folder, is left alone.
func TestAStrangerPartIsNotRemoved(t *testing.T) {
	store, ticking, keys := filled(t, "stranger-part", 1)
	root := stranger(t, filepath.Join(store.folder, "movie.part"))
	fannedPath := filepath.Join(store.folder,
		keys[0].Name()[:fanDigits], "movie.part")
	fanned := stranger(t, fannedPath)
	ticking.at = started.Add(staleParts + minuteLater)
	if err := store.Put(NewKey(store.kind), nil); err != nil {
		t.Fatal(err)
	}
	if !kept(root) || !kept(fanned) {
		t.Fatal("a stale file that is no entry's part was removed")
	}
}
