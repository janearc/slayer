package weather

import (
	"path/filepath"
	"strings"
	"testing"
)

// The cache is SLAYER_WEATHER where it is set, and slayer/weather under
// the user's cache where it is not.
func TestTheCacheIsWhereHillsideKeepsIt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(cacheVariable, dir)
	if got, err := CacheDir(); err != nil || got != dir {
		t.Errorf("set: %q, %v", got, err)
	}
	t.Setenv(cacheVariable, "")
	got, err := CacheDir()
	if err != nil || !strings.HasSuffix(got,
		filepath.Join("slayer", "weather")) {
		t.Errorf("unset: %q, %v", got, err)
	}
}

// anti-test: with no home to find the user's cache in, the cache is
// refused rather than made somewhere else.
func TestNoHomeIsNoCache(t *testing.T) {
	t.Setenv(cacheVariable, "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	if got, err := CacheDir(); err == nil {
		t.Errorf("no home gave %q", got)
	}
}
