package weather

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lookupOf is a lookup of a test's own variables.
func lookupOf(said map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, set := said[name]
		return value, set
	}
}

// The cache is the weather folder under SLAYER_CACHE.
func TestTheCacheIsWhereHillsideKeepsIt(t *testing.T) {
	dir := t.TempDir()
	got, err := CacheDir(lookupOf(map[string]string{"SLAYER_CACHE": dir}))
	if err != nil || got != filepath.Join(dir, "weather") {
		t.Errorf("set: %q, %v", got, err)
	}
}

// anti-test: with SLAYER_CACHE unset, the cache is refused by name, even
// where the renders home, the user's cache and the temporary folder are
// given, and nothing is made in any of them.
func TestNoCacheHomeIsNoCache(t *testing.T) {
	folders := map[string]string{"SLAYER_RENDERS": t.TempDir(),
		"HOME": t.TempDir(), "XDG_CACHE_HOME": t.TempDir(),
		"TMPDIR": t.TempDir()}
	got, err := CacheDir(lookupOf(folders))
	if err == nil || !strings.Contains(err.Error(), "SLAYER_CACHE") {
		t.Errorf("no home gave %q, %v", got, err)
	}
	for name, folder := range folders {
		if found, _ := os.ReadDir(folder); len(found) != 0 {
			t.Errorf("%s was written to: %v", name, found)
		}
	}
}

// anti-test: a relative SLAYER_CACHE is refused by name.
func TestARelativeCacheHomeIsNoCache(t *testing.T) {
	_, err := CacheDir(lookupOf(map[string]string{
		"SLAYER_CACHE": "relative"}))
	if err == nil || !strings.Contains(err.Error(), "SLAYER_CACHE") {
		t.Errorf("a relative home was taken: %v", err)
	}
}
