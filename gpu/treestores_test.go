//go:build draw

package gpu

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

func TestFormOfKnowsTheThreeWays(t *testing.T) {
	for said, want := range map[string]string{"": inOneFile,
		inOneFile: inOneFile, asFiles: asFiles, perProcess: perProcess} {
		if got, err := formOf(said); err != nil || got != want {
			t.Errorf("%q: %q, %v", said, got, err)
		}
	}
}

func TestFormOfRefusesAnotherWay(t *testing.T) {
	if _, err := formOf("sqlite"); err == nil {
		t.Fatal("a way there is not was taken")
	}
}

// shelfAs is a shelf in a fresh folder kept a way.
func shelfAs(t *testing.T, form string) *treeShelf {
	t.Helper()
	kept := shelfIn(t, 1)
	kept.form = form
	return kept
}

func TestATreeKeptAsAFileIsTheTreeGrown(t *testing.T) {
	faces := someFaces(cachedFaces, 21)
	want := grownFresh(faces)
	kept := shelfAs(t, asFiles)
	kept.grown(slices.Clone(faces))
	got := kept.grown(slices.Clone(faces))
	if !reflect.DeepEqual(got, want) || kept.tally.kept != 1 ||
		kept.tally.grown != 1 {
		t.Fatalf("tally %+v", kept.tally)
	}
}

func TestATreeOfAnotherFaceIsAnotherFile(t *testing.T) {
	kept := shelfAs(t, asFiles)
	kept.grown(someFaces(cachedFaces, 22))
	kept.grown(someFaces(cachedFaces, 23))
	if kept.tally.kept != 0 || kept.tally.grown != 2 {
		t.Fatalf("tally %+v", kept.tally)
	}
}

func TestAProcessReadsItsOwnTreesBeforeAMerge(t *testing.T) {
	faces := someFaces(cachedFaces, 24)
	kept := shelfAs(t, perProcess)
	kept.grown(slices.Clone(faces))
	got := kept.grown(slices.Clone(faces))
	if !reflect.DeepEqual(got, grownFresh(faces)) || kept.tally.kept != 1 {
		t.Fatalf("tally %+v", kept.tally)
	}
	if _, err := os.Stat(kept.path); err == nil {
		t.Fatal("the main file was written before a merge")
	}
}

// ended is the number of a process that has run and ended.
func ended(t *testing.T) int {
	t.Helper()
	done := exec.Command("true")
	if err := done.Run(); err != nil {
		t.Fatal(err)
	}
	return done.Process.Pid
}

// grownBy is a shelf's own file renamed as a process's of that number.
func grownBy(t *testing.T, kept *treeShelf, pid int, seed uint64) string {
	t.Helper()
	kept.grown(someFaces(cachedFaces, seed))
	as := kept.path + ownSuffix + strconv.Itoa(pid)
	if err := os.Rename(kept.own(), as); err != nil {
		t.Fatal(err)
	}
	return as
}

func TestAMergeFoldsEndedRunsIntoANewMainFile(t *testing.T) {
	kept := shelfAs(t, perProcess)
	gone := grownBy(t, kept, ended(t), 25)
	said, err := mergeInto(kept.path, os.Getpid(), shortWait, nil)
	if err != nil || said != "merged 1 runs' trees" {
		t.Fatalf("%q, %v", said, err)
	}
	if _, err := os.Stat(gone); err == nil {
		t.Fatal("a merged run's file was kept")
	}
	reader := shelfAs(t, perProcess)
	reader.path = kept.path
	reader.grown(someFaces(cachedFaces, 25))
	if reader.tally.kept != 1 {
		t.Fatalf("the merged tree was not read: %+v", reader.tally)
	}
}

func TestAMergeLeavesARunningProcesssFileAlone(t *testing.T) {
	kept := shelfAs(t, perProcess)
	running := grownBy(t, kept, os.Getppid(), 26)
	said, err := mergeInto(kept.path, os.Getpid(), shortWait, nil)
	if err != nil || said != "nothing to merge" {
		t.Fatalf("%q, %v", said, err)
	}
	if _, err := os.Stat(running); err != nil {
		t.Fatal("a running process's file was taken")
	}
}

func TestAMergeThatFailsBeforeItsRenameLeavesTheMainFileWhole(t *testing.T) {
	kept := shelfAs(t, perProcess)
	grownBy(t, kept, ended(t), 27)
	if _, err := mergeInto(kept.path, os.Getpid(), shortWait,
		nil); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(kept.path)
	if err != nil {
		t.Fatal(err)
	}
	second := grownBy(t, kept, ended(t), 28)
	_, err = mergeInto(kept.path, os.Getpid(), shortWait, func() error {
		return errors.New("killed")
	})
	after, _ := os.ReadFile(kept.path)
	if err == nil || !bytes.Equal(before, after) {
		t.Fatalf("a failed merge changed the main file: %v", err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatal("a failed merge removed the run it did not merge")
	}
	left, _ := filepath.Glob(kept.path + mergeSuffix + "*")
	if len(left) != 0 {
		t.Fatalf("a failed merge left %v", left)
	}
}

func TestAMergeInHandIsNotJoined(t *testing.T) {
	kept := shelfAs(t, perProcess)
	grownBy(t, kept, ended(t), 29)
	if err := os.WriteFile(kept.path+mergeLock, nil, fileMode); err != nil {
		t.Fatal(err)
	}
	said, err := mergeInto(kept.path, os.Getpid(), shortWait, nil)
	if err != nil || said != "another merge is in hand" {
		t.Fatalf("%q, %v", said, err)
	}
}
