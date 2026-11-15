package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// The script reads and every short in it is found: a script the walk
// package refuses would stop the stills from being labelled. The
// stills are not in the repository, so a short's still missing is the
// only refusal allowed here.
func TestTheScriptReads(t *testing.T) {
	out := filepath.Join(t.TempDir(), "field.png")
	err := run([]string{"magnetar", "magnetar-walk.yaml", t.TempDir(),
		"field", out}, io.Discard)
	if err != nil && !errors.Is(err, os.ErrNotExist) &&
		!strings.Contains(err.Error(), "magnetar-field.png") {
		t.Errorf("the script: %v", err)
	}
}

// anti-tests: too few arguments, and a short the script has not.
func TestTheCommandRefuses(t *testing.T) {
	if err := run([]string{"magnetar"}, io.Discard); !errors.Is(err,
		errUsage) {
		t.Errorf("no arguments: %v", err)
	}
	err := run([]string{"magnetar", "magnetar-walk.yaml", "", "sideways",
		"x.png"}, io.Discard)
	if err == nil {
		t.Error("a short the script has not was found")
	}
}

// the film's frames: the tune's length at the film's rate, glass-acid's
// 45 seconds at 24 a second, defined by the tune
const tuneFrames = 45 * filmRate

// The film's plan fills the tune exactly, and every sentence in it is
// a line of the scene's script marked for the film, inside the short.
func TestTheFilmPlanFillsTheTuneAndSpeaksTheScript(t *testing.T) {
	read, err := readPlan("magnetar-film.yaml")
	if err != nil {
		t.Fatal(err)
	}
	frames := read.Intro.Frames
	for _, segment := range read.Segments {
		frames += segment.Count
	}
	if frames != tuneFrames {
		t.Errorf("the film is %d frames, the tune %d", frames, tuneFrames)
	}
	filmed := map[string]bool{}
	for _, line := range named.MagnetarScript() {
		filmed[line.Say] = line.Film
	}
	short := float64(frames-read.Intro.Frames) / filmRate
	for _, line := range read.Lines {
		if !filmed[line.Say] {
			t.Errorf("%q is not a film line of the script", line.Say)
		}
		if line.From < 0 || line.Until > short+slack ||
			line.Until <= line.From {
			t.Errorf("%q is said %g to %g", line.Say, line.From,
				line.Until)
		}
	}
}

// slack is how far past the short a line may end: its times are
// written to four places; picked by the author of this scene,
// 2026-11-12
const slack = 1e-3

// anti-test: a segment that holds fewer pictures than the plan says is
// refused by name.
func TestAShortSegmentIsRefused(t *testing.T) {
	read, err := readPlan("magnetar-film.yaml")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, read.Segments[0].Frames),
		0o755); err != nil {
		t.Fatal(err)
	}
	_, _, err = read.source(dir)
	if err == nil || !strings.Contains(err.Error(),
		read.Segments[0].Frames) {
		t.Errorf("an empty segment: %v", err)
	}
}
