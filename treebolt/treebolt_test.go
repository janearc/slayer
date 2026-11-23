package treebolt

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

// testWait is how long a test waits for the file's lock: long enough
// for a slow machine, short enough that a hang is found.
const testWait = 20 * time.Second

// folderMode is the permissions of a folder a test makes
const folderMode = 0o755

// treeBytes is the bytes of the value a test tree holds
const treeBytes = 100_000

// treesMade is how many trees a test file is made with
const treesMade = 10

// entryBytes is what one test tree counts for: its key and its value
const entryBytes = int64(len("key-0") + treeBytes)

// noon is a time to measure the uses of the trees from
var noon = time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC)

// treeKey is the key of the test tree at an index
func treeKey(index int) []byte { return []byte(fmt.Sprintf("key-%d", index)) }

// filled makes a file of treesMade trees, the tree at index last used
// the number of days before noon that useOf says, none where it is the
// zero time, and says the file's path.
func filled(t *testing.T, useOf func(index int) time.Time) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), File)
	file, err := Open(path, testWait, false)
	if err != nil {
		t.Fatal(err)
	}
	for index := range treesMade {
		err = file.Update(func(writing *bolt.Tx) error {
			value := bytes.Repeat([]byte{byte(index)}, treeBytes)
			return Put(writing, treeKey(index), value, useOf(index))
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// daysAgo is noon that many days before, with index 0 the oldest
func daysAgo(index int) time.Time {
	return noon.Add(-time.Duration(treesMade-index) * 24 * time.Hour)
}

// held is the keys a file holds, by looking at it
func held(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := Open(path, testWait, true)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	err = file.View(func(reading *bolt.Tx) error {
		for _, each := range entriesOf(reading) {
			found[string(each.Key)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return found
}

func TestLookSaysTheEntriesTheirBytesAndTheRangeOfUse(t *testing.T) {
	path := filled(t, daysAgo)
	got, err := Look(path, testWait)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries != treesMade || got.Live != treesMade*entryBytes ||
		got.Unrecorded != 0 || got.Disc < got.Live || got.Path != path ||
		!got.Oldest.Equal(daysAgo(0)) ||
		!got.Newest.Equal(daysAgo(treesMade-1)) {
		t.Fatalf("looked at %+v", got)
	}
}

func TestLookCountsTheTreesWithNoRecordedUse(t *testing.T) {
	path := filled(t, func(int) time.Time { return time.Time{} })
	got, err := Look(path, testWait)
	if err != nil || got.Entries != treesMade ||
		got.Unrecorded != treesMade || !got.Oldest.IsZero() {
		t.Fatalf("looked at %+v, %v", got, err)
	}
}

// anti-test: a file not made yet holds nothing, and looking makes none.
func TestLookAtNoFileMakesNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), File)
	got, err := Look(path, testWait)
	if err != nil || got.Entries != 0 || got.Disc != 0 {
		t.Fatalf("looked at %+v, %v", got, err)
	}
	_, err = rewrite(path, testWait, everything)
	if err != nil || sizeOf(path) != 0 {
		t.Fatalf("a rewrite of no file made one: %v", err)
	}
}

func TestTrimDropsTheLeastRecentlyUsedDownToTheTarget(t *testing.T) {
	path := filled(t, daysAgo)
	before := sizeOf(path)
	bound := Bound{Cap: before - 1, Target: treesMade / 2 * entryBytes}
	got, err := Trim(path, testWait, bound, noon)
	if err != nil || got.Dropped != treesMade/2 || got.Kept != treesMade/2 {
		t.Fatalf("trimmed %+v, %v", got, err)
	}
	kept := held(t, path)
	for index := range treesMade {
		if want := index >= treesMade/2; kept[string(treeKey(index))] != want {
			t.Errorf("tree %d kept is %v, want %v", index, !want, want)
		}
	}
	if got.After >= before {
		t.Errorf("the file went from %d to %d bytes", before, got.After)
	}
}

// anti-test: a file under its cap is not opened, so one held by
// another process costs nothing, and none of its trees go.
func TestTrimLeavesAFileUnderItsCapAlone(t *testing.T) {
	path := filled(t, daysAgo)
	holder, err := Open(path, testWait, false)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	bound := Bound{Cap: sizeOf(path), Target: entryBytes}
	got, err := Trim(path, time.Millisecond, bound, noon)
	if err != nil || got.Dropped != 0 {
		t.Fatalf("trimmed %+v, %v", got, err)
	}
}

// anti-test: a tree used in the last ten minutes is never dropped, even
// when the file stays over its cap for it.
func TestTrimNeverDropsATreeUsedJustNow(t *testing.T) {
	path := filled(t, func(int) time.Time { return noon.Add(-time.Minute) })
	bound := Bound{Cap: sizeOf(path) - 1, Target: entryBytes}
	got, err := Trim(path, testWait, bound, noon)
	if err != nil || got.Dropped != 0 || got.Kept != treesMade ||
		len(held(t, path)) != treesMade {
		t.Fatalf("trimmed %+v, %v", got, err)
	}
}

func TestTrimDropsTheTreesWithNoRecordedUseFirst(t *testing.T) {
	path := filled(t, func(index int) time.Time {
		if index%2 == 0 {
			return time.Time{}
		}
		return daysAgo(index)
	})
	bound := Bound{Cap: sizeOf(path) - 1, Target: treesMade / 2 * entryBytes}
	if _, err := Trim(path, testWait, bound, noon); err != nil {
		t.Fatal(err)
	}
	for index, kept := range []bool{false, true, false, true, false, true,
		false, true, false, true} {
		if held(t, path)[string(treeKey(index))] != kept {
			t.Errorf("tree %d kept is %v, want %v", index, !kept, kept)
		}
	}
}

func TestClearEmptiesTheFileAndGivesTheDiscBack(t *testing.T) {
	path := filled(t, daysAgo)
	before := sizeOf(path)
	got, err := Clear(path, testWait)
	if err != nil || got.Dropped != treesMade || sizeOf(path) >= before {
		t.Fatalf("cleared %+v, %v, %d from %d bytes", got, err,
			sizeOf(path), before)
	}
	file, err := Open(path, testWait, false)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Update(func(writing *bolt.Tx) error {
		return Put(writing, treeKey(0), []byte("after"), noon)
	})
	if err != nil || file.Close() != nil || len(held(t, path)) != 1 {
		t.Fatalf("a cleared file did not take a tree: %v", err)
	}
}

// anti-test: a rewrite that cannot make its new file leaves the old one
// as it was.
func TestARewriteThatFailsLeavesTheFileAsItWas(t *testing.T) {
	path := filled(t, daysAgo)
	blocker := filepath.Join(path+tempSuffix, "in-the-way")
	if err := writeFile(blocker); err != nil {
		t.Fatal(err)
	}
	if _, err := Clear(path, testWait); err == nil {
		t.Fatal("a rewrite with its new file's name taken succeeded")
	}
	if len(held(t, path)) != treesMade {
		t.Fatal("the failed rewrite changed the file")
	}
}

// TestAProcessWaitingDuringARewriteOpensTheNewFile holds a rewrite in
// its choice, with the lock, while a writer and a reader wait for the
// lock; once it swaps the file, the writer's tree is in the new file,
// not lost in the old, and the reader does not see a tree the rewrite
// dropped.
func TestAProcessWaitingDuringARewriteOpensTheNewFile(t *testing.T) {
	path := filled(t, daysAgo)
	holding, release := make(chan bool), make(chan bool)
	rewritten := make(chan error, 1)
	go func() {
		_, err := rewrite(path, testWait, func(all []treeEntry) map[string]bool {
			holding <- true
			<-release
			return map[string]bool{string(treeKey(0)): true}
		})
		rewritten <- err
	}()
	<-holding
	stored, saw := make(chan error, 1), make(chan []byte, 1)
	go func() { stored <- storeLate(path) }()
	go func() { saw <- readFirst(path) }()
	time.Sleep(200 * time.Millisecond)
	release <- true
	if err := <-rewritten; err != nil {
		t.Fatal(err)
	}
	if err := <-stored; err != nil {
		t.Fatal(err)
	}
	if got := <-saw; got != nil {
		t.Errorf("a reader that waited saw the dropped tree: %d bytes",
			len(got))
	}
	kept := held(t, path)
	if !kept["late"] || kept[string(treeKey(0))] || len(kept) != treesMade {
		t.Fatalf("after the swap the file held %v", kept)
	}
}

// storeLate keeps a tree called late, as a writer that waited would.
func storeLate(path string) error {
	file, err := Open(path, testWait, false)
	if err != nil {
		return err
	}
	err = file.Update(func(writing *bolt.Tx) error {
		return Put(writing, []byte("late"), []byte("late"), noon)
	})
	return errors.Join(err, file.Close())
}

// writeFile makes a file, and the folders to it, with a word in it.
func writeFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("word"), FileMode)
}

// readFirst is the value of the first tree, as a reader that waited
// would find it, nil when it is gone.
func readFirst(path string) []byte {
	file, err := Open(path, testWait, true)
	if err != nil {
		return nil
	}
	defer file.Close()
	var value []byte
	_ = file.View(func(reading *bolt.Tx) error {
		value, _ = Get(reading, treeKey(0))
		return nil
	})
	return value
}

func TestAbsorbKeepsTheLaterUse(t *testing.T) {
	older := filled(t, func(int) time.Time { return noon.Add(-time.Hour) })
	newer := filled(t, func(int) time.Time { return noon })
	into, err := Open(older, testWait, false)
	if err != nil {
		t.Fatal(err)
	}
	from, err := Open(newer, testWait, true)
	if err != nil {
		t.Fatal(err)
	}
	err = from.View(func(reading *bolt.Tx) error {
		return into.Update(func(writing *bolt.Tx) error {
			return Absorb(writing, reading)
		})
	})
	if err != nil || from.Close() != nil || into.Close() != nil {
		t.Fatal(err)
	}
	if got, _ := Look(older, testWait); !got.Newest.Equal(noon) ||
		got.Entries != treesMade {
		t.Fatalf("absorbed %+v", got)
	}
}

func TestTouchWritesTheUseOfTheTreesTheFileStillHolds(t *testing.T) {
	path := filled(t, daysAgo)
	keys := [][]byte{treeKey(0), []byte("gone")}
	if err := Touch(path, testWait, keys, noon); err != nil {
		t.Fatal(err)
	}
	got, err := Look(path, testWait)
	if err != nil || !got.Newest.Equal(noon) || got.Entries != treesMade {
		t.Fatalf("touched %+v, %v", got, err)
	}
}

func TestStaleIsNoRecordOrOlderThanTheGrain(t *testing.T) {
	for _, each := range []struct {
		used time.Time
		want bool
	}{
		{time.Time{}, true},
		{noon.Add(-2 * touchGrain), true},
		{noon.Add(-touchGrain / 2), false},
		{noon, false},
	} {
		if Stale(each.used, noon) != each.want {
			t.Errorf("Stale(%v) is %v, want %v", each.used, !each.want,
				each.want)
		}
	}
}
