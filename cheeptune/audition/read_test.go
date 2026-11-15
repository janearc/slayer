package audition

import (
	"path/filepath"
	"strings"
	"testing"
)

// The page is written from scores and sheets alone. A folder is shared by
// trees that do not all hold the same voicings, and a page that rendered
// every tune could be written by none of them.

// strangerVoicing is a voicing no tree has.
const strangerVoicing = "@no-such-voicing"

// strangerSheet is the small sheet naming the voicing no tree has.
var strangerSheet = strings.Replace(smallSheet, "@dmg-pad", strangerVoicing,
	1)

func TestThePageHasARowForATuneWhoseVoicingThisTreeLacks(t *testing.T) {
	dir := folder(t, "known", "stranger")
	write(t, dir, "stranger.cuts.json", strangerSheet)
	var tunes []*Tune
	for _, name := range []string{"known", "stranger"} {
		tune, err := Read(dir, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if tune.Piece != nil {
			t.Errorf("%s was rendered by Read", name)
		}
		tunes = append(tunes, tune)
	}
	page, err := Page(tunes)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-name="stranger"`,
		`src="stranger.mp4"`, `data-name="known"`, "small and quiet"} {
		if !strings.Contains(string(page), want) {
			t.Errorf("the page lacks %s", want)
		}
	}
}

func TestAFilmOfATuneWhoseVoicingIsMissingIsStillRefusedByName(t *testing.T) {
	dir := folder(t, "stranger")
	write(t, dir, "stranger.cuts.json", strangerSheet)
	_, err := Load(dir, "stranger")
	if err == nil || !strings.Contains(err.Error(), strangerVoicing) ||
		!strings.Contains(err.Error(), "stranger") {
		t.Errorf("Load gave %v, want the tune and the voicing named", err)
	}
	tune, err := Read(dir, "stranger")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "stranger.mp4")
	if err := Film(tune, out); err == nil ||
		!strings.Contains(err.Error(), "stranger") {
		t.Errorf("a film of a tune with no sound gave %v", err)
	}
	if _, err := Plate(tune); err == nil {
		t.Error("a plate of a tune with no sound was drawn")
	}
}

func TestReadStillRefusesAMalformedSheetAndABrokenScore(t *testing.T) {
	dir := folder(t, "small")
	if _, err := Read(dir, "small"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "small.cuts.json",
		strings.Replace(smallSheet, `"seconds":4`, `"seconds":9`, 1))
	if _, err := Read(dir, "small"); err == nil {
		t.Error("a sheet of the wrong length was read")
	}
	write(t, dir, "small.cuts.json", `{"group":`)
	if _, err := Read(dir, "small"); err == nil {
		t.Error("a sheet that is not json was read")
	}
	write(t, dir, "small.cuts.json", smallSheet)
	write(t, dir, "small.json", "{")
	if _, err := Read(dir, "small"); err == nil {
		t.Error("a broken score was read")
	}
	if _, err := Read(dir, "absent"); err == nil {
		t.Error("a tune with no file was read")
	}
}
