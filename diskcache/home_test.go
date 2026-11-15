package diskcache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHomeTakesTheCacheVariableFirst(t *testing.T) {
	home, err := Home(lookupOf(map[string]string{
		homeVariable: "/fast/cache", rendersVariable: "/renders"}))
	if err != nil || home != "/fast/cache" {
		t.Fatalf("home %q, %v; want /fast/cache", home, err)
	}
}

// anti-test of TestHomeTakesTheCacheVariableFirst
func TestHomeRefusesARelativeCacheVariable(t *testing.T) {
	_, err := Home(lookupOf(map[string]string{homeVariable: "cache"}))
	if err == nil || !strings.Contains(err.Error(), homeVariable) {
		t.Fatalf("a relative %s was taken: %v", homeVariable, err)
	}
}

func TestHomeIsTheRendersHomesCache(t *testing.T) {
	home, err := Home(lookupOf(map[string]string{
		rendersVariable: "/renders"}))
	if err != nil || home != "/renders/cache" {
		t.Fatalf("home %q, %v; want /renders/cache", home, err)
	}
}

// anti-test of TestHomeIsTheRendersHomesCache
func TestHomeRefusesARelativeRendersHome(t *testing.T) {
	_, err := Home(lookupOf(map[string]string{rendersVariable: "r"}))
	if err == nil || !strings.Contains(err.Error(), rendersVariable) {
		t.Fatalf("a relative %s was taken: %v", rendersVariable, err)
	}
}

func TestHomeIsTheUsersCacheWithNeither(t *testing.T) {
	user, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	home, err := Home(lookupOf(map[string]string{homeVariable: ""}))
	if err != nil || home != filepath.Join(user, userFolder) {
		t.Fatalf("home %q, %v; want under %s", home, err, user)
	}
}

// anti-test of TestHomeIsTheUsersCacheWithNeither
func TestHomeWithNoUserCacheFolderSaysWhatToSet(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	_, err := Home(lookupOf(nil))
	if err == nil || !strings.Contains(err.Error(), homeVariable) {
		t.Fatalf("no user cache folder was not refused: %v", err)
	}
}
