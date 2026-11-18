package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/sources"
)

// treeFile is the sheet's name in a shelf: its key, and the ending its
// url has.
const treeFile = "candle.txt"

// wrongSheet is a copy of the sheet that is not the one pinned.
const wrongSheet = "the candle burns down 9.99 millimetres an hour"

// shelved writes bytes into a folder as the candle source, making the
// folder if it is not there.
func shelved(t *testing.T, dir, bytes string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, treeFile), []byte(bytes),
		openFile); err != nil {
		t.Fatal(err)
	}
}

// verifiedAlone is a verify of the rig's number with its shelf where
// shelfDir puts it, as main does: the exit and what it said. the tree's
// shelf is the folder beside the rig's list.
func verifiedAlone(t *testing.T, at place) (int, string) {
	t.Helper()
	dir, err := shelfDir(at.list)
	if err != nil {
		t.Fatal(err)
	}
	at.shelf = sources.NewShelf(dir, nil)
	var out bytes.Buffer
	code := tool(at).Run([]string{"verify"}, &out, &out)
	return code, out.String()
}

// unset takes SLAYER_SHELF out of the environment for the test.
func unset(t *testing.T) {
	t.Helper()
	t.Setenv(sources.Placed, "")
}

// With no shelf named, verify passes against a full shelf in the tree.
func TestVerifyPassesWithNoShelfNamed(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	shelved(t, filepath.Join(made.dir, treeFolder), sheet)
	unset(t)
	if code, said := verifiedAlone(t, at); code != 0 {
		t.Fatalf("verify with no shelf named gave %d: %s", code, said)
	}
}

// With no shelf named and a tree that lacks a source, verify refuses
// and names the file the tree should hold.
func TestVerifyRefusesATreeLackingASource(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	unset(t)
	code, said := verifiedAlone(t, at)
	want := filepath.Join(made.dir, treeFolder, treeFile)
	if code != 1 || !strings.Contains(said, want) {
		t.Fatalf("verify gave %d, want a refusal naming %s: %s",
			code, want, said)
	}
}

// A shelf named in the environment is the one verify reads: a wrong
// copy in the tree does not rescue it, and a right copy there is not
// used.
func TestVerifyReadsTheShelfNamedNotTheTree(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	named := filepath.Join(made.dir, "named")
	t.Setenv(sources.Placed, named)
	shelved(t, filepath.Join(made.dir, treeFolder), sheet)
	shelved(t, named, wrongSheet)
	if code, said := verifiedAlone(t, at); code != 1 ||
		!strings.Contains(said, "the list pins") {
		t.Fatalf("verify did not read the named shelf: %d: %s", code, said)
	}
	shelved(t, filepath.Join(made.dir, treeFolder), wrongSheet)
	shelved(t, named, sheet)
	if code, said := verifiedAlone(t, at); code != 0 {
		t.Fatalf("a wrong copy in the tree failed it: %d: %s", code, said)
	}
}

// A shelf named by a relative path is refused, not read as the tree.
func TestShelfDirRefusesARelativeName(t *testing.T) {
	t.Setenv(sources.Placed, "elsewhere")
	if dir, err := shelfDir("sources.yaml"); err == nil {
		t.Fatalf("a relative shelf was taken as %q", dir)
	}
}
