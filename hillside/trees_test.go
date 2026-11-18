package hillside

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain gives every test a cache home of its own, so a run that
// needs a cache never keeps one in a real place.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "hillside-cache")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Setenv("SLAYER_CACHE", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// lookupOf is a lookup of a test's own variables.
func lookupOf(said map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, set := said[name]
		return value, set
	}
}

// The trees are kept in the tree file in the cache home, beside the
// trees kind's folder.
func TestTheTreesAreKeptInTheCacheHome(t *testing.T) {
	home := t.TempDir()
	got, err := treePath(lookupOf(map[string]string{
		"SLAYER_CACHE": home}))
	if err != nil || got != filepath.Join(home, "trees.bolt") {
		t.Fatalf("tree file %q, %v", got, err)
	}
}

// anti-test: with SLAYER_CACHE unset the tree cache is refused by name,
// whatever the renders home, the user's cache and the temporary folder
// say, and nothing is made in any of them.
func TestTheTreesAreRefusedWithNoCacheHome(t *testing.T) {
	folders := map[string]string{"SLAYER_RENDERS": t.TempDir(),
		"HOME": t.TempDir(), "XDG_CACHE_HOME": t.TempDir(),
		"TMPDIR": t.TempDir(), "SLAYER_TREE_CACHE": t.TempDir()}
	err := cachingTrees(lookupOf(folders))
	if err == nil || !strings.Contains(err.Error(), "SLAYER_CACHE") {
		t.Fatalf("no cache home gave %v", err)
	}
	for name, folder := range folders {
		if found, _ := os.ReadDir(folder); len(found) != 0 {
			t.Errorf("%s was written to: %v", name, found)
		}
	}
}

// unhomedRun is a run of the command with no SLAYER_CACHE, and with the
// folders it might have fallen back to given and left empty, so the
// caller can see nothing was made in them.
func unhomedRun(t *testing.T, arguments ...string) (int, string, string,
	[]string) {
	t.Helper()
	t.Setenv("SLAYER_CACHE", "")
	folders := []string{}
	for _, name := range []string{"SLAYER_RENDERS", "HOME",
		"XDG_CACHE_HOME", "TMPDIR"} {
		folder := t.TempDir()
		t.Setenv(name, folder)
		folders = append(folders, folder)
	}
	var said, wrong strings.Builder
	code := run(arguments, &said, &wrong)
	return code, said.String(), wrong.String(), folders
}

// anti-test: with SLAYER_CACHE unset, help exits as help does and prints
// its usage, for the command and for the sequence verb, and nothing is
// created anywhere.
func TestHelpNeedsNoCacheHome(t *testing.T) {
	for _, each := range []struct {
		arguments []string
		usage     string
	}{{[]string{"-help"}, "usage"}, {[]string{"--help"}, "usage"},
		{[]string{"sequence", "--help"}, "--plan"}} {
		arguments := each.arguments
		code, said, wrong, folders := unhomedRun(t, arguments...)
		if code != 0 || !strings.Contains(strings.ToLower(said+wrong),
			each.usage) || strings.Contains(said+wrong, "SLAYER_CACHE") {
			t.Errorf("%v gave %d, %q, %q", arguments, code, said, wrong)
		}
		for _, folder := range folders {
			if found, _ := os.ReadDir(folder); len(found) != 0 {
				t.Errorf("%v made %v in %s", arguments, found, folder)
			}
		}
	}
}

// anti-test: a real run with SLAYER_CACHE unset is still refused by
// name, and nothing is created anywhere.
func TestARunWithNoCacheHomeIsRefused(t *testing.T) {
	code, _, wrong, folders := unhomedRun(t, "-w", "8", "-h", "8", "-n", "1")
	if code != 1 || !strings.Contains(wrong, "SLAYER_CACHE") {
		t.Errorf("a run with no home gave %d, %q", code, wrong)
	}
	for _, folder := range folders {
		if found, _ := os.ReadDir(folder); len(found) != 0 {
			t.Errorf("a refused run made %v in %s", found, folder)
		}
	}
}

// anti-test: a relative SLAYER_CACHE is refused by name.
func TestTheTreesAreRefusedWithARelativeCacheHome(t *testing.T) {
	err := cachingTrees(lookupOf(map[string]string{
		"SLAYER_CACHE": "relative"}))
	if err == nil || !strings.Contains(err.Error(), "SLAYER_CACHE") {
		t.Fatalf("a relative home gave %v", err)
	}
}
