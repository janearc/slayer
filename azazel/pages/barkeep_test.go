package pages

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The times of a pace's counts, in seconds from its first.
const (
	fiveMinutes   = 300 // defined: half the window
	elevenMinutes = 660 // defined: past the window from the first count
)

// TestPaceIsTakenOverTheLastTenMinutes: thirty pictures in the first
// five minutes and none in the ten since say nothing is being made.
func TestPaceIsTakenOverTheLastTenMinutes(t *testing.T) {
	var progress pace
	start := fixtureNoon.Unix()
	progress.add(start, 0, 100)
	progress.add(start+fiveMinutes, 30, 100)
	perMinute, left := progress.add(start+fiveMinutes+elevenMinutes, 30, 100)
	if perMinute != 0 || left != unknownLeft {
		t.Fatal("an old pace was kept:", perMinute, left)
	}
}

// TestPaceWithinTheWindowSaysTheTimeLeft is the anti-test: thirty
// pictures in five minutes is six a minute, and seventy left is
// seven hundred seconds.
func TestPaceWithinTheWindowSaysTheTimeLeft(t *testing.T) {
	var progress pace
	start := fixtureNoon.Unix()
	progress.add(start, 0, 100)
	perMinute, left := progress.add(start+fiveMinutes, 30, 100)
	if perMinute != 6 || left != 700 {
		t.Fatal(perMinute, left)
	}
	if _, left = progress.add(start+fiveMinutes, 100, 100); left != 0 {
		t.Fatal("a finished render has time left:", left)
	}
}

// lineAt writes a barkeep line as someone else's watcher would, written
// at the time given.
func lineAt(t *testing.T, path string, when int64) string {
	t.Helper()
	line := "5 10 1 300 " + strconv.FormatInt(when, 10) + "\n"
	if err := os.WriteFile(path, []byte(line), listing); err != nil {
		t.Fatal(err)
	}
	return line
}

// TestLiveLineOfAnotherIsNotFree: a line younger than two minutes that
// barkeep did not write is someone's live render.
func TestLiveLineOfAnotherIsNotFree(t *testing.T) {
	keeper := barkeep{path: filepath.Join(t.TempDir(), progressName)}
	now := fixtureNoon.Unix()
	lineAt(t, keeper.path, now-liveLine+1)
	if keeper.free(now) {
		t.Fatal("someone's live line was free")
	}
}

// TestOldOwnOrMissingLinesAreFree is the anti-test: no line, an old one,
// and barkeep's own are all free to write.
func TestOldOwnOrMissingLinesAreFree(t *testing.T) {
	keeper := barkeep{path: filepath.Join(t.TempDir(), progressName)}
	now := fixtureNoon.Unix()
	if !keeper.free(now) {
		t.Fatal("no line was not free")
	}
	lineAt(t, keeper.path, now-liveLine)
	if !keeper.free(now) {
		t.Fatal("an old line was not free")
	}
	if err := keeper.write(1, 2, 0, unknownLeft, now-1); err != nil {
		t.Fatal(err)
	}
	if !keeper.free(now) || read(t, keeper.path) != "1 2 0 -1 "+
		strconv.FormatInt(now-1, 10)+"\n" {
		t.Fatal("barkeep's own line was not its own")
	}
}
