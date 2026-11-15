package diskcache

import (
	"os"
	"strconv"
	"testing"
	"time"
)

// filled is a store with entries put a minute apart, each a test
// payload, and their keys, oldest first.
func filled(t *testing.T, name string, count int) (*Store, *clock, []Key) {
	t.Helper()
	store, ticking := testStore(t, name)
	var keys []Key
	for i := range count {
		key := NewKey(store.kind).Text("i", strconv.Itoa(i))
		if err := store.Put(key, payload(entryBytes)); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(store.path(key), ticking.at,
			ticking.at); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		ticking.at = ticking.at.Add(minuteLater)
	}
	return store, ticking, keys
}

// there is whether a key's entry is on disk.
func there(store *Store, key Key) bool {
	_, err := os.Stat(store.path(key))
	return err == nil
}

func TestUsageCountsEntriesAndTheirTimes(t *testing.T) {
	store, _, _ := filled(t, "usage-counted", 2)
	usage, err := store.Usage()
	if err != nil || usage.Entries() != 2 || usage.Bytes() < 2*entryBytes ||
		!usage.Oldest().Equal(started) ||
		!usage.Newest().Equal(started.Add(minuteLater)) ||
		usage.Kind() != store.kind || usage.Folder() != store.folder {
		t.Fatalf("usage %+v, %v", usage, err)
	}
}

// anti-test of TestUsageCountsEntriesAndTheirTimes
func TestUsageOfAnEmptyKindIsNothing(t *testing.T) {
	store, _ := testStore(t, "usage-empty")
	usage, err := store.Usage()
	if err != nil || usage.Entries() != nothing || usage.Bytes() != nothing ||
		!usage.Oldest().IsZero() {
		t.Fatalf("usage %+v, %v", usage, err)
	}
}

func TestOverTheCapTheLeastRecentlyUsedGoesFirst(t *testing.T) {
	store, ticking, keys := filled(t, "cap-lru", 3)
	ticking.at = ticking.at.Add(heldFor)
	if _, err := store.Get(keys[0]); err != nil {
		t.Fatal(err)
	}
	ticking.at = ticking.at.Add(heldFor)
	if err := store.Put(NewKey(store.kind).Text("i", "new"),
		payload(entryBytes)); err != nil {
		t.Fatal(err)
	}
	if there(store, keys[1]) || !there(store, keys[0]) ||
		!there(store, keys[2]) {
		t.Fatalf("kept: first %v, second %v, third %v; want the "+
			"second gone", there(store, keys[0]), there(store, keys[1]),
			there(store, keys[2]))
	}
}

// anti-test of TestOverTheCapTheLeastRecentlyUsedGoesFirst
func TestOverTheCapAnEntryJustUsedIsKept(t *testing.T) {
	store, _, keys := filled(t, "cap-held", 4)
	for _, key := range keys {
		if !there(store, key) {
			t.Fatal("an entry used inside heldFor was removed to make room")
		}
	}
}

// anti-test of TestOverTheCapTheLeastRecentlyUsedGoesFirst
func TestUnderTheCapNothingGoes(t *testing.T) {
	store, ticking, keys := filled(t, "cap-under", 2)
	ticking.at = ticking.at.Add(time.Hour * 24)
	if err := store.Put(keys[0], payload(entryBytes)); err != nil {
		t.Fatal(err)
	}
	if !there(store, keys[1]) {
		t.Fatal("an entry was removed under the cap")
	}
}

// parted is a .part file in the store's folder, last written at a time.
func parted(t *testing.T, store *Store, name string, at time.Time) string {
	t.Helper()
	path := store.folder + "/" + name + partSuffix
	if err := os.WriteFile(path, nil, fileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAStalePartIsRemoved(t *testing.T) {
	store, ticking := testStore(t, "parts-stale")
	stale := parted(t, store, "killed", started)
	ticking.at = started.Add(staleParts + minuteLater)
	if err := store.Put(NewKey(store.kind), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Fatal("a killed writer's part was kept")
	}
}

// anti-test of TestAStalePartIsRemoved
func TestAFreshPartIsLeftToItsWriter(t *testing.T) {
	store, ticking := testStore(t, "parts-fresh")
	fresh := parted(t, store, "writing", started)
	ticking.at = started.Add(minuteLater)
	if err := store.Put(NewKey(store.kind), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("a part being written was removed")
	}
}

func TestClearRemovesEveryEntry(t *testing.T) {
	store, _, keys := filled(t, "clear-all", 2)
	removed, err := store.Clear()
	if err != nil || removed != 2 || there(store, keys[0]) {
		t.Fatalf("removed %d, %v", removed, err)
	}
}

// anti-test of TestClearRemovesEveryEntry
func TestClearOfAFolderGoneSaysSo(t *testing.T) {
	store, _ := testStore(t, "clear-gone")
	store.folder = store.folder + "/not-there/" + string([]byte{0})
	if _, err := store.Clear(); err == nil {
		t.Fatal("clearing a folder that cannot be read said nothing")
	}
}

// anti-test of TestUsageCountsEntriesAndTheirTimes
func TestUsageDoesNotCountAPart(t *testing.T) {
	store, _, _ := filled(t, "usage-part", 2)
	parted(t, store, "writing", started)
	usage, err := store.Usage()
	if err != nil || usage.Entries() != 2 {
		t.Fatalf("usage %+v, %v; want the two entries, not the part",
			usage, err)
	}
}

// anti-test of TestClearRemovesEveryEntry
func TestClearLeavesAnotherKindsEntries(t *testing.T) {
	home := t.TempDir()
	mine, err := Open(home, testKind(t, "clear-mine"))
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := Open(home, testKind(t, "clear-theirs"))
	if err != nil {
		t.Fatal(err)
	}
	key, other := NewKey(mine.kind).Text("a", "1"), NewKey(theirs.kind)
	if err := mine.Put(key, nil); err != nil {
		t.Fatal(err)
	}
	if err := theirs.Put(other, nil); err != nil {
		t.Fatal(err)
	}
	if removed, err := mine.Clear(); err != nil || removed != 1 {
		t.Fatalf("removed %d, %v", removed, err)
	}
	if !there(theirs, other) {
		t.Fatal("clearing one kind removed another kind's entry")
	}
}
