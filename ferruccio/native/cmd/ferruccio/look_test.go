package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/look"
)

// writeLooks makes a looks file beside a set named set.yaml in a new
// directory and returns the set's path.
func writeLooks(t *testing.T, blocks ...string) string {
	t.Helper()
	directory := t.TempDir()
	set := filepath.Join(directory, "set.yaml")
	for _, block := range blocks {
		if err := look.Append(look.PathFor(set), block); err != nil {
			t.Fatal(err)
		}
	}
	return set
}

func TestLookPrintsTheNewestLookOnly(t *testing.T) {
	set := writeLooks(t, "---\nlook: one\n", "---\nlook: two\n")
	code, output := runAtelier("look", set)
	if code != 0 || output != "---\nlook: two\n" {
		t.Fatalf("look returned %d: %q", code, output)
	}
}

func TestLookAllPrintsEveryLookOldestFirst(t *testing.T) {
	set := writeLooks(t, "---\nlook: one\n", "---\nlook: two\n")
	code, output := runAtelier("look", "--all", set)
	if code != 0 || output != "---\nlook: one\n---\nlook: two\n" {
		t.Fatalf("look --all returned %d: %q", code, output)
	}
}

func TestLookFailsNamingTheFileWhenThereAreNoLooks(t *testing.T) {
	set := filepath.Join(t.TempDir(), "set.yaml")
	code, output := runAtelier("look", set)
	if code == 0 || !strings.Contains(output, "set.looks.txt") {
		t.Fatalf("look at a set with no looks returned %d: %s", code, output)
	}
}

func TestLookNeedsExactlyOneSet(t *testing.T) {
	for _, args := range [][]string{{"look"}, {"look", "a.yaml", "b.yaml"}} {
		if code, output := runAtelier(args...); code == 0 || !strings.Contains(output, "one set file") {
			t.Fatalf("%v returned %d: %s", args, code, output)
		}
	}
}

func TestLookFindsAnExamplesLooksInTheEvidenceDirectory(t *testing.T) {
	evidence := t.TempDir()
	if err := look.Append(filepath.Join(evidence, "fizz.looks.txt"), "---\nlook: kept\n"); err != nil {
		t.Fatal(err)
	}
	example := filepath.Join(t.TempDir(), "fizz.yaml")
	if code, _ := runAtelier("look", example); code == 0 {
		t.Fatal("looks that are not beside the set were found without being asked for")
	}
	code, output := runAtelier("look", "--evidence", evidence, example)
	if code != 0 || output != "---\nlook: kept\n" {
		t.Fatalf("look in the evidence directory returned %d: %q", code, output)
	}
}

func TestLookWritesNothing(t *testing.T) {
	set := writeLooks(t, "---\nlook: one\n")
	before, err := os.ReadDir(filepath.Dir(set))
	if err != nil {
		t.Fatal(err)
	}
	runAtelier("look", "--all", set)
	after, _ := os.ReadDir(filepath.Dir(set))
	if len(before) != len(after) {
		t.Fatalf("look changed the directory: %d files, then %d", len(before), len(after))
	}
}

func Example_look() {
	directory, err := os.MkdirTemp("", "ferruccio-look-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(directory)
	set := filepath.Join(directory, "street.yaml")
	if err := look.Append(look.PathFor(set), "---\nlook: 2026-11-20T01:52:10Z\n"); err != nil {
		panic(err)
	}
	code, output := runAtelier("look", set)
	fmt.Print(code, " ", output)
	// Output:
	// 0 ---
	// look: 2026-11-20T01:52:10Z
}
