//go:build draw

package door

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --check says each tune's length, peak and voicing, by a folder given
// relative to the working directory.
func TestTunesChecksEachTune(t *testing.T) {
	dir := library(t)
	got := tuning(t, filepath.Dir(dir), filepath.Base(dir), "--check")
	if got.code != nothing || !strings.HasPrefix(got.said,
		"ok   small") || !strings.Contains(got.said, "@dmg-pad") {
		t.Errorf("tunes --check gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
}

// a tune that will not load is said, the others are
// still checked, and the check fails saying how many.
// anti-test of TestTunesChecksEachTune
func TestTunesCheckFailsOnATuneThatWillNotLoad(t *testing.T) {
	sheet, err := os.ReadFile(filepath.Join("testdata", "tunes",
		"small.cuts.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := library(t, "broken.json", "{", "broken.cuts.json",
		string(sheet))
	got := tuning(t, dir, dir, "--check")
	if got.code != one || !strings.Contains(got.said, "FAIL broken") ||
		!strings.Contains(got.said, "ok   small") ||
		!strings.Contains(got.wrongSaid, "1 of 2 tunes would not load") {
		t.Errorf("tunes --check gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
}

// --plates, --page and --films each write what they say beside the
// tunes, and a dry run of each says it and writes nothing.
func TestTunesWritesStillsThePageAndFilms(t *testing.T) {
	for _, each := range []struct{ flag, wrote string }{
		{"--plates", "small.png"},
		{"--page", "index.html"},
		{"--films", "small.mp4"},
	} {
		dir := library(t)
		dry := tuning(t, dir, dir, each.flag, "--dry-run")
		if dry.code != nothing || dry.said != "would write "+
			filepath.Join(dir, each.wrote)+"\n" {
			t.Errorf("%s --dry-run said %q %q", each.flag, dry.said,
				dry.wrongSaid)
		}
		if len(files(t, dir)) != pair {
			t.Errorf("%s --dry-run wrote %v", each.flag, files(t, dir))
		}
		got := tuning(t, dir, dir, each.flag, "small")
		if got.code != nothing || got.said != "wrote "+
			filepath.Join(dir, each.wrote)+"\n" {
			t.Errorf("%s said %q %q", each.flag, got.said, got.wrongSaid)
		}
		if info, err := os.Stat(filepath.Join(dir, each.wrote)); err != nil ||
			info.Size() == nothing {
			t.Errorf("%s wrote no %s: %v", each.flag, each.wrote, err)
		}
	}
	dir := library(t)
	if got := tuning(t, dir, dir, "--check", "--dry-run"); got.said !=
		"would write nothing\n" {
		t.Errorf("--check --dry-run said %q", got.said)
	}
	tuning(t, dir, dir, "--plates")
	if still := pictureOf(t, filepath.Join(dir, "small.png")); still.Bounds().
		Empty() {
		t.Error("the still is empty")
	}
}

// --plates with --out writes the stills in that folder, which it makes,
// and none beside the tunes; --sounds draws a studio voicing's sheet to
// --out, and its dry run writes nothing.
func TestTunesWritesStillsAndSoundsWhereOutSays(t *testing.T) {
	dir := library(t)
	stills := filepath.Join(t.TempDir(), "stills")
	got := tuning(t, dir, dir, "--plates", "--out", stills)
	if got.code != nothing {
		t.Fatalf("--plates --out gave %q", got.wrongSaid)
	}
	pictureOf(t, filepath.Join(stills, "small.png"))
	if len(files(t, dir)) != pair {
		t.Errorf("--plates --out wrote beside the tunes: %v", files(t, dir))
	}
	work := t.TempDir()
	dry := tuning(t, work, "--sounds", "@glass-bells", "--dry-run")
	if dry.said != "would write "+filepath.Join(work, outFolder,
		soundsName)+"\n" || len(files(t, work)) != nothing {
		t.Errorf("a dry --sounds said %q and wrote %v", dry.said,
			files(t, work))
	}
	got = tuning(t, work, "--sounds", "@glass-bells", "--out", "sheet.png")
	if got.code != nothing {
		t.Fatalf("--sounds gave %q", got.wrongSaid)
	}
	pictureOf(t, filepath.Join(work, "sheet.png"))
}
