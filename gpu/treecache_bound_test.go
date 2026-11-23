//go:build draw

package gpu

import (
	"reflect"
	"slices"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/janearc/slayer/treebolt"
)

// anHourOn is a time long enough after a store that nothing it stored
// is held back from eviction
const anHourOn = time.Hour

// keptValue is whether the file holds a tree under the key
func keptValue(t *testing.T, kept *treeShelf, key programKey) bool {
	t.Helper()
	value, err := kept.look(key)
	if err != nil {
		t.Fatal(err)
	}
	return value != nil
}

// the trim drops the least recently used tree when the file is over its
// cap, and the one it keeps is still the tree grow makes
func TestTrimDropsTheLeastRecentlyUsedTreeAndKeepsTheOther(t *testing.T) {
	older, newer := someFaces(cachedFaces, 11), someFaces(cachedFaces, 12)
	kept := shelfIn(t, 1)
	kept.grown(slices.Clone(older))
	time.Sleep(2 * time.Millisecond)
	kept.grown(slices.Clone(newer))
	oneTree := int64(len(packTree(grownFresh(older))) + len(programKey{}))
	bound := treebolt.Bound{Cap: oneTree, Target: oneTree}
	said, err := kept.trimmed(bound, time.Now().Add(anHourOn))
	if err != nil || said == "" {
		t.Fatalf("trimmed %q, %v", said, err)
	}
	if keptValue(t, kept, kept.key(older)) ||
		!keptValue(t, kept, kept.key(newer)) {
		t.Fatal("the trim did not drop the older tree and keep the newer")
	}
	if got := kept.grown(slices.Clone(newer)); !reflect.DeepEqual(got,
		grownFresh(newer)) {
		t.Fatal("the tree kept is not the tree grow makes")
	}
}

// anti-test: a file under its cap loses nothing and says nothing
func TestTrimLeavesAFileUnderItsCap(t *testing.T) {
	faces := someFaces(cachedFaces, 13)
	kept := shelfIn(t, 1)
	kept.grown(slices.Clone(faces))
	said, err := kept.trimmed(treebolt.Standard(), time.Now())
	if err != nil || said != "" || !keptValue(t, kept, kept.key(faces)) {
		t.Fatalf("trimmed %q, %v", said, err)
	}
}

// a tree kept with no recorded use, as every tree of an older build is,
// is stamped at the run's end when a run takes it
func TestATreeTakenWithNoRecordedUseIsStampedAtTheEnd(t *testing.T) {
	faces := someFaces(cachedFaces, 14)
	kept := shelfIn(t, 1)
	key := kept.key(faces)
	file, err := treebolt.Open(kept.path, kept.wait, false)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Update(func(writing *bolt.Tx) error {
		return treebolt.Put(writing, key[:], packTree(grownFresh(faces)),
			time.Time{})
	})
	if err != nil || file.Close() != nil {
		t.Fatal(err)
	}
	if got := kept.grown(slices.Clone(faces)); !reflect.DeepEqual(got,
		grownFresh(faces)) || kept.tally.kept != 1 {
		t.Fatalf("the tree was not taken: %+v", kept.tally)
	}
	if _, err := kept.trimmed(treebolt.Standard(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if got, err := treebolt.Look(kept.path, kept.wait); err != nil ||
		got.Unrecorded != 0 {
		t.Fatalf("after the run %+v, %v", got, err)
	}
}
