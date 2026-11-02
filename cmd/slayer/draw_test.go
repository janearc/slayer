package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/compare"
	"github.com/janearc/slayer/scene/thing"
)

// ball writes the ball's set file, changed by each pair of was and now,
// where the test keeps its files, and says where.
func ball(t *testing.T, changes ...string) string {
	t.Helper()
	source, err := os.ReadFile("testdata/ball.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := string(source)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(written, changes[i]) {
			t.Fatalf("%q is not in the ball", changes[i])
		}
		written = strings.Replace(written, changes[i], changes[i+1], 1)
	}
	return kept(t, "ball.yaml", written)
}

// kept writes a file where the test keeps its files, and says where.
func kept(t *testing.T, name, written string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// a set is drawn as the terminal shows it, as a real picture and in its
// own look, each named for the file and, when it is drawn at a time, the
// time, with the verdict beside them, into a folder made for them.
func TestASetIsDrawnAndJudged(t *testing.T) {
	out := filepath.Join(t.TempDir(), "made")
	honest, err := draw([]string{ball(t), "-k", "1", "-out", out})
	if err != nil || !honest {
		t.Fatalf("the ball is %v, %v", honest, err)
	}
	for _, suffix := range []string{"-text.png", "-real.png", "-look.png",
		"-verdict.json"} {
		if _, err := os.Stat(filepath.Join(out, "ball"+suffix)); err != nil {
			t.Error(err)
		}
	}
	if _, err := draw([]string{ball(t), "-k", "1", "-at", "1s", "-camera",
		"front", "-out", out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "ball-at-1s-real.png")); err != nil {
		t.Error(err)
	}
}

// a comparison whose worst pixel is past what an eye can see is not
// honest, and one within it is.
func TestTheVerdictIsSaidInAWord(t *testing.T) {
	if said := judged(compare.Report{Worst: 1}); said != "not honest" {
		t.Errorf("a worst of one is %q", said)
	}
	if said := judged(compare.Report{}); said != "honest" {
		t.Errorf("a worst of nought is %q", said)
	}
}

// what cannot be drawn is refused, saying why: no file named, a flag
// for a file, a flag slayer does not know, a -k under one, a file not
// there or not a set, a set with no world or one the tracer cannot
// hold, a field judged too coarsely, a camera not there or off an asset
// slayer cannot place, and pictures or a verdict that cannot be written.
func TestWhatCannotBeDrawnIsRefused(t *testing.T) {
	file := kept(t, "file", "")
	taken := t.TempDir()
	for _, name := range []string{"ball-real.png", "ball-verdict.json"} {
		if err := os.MkdirAll(filepath.Join(taken, "real", name),
			0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(taken, "verdict",
		"ball-verdict.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	quasar := ball(t, "    spheres:", "    bodies:\n      hole:\n"+
		"        template: quasar\n        middle: [0, 100, 0]\n"+
		"        detail: 1\n    spheres:")
	for _, refusal := range []struct {
		args []string
		said string
	}{
		{nil, "which set?"},
		{[]string{"-k", "1"}, "which set?"},
		{[]string{ball(t), "-wobble"}, "flag provided but not defined"},
		{[]string{ball(t), "-k", "0"}, "-k is at least 1, not 0"},
		{[]string{filepath.Join(taken, "nowhere.yaml")}, "no such file"},
		{[]string{ball(t, "scene:", "scenery:")}, "scenery is not a key"},
		{[]string{kept(t, "bare.yaml", "scene:\n  name: x\n")},
			"says no world to draw"},
		{[]string{ball(t, "colour: {swatch: clay}",
			"glow: {temperature: 10, luminance: 1}")},
			"ball.yaml: material clay: a black body at 10 K"},
		{[]string{quasar, "-k", "1"},
			"has a field, and a reference is never coarser than what it " +
				"judges: -k is at least 2, not 1"},
		{[]string{ball(t), "-camera", "back"}, "the set has no camera back"},
		{[]string{ball(t), "-camera", "off"},
			`camera off: stands off "the box", which is not on the set`},
		{[]string{ball(t), "-k", "1", "-out", filepath.Join(file, "under")},
			"not a directory"},
		{[]string{ball(t), "-k", "1", "-out", filepath.Join(taken, "real")},
			"ball-real.png"},
		{[]string{ball(t), "-k", "1", "-out",
			filepath.Join(taken, "verdict")}, "ball-verdict.json"},
	} {
		args := refusal.args
		// a case that names a file draws, if it is not refused, where
		// the test keeps its files and never beside the code; its own
		// -out, later, wins
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			args = append([]string{args[0], "-out", t.TempDir()},
				args[1:]...)
		}
		if _, err := draw(args); err == nil ||
			!strings.Contains(err.Error(), refusal.said) {
			t.Errorf("%v: %v, wanted %q", refusal.args, err, refusal.said)
		}
	}
}

// a world with a field is never judged by a reference coarser than
// twice, and one without by one no coarser than itself.
func TestFinenessIsTwiceForAField(t *testing.T) {
	plain := thing.NewWorld()
	if got := Fineness(plain); got != 1 {
		t.Errorf("a plain world's fineness is %d", got)
	}
	plain.AddField(thing.Field{Mass: 1})
	if got := Fineness(plain); got != 2 {
		t.Errorf("a world with a field's fineness is %d", got)
	}
}
