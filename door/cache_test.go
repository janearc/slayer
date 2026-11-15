package door

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/diskcache"
)

// The kind the verb's tests keep in, declared once; its cap is a
// mebibyte, so the line says one.
var verbKind = diskcache.MustKind("door-verb-test", mebibyte, "a test's")

// cacheEnv is an env whose cache home is a test's folder, with an entry
// of the test kind in it.
func cacheEnv(t *testing.T) (Env, *diskcache.Store) {
	t.Helper()
	home := t.TempDir()
	env, err := NewEnv(Given{Context: context.Background(),
		Environment: func(name string) (string, bool) {
			return home, name == "SLAYER_CACHE"
		}, Now: noon, Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	store, err := diskcache.Open(home, verbKind)
	if err != nil {
		t.Fatal(err)
	}
	key := diskcache.NewKey(verbKind).Text("a", "1")
	if err := store.Put(key, []byte("kept")); err != nil {
		t.Fatal(err)
	}
	return env, store
}

func TestCacheSaysEachKindItsEntriesAndFolder(t *testing.T) {
	env, store := cacheEnv(t)
	got := running(Tool(env), "cache")
	if got.code != nothing || !strings.Contains(got.said,
		"door-verb-test: 1 entries, 0.0 of 1 MiB (a test's)") ||
		!strings.Contains(got.said, "in "+store.Folder()) ||
		!strings.Contains(got.said, "used from ") {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
}

func TestCacheClearEmptiesOneKind(t *testing.T) {
	env, store := cacheEnv(t)
	got := running(Tool(env), "cache", "--clear", "door-verb-test")
	usage, _ := store.Usage()
	if got.code != nothing || usage.Entries() != nothing ||
		!strings.Contains(got.said, "door-verb-test: removed 1 entries") {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
}

// anti-test of TestCacheClearEmptiesOneKind
func TestCacheClearDryRunRemovesNothing(t *testing.T) {
	env, store := cacheEnv(t)
	got := running(Tool(env), "cache", "--clear", "--dry-run",
		"door-verb-test")
	usage, _ := store.Usage()
	if got.code != nothing || usage.Entries() != one ||
		!strings.Contains(got.said, "would remove 1 entries") {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
}

func TestCacheClearEmptiesEveryKind(t *testing.T) {
	env, store := cacheEnv(t)
	got := running(Tool(env), "cache", "--clear")
	usage, _ := store.Usage()
	if got.code != nothing || usage.Entries() != nothing {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
}

// anti-test of TestCacheClearEmptiesOneKind
func TestCacheRefusesAKindThereIsNot(t *testing.T) {
	env, store := cacheEnv(t)
	got := running(Tool(env), "cache", "--clear", "no-such-kind")
	usage, _ := store.Usage()
	if got.code == nothing || usage.Entries() != one ||
		!strings.Contains(got.wrongSaid+got.said, "no kind") {
		t.Fatalf("%d: %s%s", got.code, got.said, got.wrongSaid)
	}
}

// anti-test of TestCacheClearEmptiesOneKind
func TestCacheRefusesAKindWithoutClear(t *testing.T) {
	env, _ := cacheEnv(t)
	got := running(Tool(env), "cache", "door-verb-test")
	if got.code == nothing {
		t.Fatalf("a kind without --clear was taken: %s", got.said)
	}
}

// anti-test of TestCacheSaysEachKindItsEntriesAndFolder
func TestCacheRefusesARelativeHome(t *testing.T) {
	env, err := NewEnv(Given{Context: context.Background(),
		Environment: func(name string) (string, bool) {
			return "relative", name == "SLAYER_CACHE"
		}, Now: noon, Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if got := running(Tool(env), "cache"); got.code == nothing {
		t.Fatalf("a relative home was used: %s", got.said)
	}
}

// anti-test of TestCacheSaysEachKindItsEntriesAndFolder
func TestCacheSaysAnEmptyKindWithoutTimes(t *testing.T) {
	env, store := cacheEnv(t)
	if _, err := store.Clear(); err != nil {
		t.Fatal(err)
	}
	got := running(Tool(env), "cache")
	if strings.Contains(got.said, "used from") {
		t.Fatalf("an empty kind said when it was used: %s", got.said)
	}
}

func Example_cache() {
	home, _ := os.MkdirTemp("", "cache-example")
	defer os.RemoveAll(home)
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(name string) (string, bool) {
			return home, name == "SLAYER_CACHE"
		}, Now: func() time.Time { return time.Now() }, Directory: home})
	var said strings.Builder
	Tool(env).Run([]string{"cache", "--clear", "--dry-run",
		"door-verb-test"}, &said, &said)
	fmt.Print(strings.ReplaceAll(said.String(), home, "HOME"))
	// Output: door-verb-test: would remove 0 entries from HOME/door-verb-test
}
