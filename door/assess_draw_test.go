//go:build draw

package door

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// a set and a film job each will draw: assess says so, what it would
// draw and each file draw would write, and writes nothing.
func TestAssessSaysAFileWillDraw(t *testing.T) {
	directory := t.TempDir()
	got := assessed(t, directory, widow(t))
	want := "widow.yaml will draw, as a still; nothing was drawn\n" +
		"draw would write picture " + filepath.Join(directory, outFolder,
		"widow.png") + "\n"
	if got.code != nothing || got.said != want {
		t.Errorf("assess said %q %q", got.said, got.wrongSaid)
	}
	film := written(t, directory, "film.yaml", filmJob(t,
		"outputs: {note: a film}\n"))
	got = assessed(t, directory, film)
	if got.code != nothing || !strings.HasPrefix(got.said, "film.yaml "+
		"will draw, as a film of 3 frames") || !strings.Contains(got.said,
		"draw would not write the note: no renders home") {
		t.Errorf("assess of a film said %q %q", got.said, got.wrongSaid)
	}
	if !slices.Equal(files(t, directory), []string{"film.yaml"}) {
		t.Errorf("assess wrote %v", files(t, directory))
	}
}

// each thing draw would refuse is said, after the file's
// name, as the reason it will not draw.
// anti-test of TestAssessSaysAFileWillDraw
func TestAssessSaysWhyAFileWillNotDraw(t *testing.T) {
	directory := t.TempDir()
	for _, each := range []struct{ name, text, says string }{
		{"list.yaml", "- a\n- b\n", "is not a job, a plan or a set"},
		{"named.yaml", jobLine + "scene: {named: lake}\n",
			`no such scene: "lake"`},
		{"stock.yaml", stillJob(t, "stock: {name: nostock}\n"),
			"value must be one of"},
		{"sound.yaml", filmJob(t,
			"sound: {tune: \"@crypt\", voicing: \"@crypt-dmg\"}\n"+
				"outputs: {frames: film}\n"),
			"the job says sound and writes no movie"},
	} {
		path := written(t, directory, each.name, each.text)
		got := assessed(t, directory, path)
		if got.code != one || !strings.Contains(got.wrongSaid,
			each.name+" will not draw: ") || !strings.Contains(
			got.wrongSaid, each.says) {
			t.Errorf("%s gave %d: %q", each.name, got.code, got.wrongSaid)
		}
	}
	got := assessed(t, directory, widow(t), "--width", "x")
	if !strings.Contains(got.wrongSaid, `widow.yaml will not draw: `+
		`--width takes a whole number, not "x"`) {
		t.Errorf("a bad width gave %q", got.wrongSaid)
	}
	got = assessed(t, directory, "gone.yaml")
	if got.code != one || !strings.Contains(got.wrongSaid, "gone.yaml "+
		"will not draw") {
		t.Errorf("a file not there gave %q", got.wrongSaid)
	}
}

// a set is judged as the terminal shows it, as a real picture and in
// its own look, each named for the file and, when it is drawn at a
// time, the time, with the verdict beside them, into a folder made for
// them. Moved from slayer set.
func TestAssessJudgesASet(t *testing.T) {
	out := filepath.Join(t.TempDir(), "made")
	got := assessed(t, t.TempDir(), ball(t), "--judge", "--k", "1",
		"--out", out)
	if got.code != nothing || !strings.HasPrefix(got.said,
		"set ball at 0s: honest.") {
		t.Fatalf("the ball gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	for _, suffix := range []string{"-text.png", "-real.png", "-look.png",
		"-verdict.json"} {
		if _, err := os.Stat(filepath.Join(out, "ball"+suffix)); err != nil {
			t.Error(err)
		}
	}
	var kept struct{ Honest bool }
	raw, err := os.ReadFile(filepath.Join(out, "ball-verdict.json"))
	if err != nil || json.Unmarshal(raw, &kept) != nil || !kept.Honest {
		t.Errorf("the verdict kept is %q, %v", raw, err)
	}
	got = assessed(t, t.TempDir(), ball(t), "--judge", "--k", "1", "--at",
		"1s", "--camera", "front", "--out", out)
	if got.code != nothing {
		t.Fatalf("the ball at 1s gave %q", got.wrongSaid)
	}
	if _, err := os.Stat(filepath.Join(out, "ball-at-1s-real.png")); err != nil {
		t.Error(err)
	}
}

// what cannot be judged is refused, saying why: a flag assess does not
// know, a --k under one, a file not there or not a set, a set with no
// world or one the tracer cannot hold, a field judged too coarsely, a
// camera not there or off an asset slayer cannot place, bad values,
// and pictures or a verdict that cannot be written. Moved from slayer
// set.
// anti-test of TestAssessJudgesASet
// anti-test of TestAssessJudgesIntoOutAndDryRunsWritingNothing
func TestWhatCannotBeJudgedIsRefused(t *testing.T) {
	file := written(t, t.TempDir(), "file", "")
	taken := t.TempDir()
	for _, name := range []string{"ball-real.png", "ball-verdict.json"} {
		if err := os.MkdirAll(filepath.Join(taken, "real", name),
			folderMode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(taken, "verdict",
		"ball-verdict.json"), folderMode); err != nil {
		t.Fatal(err)
	}
	quasar := ball(t, "    spheres:", "    bodies:\n      hole:\n"+
		"        template: quasar\n        middle: [0, 100, 0]\n"+
		"        detail: 1\n    spheres:")
	for _, refusal := range []struct {
		args []string
		said string
	}{
		{[]string{ball(t), "--wobble"}, "unknown flag: --wobble"},
		{[]string{ball(t), "--k", "0"}, "--k is at least 1, not 0"},
		{[]string{ball(t), "--k", "x"}, `--k takes a whole number`},
		{[]string{ball(t), "--samples", "x"}, `--samples takes a whole`},
		{[]string{ball(t), "--at", "x"}, `--at takes a time such as 20s`},
		{[]string{filepath.Join(taken, "nowhere.yaml")}, "no such file"},
		{[]string{ball(t, "scene:", "scenery:")}, "scenery is not a key"},
		{[]string{written(t, t.TempDir(), "bare.yaml",
			"scene:\n  name: x\n")}, "says no world to draw"},
		{[]string{ball(t, "colour: {swatch: clay}",
			"glow: {temperature: 10, luminance: 1}")},
			"ball.yaml: material clay: a black body at 10 K"},
		{[]string{quasar, "--k", "1"}, "has a field, and a reference is " +
			"never coarser than what it judges: --k is at least 2, not 1"},
		{[]string{ball(t), "--camera", "back"}, "the set has no camera back"},
		{[]string{ball(t), "--camera", "off"},
			`camera off: stands off "the box", which is not on the set`},
		{[]string{ball(t), "--k", "1", "--out", filepath.Join(file,
			"under")}, "not a directory"},
		{[]string{ball(t), "--k", "1", "--out", filepath.Join(taken,
			"real")}, "ball-real.png"},
		{[]string{ball(t), "--k", "1", "--out", filepath.Join(taken,
			"verdict")}, "ball-verdict.json"},
	} {
		// a case that names no --out judges, if it is not refused,
		// where the test keeps its files and never beside the code
		args := append([]string{"--judge"}, refusal.args...)
		if !slices.Contains(args, "--out") {
			args = append(args, "--out", t.TempDir())
		}
		got := assessed(t, t.TempDir(), args...)
		if got.code == nothing || !strings.Contains(got.wrongSaid,
			refusal.said) {
			t.Errorf("%v: %q, wanted %q", refusal.args, got.wrongSaid,
				refusal.said)
		}
	}
}

// smoke 1 judged twice: drawn the first time and served from the cache
// the second, honest both times, with its two pictures and its verdict
// copied to --out. Moved from smokecheck.
func TestAssessJudgesASmokeAndServesItAgain(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	out := t.TempDir()
	for i, want := range []string{"drew 1, served 0", "drew 0, served 1"} {
		got := assessed(t, t.TempDir(), "--smokes", "1", "--k", "1",
			"--out", out)
		if got.code != nothing || !strings.Contains(got.said, want) ||
			!strings.Contains(got.said, "honest") {
			t.Fatalf("run %d gave %d: %q %q", i, got.code, got.said,
				got.wrongSaid)
		}
	}
	copiedSmoke(t, out)
	got := assessed(t, t.TempDir(), "--smokes", "1", "--small", "--fresh")
	if got.code != nothing || !strings.Contains(got.said, "drew 1, "+
		"served 0") {
		t.Errorf("small and fresh gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
}

// smokeCopies is what a smoke judged copies to --out: its two pictures
// and its verdict. Defined, by render/smokejudge.
const smokeCopies = 3

// copiedSmoke holds what a smoke judged copied to a folder: two
// pictures that decode, each at least the terminal's size, and a
// verdict that parses and compared at least half its pixels. Moved from
// smokecheck.
func copiedSmoke(t *testing.T, out string) {
	t.Helper()
	copied := files(t, out)
	if len(copied) != smokeCopies {
		t.Fatalf("copied %v", copied)
	}
	pictures := nothing
	for _, name := range copied {
		path := filepath.Join(out, name)
		if filepath.Ext(name) == pictureExtension {
			size := pictureOf(t, path).Bounds()
			if size.Dx() < textWide || size.Dy() < textHigh {
				t.Errorf("%s is %v", name, size)
			}
			pictures++
			continue
		}
		var report struct{ Compared, Pixels int }
		raw, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(raw, &report) != nil ||
			report.Pixels == nothing ||
			report.Compared < report.Pixels/pair {
			t.Errorf("the verdict %s is %q, %v", name, raw, err)
		}
	}
	if pictures != pair {
		t.Errorf("copied %d pictures, not two", pictures)
	}
}

// assess refuses what draw refuses before a film's first frame, by
// draw's own words: a tune there is not, an odd width the movie cannot
// take, a manifest that does not read, a folder of pictures with no
// manifest, and a picture the manifest does not list. The anti-test: a
// tune there is, and a frames folder not there, will draw, and neither
// the folder nor the movie's is made.
// anti-test of TestAssessSaysAFileWillDraw
func TestAssessRefusesWhatAFilmRefusesBeforeItsFirstFrame(t *testing.T) {
	directory := t.TempDir()
	tune := func(score string) string {
		return filmJob(t, "sound: {tune: \""+score+"\", voicing: "+
			"\"@crypt-dmg\"}\n")
	}
	framed := func(folder string) string {
		return filmJob(t, "outputs: {frames: "+folder+"}\n")
	}
	unread := filepath.Join(directory, "unread")
	written(t, mkdir(t, unread), manifestName, "{")
	pictured := filepath.Join(directory, "pictured")
	written(t, mkdir(t, pictured), "a.png", "a picture")
	unlisted := filepath.Join(directory, "unlisted")
	written(t, mkdir(t, unlisted), manifestName, `{"key": "x"}`)
	written(t, unlisted, "a.png", "a picture")
	odd := strings.Replace(filmJob(t, ""), fmt.Sprintf("width: %d",
		number(t, "width")), fmt.Sprintf("width: %d",
		number(t, "width")-one), one)
	for _, each := range []struct{ name, text, says string }{
		{"nonesuch.yaml", tune("@nonesuch"), "nonesuch"},
		{"odd.yaml", odd, "is odd: H.264 halves the colour across"},
		{"unread.yaml", framed(unread), "cannot be read"},
		{"pictured.yaml", framed(pictured),
			"holds pictures and no manifest.json"},
		{"unlisted.yaml", framed(unlisted), "holds a.png, which its " +
			"manifest.json does not list"},
	} {
		path := written(t, directory, each.name, each.text)
		got := assessed(t, directory, path)
		if got.code != one || !strings.Contains(got.wrongSaid,
			each.name+" will not draw: ") || !strings.Contains(
			got.wrongSaid, each.says) {
			t.Errorf("%s gave %d: %q", each.name, got.code, got.wrongSaid)
		}
		drawn := running(Tool(envIn(t, context.Background(), directory)),
			"draw", path)
		if drawn.code != one || !strings.Contains(drawn.wrongSaid,
			each.says) {
			t.Errorf("draw of %s gave %d: %q", each.name, drawn.code,
				drawn.wrongSaid)
		}
	}
	fresh := filepath.Join(directory, "fresh")
	movies := filepath.Join(directory, "movies")
	good := written(t, directory, "good.yaml", tune("@crypt")+
		"outputs: {frames: "+fresh+", movie: "+filepath.Join(movies,
		"good.mp4")+"}\n")
	got := assessed(t, directory, good)
	if got.code != nothing || !strings.Contains(got.said, "good.yaml "+
		"will draw") {
		t.Errorf("a good film gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	for _, folder := range []string{fresh, movies} {
		if _, err := os.Stat(folder); err == nil {
			t.Errorf("assess made %s", folder)
		}
	}
}

// frames of an earlier version of this job are said, as frames draw
// would redraw, and left exactly as they are: the manifest and the
// frame are still there, unchanged, after assess.
func TestAssessSaysStaleFramesAndLeavesThem(t *testing.T) {
	directory := t.TempDir()
	stale := mkdir(t, filepath.Join(directory, "stale"))
	record := `{"key": "an earlier version", "frames": ` +
		`{"frame-000000.png": "abc"}}`
	written(t, stale, manifestName, record)
	written(t, stale, "frame-000000.png", "an old frame")
	path := written(t, directory, "film.yaml", filmJob(t,
		"outputs: {frames: "+stale+"}\n"))
	got := assessed(t, directory, path)
	if got.code != nothing || !strings.Contains(got.said, "the frames "+
		"there are from an earlier version of this job; draw would "+
		"redraw them\nfilm.yaml will draw") {
		t.Errorf("stale frames gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	kept, err := os.ReadFile(filepath.Join(stale, manifestName))
	if err != nil || string(kept) != record {
		t.Errorf("the manifest is now %q, %v", kept, err)
	}
	frame, err := os.ReadFile(filepath.Join(stale, "frame-000000.png"))
	if err != nil || string(frame) != "an old frame" {
		t.Errorf("the frame is now %q, %v", frame, err)
	}
}
