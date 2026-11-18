package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// a file's kind is told by its name.
func TestAFilesKindIsToldByItsName(t *testing.T) {
	for path, want := range map[string]string{
		"trace/surface.go":        "go",
		"trace/surface_test.go":   "go tests",
		"gpu/trace.metal":         "metal",
		"metal/bridge.m":          "objective-c",
		"metal/bridge.h":          "headers",
		"occultation/fit.c":       "c",
		"doc/sets.md":             "markdown",
		"sets/fizz.yaml":          "yaml",
		"manufacture/shoot.yml":   "yaml",
		"improv/moonlake/film.sh": "shell",
		".game":                   "gamefile",
		"azazel/.game":            "gamefile",
	} {
		if got := kindOf(path); got != want {
			t.Errorf("%s is %q, not %q", path, got, want)
		}
	}
}

// a name that only looks like a kind's is not one: an ending is the end
// of the name, a test is not go, and a folder's name says nothing.
func TestANameThatOnlyLooksLikeAKindIsEverythingElse(t *testing.T) {
	for _, path := range []string{
		"go.sum", "notes.md.bak", "surface.go.orig", "docs.md/table.csv",
		"README", "game", "x.game", "bridge.mm", "a.metal.txt", "",
	} {
		if got := kindOf(path); got != theRest {
			t.Errorf("%q is %q, not %q", path, got, theRest)
		}
	}
	if kindOf("a_test.go") == "go" || kindOf("a.go") == "go tests" {
		t.Error("a test and a go file are told apart by _test.go")
	}
}

// no two kinds claim one file, so the order they are listed in decides
// only how they are printed.
func TestNoTwoKindsClaimOneFile(t *testing.T) {
	for _, path := range []string{"a.go", "a_test.go", "a.metal", "a.m",
		"a.c", "a.h", "a.md", "a.yaml", "a.yml", "a.sh", ".game"} {
		claims := 0
		for _, each := range kinds {
			if each.is(path) {
				claims++
			}
		}
		if claims != 1 {
			t.Errorf("%d kinds claim %s", claims, path)
		}
	}
}

// the kinds are printed with go first and its tests after it.
func TestGoIsPrintedBeforeItsTests(t *testing.T) {
	if listed := names(); listed[0] != "go" || listed[1] != "go tests" {
		t.Errorf("the kinds begin %v", listed[:2])
	}
}

// a line is a newline, and a last line with none counts.
func TestALineIsANewline(t *testing.T) {
	for contents, want := range map[string]int{
		"": 0, "\n": 1, "a": 1, "a\n": 1, "a\nb": 2, "a\nb\n": 2, "\n\n\n": 3,
	} {
		lines, text := linesOf([]byte(contents))
		if lines != want || !text {
			t.Errorf("%q has %d lines (text %v), not %d", contents, lines,
				text, want)
		}
	}
}

// what is not text has no lines, however many newlines are in it.
func TestWhatIsNotTextHasNoLines(t *testing.T) {
	for _, contents := range []string{"\x00", "a\nb\n\x00", "\x00\n\n\n"} {
		if lines, text := linesOf([]byte(contents)); lines != 0 || text {
			t.Errorf("%q has %d lines (text %v)", contents, lines, text)
		}
	}
}

// held is a file's contents by its path, for a test to read from.
type held map[string]string

// read is the contents of a path, or an error for one that is not held.
func (files held) read(path string) ([]byte, error) {
	contents, found := files[path]
	if !found {
		return nil, errors.New("not held")
	}
	return []byte(contents), nil
}

// each kind's files and lines are added up, in the kinds' order with
// everything else last, and a kind with no file is left out.
func TestEachKindIsAddedUpInOrder(t *testing.T) {
	files := held{"b.md": "one\ntwo\n", "a.go": "x\n", "c.go": "y\nz",
		"a_test.go": "t\n", "pic.png": "\x00\n\n", "LICENSE": "words\n"}
	counts, err := counted([]string{"b.md", "a.go", "c.go", "a_test.go",
		"pic.png", "LICENSE"}, files.read)
	if err != nil {
		t.Fatal(err)
	}
	want := []count{{"go", 2, 3}, {"go tests", 1, 1}, {"markdown", 1, 2},
		{theRest, 2, 1}}
	if len(counts) != len(want) {
		t.Fatalf("the counts are %+v", counts)
	}
	for index := range want {
		if counts[index] != want[index] {
			t.Errorf("count %d is %+v, not %+v", index, counts[index],
				want[index])
		}
	}
}

// a file that cannot be read is an error naming it, and no count: a
// count that left a file out would be a wrong one.
func TestAFileThatCannotBeReadIsAnErrorAndNoCount(t *testing.T) {
	files := held{"a.go": "x\n"}
	counts, err := counted([]string{"a.go", "gone.go"}, files.read)
	if err == nil || !strings.Contains(err.Error(), "gone.go") ||
		counts != nil {
		t.Errorf("the counts are %+v and the error %v", counts, err)
	}
}

// what git tracks here is this tool's own files, each by its path from
// the folder the tool is run in.
func TestWhatGitTracksHereIsTheToolsOwnFiles(t *testing.T) {
	paths, err := tracked()
	if err != nil {
		t.Fatal(err)
	}
	for _, own := range []string{"main.go", "main_test.go"} {
		found := false
		for _, path := range paths {
			found = found || path == own
		}
		if !found {
			t.Errorf("%s is not among what git tracks: %v", own, paths)
		}
	}
}

// what git tracks is paths and nothing else: no empty name, no nought
// byte left from the list, and nothing from outside this folder.
func TestWhatGitTracksIsPathsAndNothingElse(t *testing.T) {
	paths, err := tracked()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if path == "" || strings.ContainsRune(path, 0) ||
			strings.HasPrefix(path, "/") || strings.HasPrefix(path, "..") {
			t.Errorf("%q is not a path under this folder", path)
		}
	}
}

// the table has a row a kind and a total, and the shares are of the
// lines.
func TestTheTableSharesTheLines(t *testing.T) {
	said := table([]count{{"go", 3, 75}, {"markdown", 1, 25}})
	want := "kind               files     lines   share\n" +
		"go                     3        75   75.0%\n" +
		"markdown               1        25   25.0%\n" +
		"total                  4       100  100.0%\n"
	if said != want {
		t.Errorf("the table is\n%s\nnot\n%s", said, want)
	}
}

// with no lines at all, every share is nought and nothing is divided
// by nought.
func TestWithNoLinesEveryShareIsNought(t *testing.T) {
	said := table([]count{{theRest, 2, 0}})
	if !strings.Contains(said, "everything else        2         0    0.0%") ||
		!strings.Contains(said, "total                  2         0    0.0%") ||
		strings.Contains(said, "NaN") {
		t.Errorf("the table is\n%s", said)
	}
	if empty := table(nil); !strings.Contains(empty, "total") ||
		strings.Contains(empty, "NaN") {
		t.Errorf("the empty table is\n%s", empty)
	}
}

// outside a git folder there is nothing to list, and tracked says git
// could not.
func TestTrackedOutsideAGitFolderSaysGitFailed(t *testing.T) {
	bare := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(bare))
	t.Chdir(bare)
	paths, err := tracked()
	if err == nil || paths != nil ||
		!strings.Contains(err.Error(), "git ls-files") {
		t.Errorf("tracked in a folder with no git gave %v, %v; want a "+
			"refusal naming git ls-files", paths, err)
	}
}
