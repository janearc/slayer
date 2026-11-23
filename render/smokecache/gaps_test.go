package smokecache

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRootWithNoCacheDirectory checks that the refusal to find a cache
// home is returned when SLAYER_CACHE is not set.
func TestRootWithNoCacheDirectory(t *testing.T) {
	got, err := Root(rootLookup(nil))
	if err == nil || got != "" {
		t.Errorf("got %q, %v", got, err)
	}
}

// TestOneLine checks that only the first line of a message is kept, marked
// as cut, and that a single line comes back without its edges.
func TestOneLine(t *testing.T) {
	cases := map[string]string{
		"  plain  ":             "plain",
		"first\nsecond\nthird":  "first (and more)",
		"\n\nfirst\nsecond\n\n": "first (and more)",
		"":                      "",
	}
	for in, want := range cases {
		if got := oneLine(in); got != want {
			t.Errorf("oneLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestStoreFailures checks that a parent folder that cannot be written to,
// and a name in the way that is not a folder, are each returned as errors
// with nothing left behind.
func TestStoreFailures(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not bind the superuser")
	}
	t.Run("parent not writable", func(t *testing.T) {
		root := t.TempDir()
		parent := filepath.Join(root, "v1")
		if err := os.MkdirAll(parent, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(parent, 0o555); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(parent, 0o755)
		e, err := Store(root, key(), "record", false, fill("a"))
		if err == nil || e != nil {
			t.Fatalf("got %v, %v", e, err)
		}
	})
	t.Run("name taken by a file", func(t *testing.T) {
		root := t.TempDir()
		parent := filepath.Join(root, "v1")
		if err := os.MkdirAll(parent, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(parent, key().Name())
		if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		e, err := Store(root, key(), "record", false, fill("a"))
		if err == nil || e != nil || errors.Is(err, os.ErrExist) {
			t.Fatalf("got %v, %v", e, err)
		}
		left, readErr := os.ReadDir(parent)
		if readErr != nil || len(left) != 1 {
			t.Errorf("left behind: %v %v", left, readErr)
		}
	})
}

// TestIsSlayerNeedsItsModuleLine checks that a go.mod of another module, or
// with no module line, is not slayer's.
func TestIsSlayerNeedsItsModuleLine(t *testing.T) {
	cases := map[string]bool{
		"module " + slayerModule + "\n\ngo 1.22\n": true,
		"module example.com/other\n":               false,
		"go 1.22\n// no module line\n":             false,
		"":                                         false,
	}
	for text, want := range cases {
		dir := t.TempDir()
		write(t, dir, "go.mod", text)
		if got := isSlayer(dir); got != want {
			t.Errorf("go.mod %q: got %t, want %t", text, got, want)
		}
	}
}

// TestReplacedRefusesATreeWithoutLibtheme checks that a folder that is not a
// module, and a module that does not use libtheme, each give an error.
func TestReplacedRefusesATreeWithoutLibtheme(t *testing.T) {
	bare := t.TempDir()
	if dir, err := replaced(bare); err == nil || dir != "" {
		t.Errorf("no module: %q, %v", dir, err)
	}
	module := t.TempDir()
	write(t, module, "go.mod", "module example.com/x\n\ngo 1.21\n")
	if dir, err := replaced(module); err == nil || dir != "" {
		t.Errorf("unrelated module: %q, %v", dir, err)
	}
}

// TestCommitSaysWhenStatusFails checks that a tree whose index cannot be
// read still gives its commit, with the reason its changes are unknown.
func TestCommitSaysWhenStatusFails(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "a.go", "package a\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "--no-verify", "-m", "one")
	head := git(t, dir, "rev-parse", "HEAD")
	write(t, filepath.Join(dir, ".git"), "index", "this is not an index")
	got := commit(dir)
	if !strings.HasPrefix(got, head+", its changes unknown: ") {
		t.Errorf("got %q", got)
	}
}
