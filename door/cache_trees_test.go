package door

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/janearc/slayer/treebolt"
)

// treesMade is how many trees a test puts in the tree file
const treesMade = 3

// treeFileIn makes the tree file in a cache home, with treesMade trees
// last used at the time, and says its path.
func treeFileIn(t *testing.T, home string, used time.Time) string {
	t.Helper()
	path := filepath.Join(home, treebolt.File)
	file, err := treebolt.Open(path, time.Minute, false)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Update(func(writing *bolt.Tx) error {
		for _, name := range []string{"one", "two", "three"} {
			err := treebolt.Put(writing, []byte(name), []byte(name), used)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || file.Close() != nil {
		t.Fatal(err)
	}
	return path
}

// homeOf is the cache home of a test's store.
func homeOf(folder string) string { return filepath.Dir(folder) }

func TestCacheSaysTheTreeFileLikeTheOtherKinds(t *testing.T) {
	env, store := cacheEnv(t)
	path := treeFileIn(t, homeOf(store.Folder()), noon())
	got := running(Tool(env), "cache")
	for _, want := range []string{"trees (one file): 3 entries, ",
		"MiB on disc, ", "of 8192 MiB", "(moonlake's trees are about",
		"  used from ", "  in " + path} {
		if got.code != nothing || !strings.Contains(got.said, want) {
			t.Fatalf("%q is not in %d: %s%s", want, got.code, got.said,
				got.wrongSaid)
		}
	}
}

func TestCacheClearTreesClearsTheFileToo(t *testing.T) {
	env, store := cacheEnv(t)
	home := homeOf(store.Folder())
	path := treeFileIn(t, home, noon())
	got := running(Tool(env), "cache", "--clear", "trees")
	usage, err := treebolt.Look(path, time.Minute)
	if got.code != nothing || err != nil || usage.Entries != nothing ||
		!strings.Contains(got.said, "trees: removed 3 entries from "+path) {
		t.Fatalf("%d: %s%s %+v %v", got.code, got.said, got.wrongSaid,
			usage, err)
	}
}

func TestCacheClearWithNoKindClearsTheFileToo(t *testing.T) {
	env, store := cacheEnv(t)
	path := treeFileIn(t, homeOf(store.Folder()), noon())
	got := running(Tool(env), "cache", "--clear")
	usage, err := treebolt.Look(path, time.Minute)
	if got.code != nothing || err != nil || usage.Entries != nothing {
		t.Fatalf("%d: %s%s %+v %v", got.code, got.said, got.wrongSaid,
			usage, err)
	}
}

func TestCacheClearDryRunSaysTheFileAndKeepsIt(t *testing.T) {
	env, store := cacheEnv(t)
	path := treeFileIn(t, homeOf(store.Folder()), noon())
	got := running(Tool(env), "cache", "--clear", "trees", "--dry-run")
	usage, err := treebolt.Look(path, time.Minute)
	if got.code != nothing || err != nil || usage.Entries != treesMade ||
		!strings.Contains(got.said, "trees: would remove 3 entries from "+
			path) {
		t.Fatalf("%d: %s%s %+v %v", got.code, got.said, got.wrongSaid,
			usage, err)
	}
}

// anti-test: clearing another kind leaves the tree file as it was.
func TestCacheClearOfAnotherKindLeavesTheTreeFileAlone(t *testing.T) {
	env, store := cacheEnv(t)
	path := treeFileIn(t, homeOf(store.Folder()), noon())
	got := running(Tool(env), "cache", "--clear", "door-verb-test")
	usage, err := treebolt.Look(path, time.Minute)
	if got.code != nothing || err != nil || usage.Entries != treesMade ||
		strings.Contains(got.said, "(one file)") {
		t.Fatalf("%d: %s%s %+v %v", got.code, got.said, got.wrongSaid,
			usage, err)
	}
}

// anti-test: with no tree file made, a listing says no entries and does
// not make one.
func TestCacheMakesNoTreeFileToSayIt(t *testing.T) {
	env, store := cacheEnv(t)
	path := filepath.Join(homeOf(store.Folder()), treebolt.File)
	got := running(Tool(env), "cache")
	if got.code != nothing ||
		!strings.Contains(got.said, "trees (one file): 0 entries") {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
	if usage, _ := treebolt.Look(path, time.Minute); usage.Disc != nothing {
		t.Fatalf("a listing made %s", path)
	}
}
