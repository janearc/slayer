//go:build draw

package door

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/hillside"
	"github.com/janearc/slayer/job"
)

// a still job writes its picture, equal to the session's frame developed
// by the scene's tone with its hud drawn.
func TestDrawStillIsTheSessionsPicture(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml",
		stillJob(t, "outputs: {picture: still.png}\n"))
	said := drew(t, envIn(t, context.Background(), directory), path)
	picture := filepath.Join(directory, "still.png")
	if !strings.Contains(said, "wrote "+picture) {
		t.Errorf("the still's line is not said: %q", said)
	}
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := hillside.Open(context.Background(), wanted,
		hillside.Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	frame, err := session.Frame(context.Background(), nothing,
		hillside.Ask{})
	if err != nil {
		t.Fatal(err)
	}
	want := session.Tone().Develop(frame.Light())
	if err := frame.Overlay(want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pictureOf(t, picture).Pix, want.Pix) {
		t.Error("the still is not the session's developed frame")
	}
}

// a still with raw light writes it beside the picture,
// and one without writes no raw file.
// anti-test of TestDrawStillIsTheSessionsPicture
func TestDrawStillWritesRawOnlyWhenAsked(t *testing.T) {
	directory := t.TempDir()
	plain := written(t, directory, "plain.yaml",
		stillJob(t, "outputs: {picture: plain.png}\n"))
	drew(t, envIn(t, context.Background(), directory), plain)
	if got := files(t, directory); !slices.Equal(got,
		[]string{"plain.png", "plain.yaml"}) {
		t.Errorf("a still without raw wrote %v", got)
	}
	rawed := written(t, directory, "rawed.yaml", stillJob(t,
		"outputs: {picture: rawed.png, raw: rawed.pfm}\n"))
	drew(t, envIn(t, context.Background(), directory), rawed)
	if _, err := os.Stat(filepath.Join(directory,
		"rawed.pfm")); err != nil {
		t.Errorf("a still with raw did not write it: %v", err)
	}
}

// a film writes its frames and a manifest; a second run draws none; a
// deleted frame is drawn again; a changed job is refused.
func TestDrawFilmResumes(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n"))
	env := envIn(t, context.Background(), directory)
	frames := filmFrames(t)
	first := drew(t, env, path)
	folder := filepath.Join(directory, "film")
	if got := len(files(t, folder)); got != frames+one {
		t.Errorf("the film wrote %d files, not %d frames and a manifest",
			got, frames)
	}
	if !strings.Contains(first, fmt.Sprintf("drew %d frames and kept 0",
		frames)) {
		t.Errorf("the first run did not draw every frame: %q", first)
	}
	again := drew(t, env, path)
	if !strings.Contains(again, "nothing to draw") {
		t.Errorf("the second run drew again: %q", again)
	}
	gone := filepath.Join(folder, fmt.Sprintf(frameName, one))
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	third := drew(t, env, path)
	if !strings.Contains(third, fmt.Sprintf("drew 1 frames and kept %d",
		frames-one)) {
		t.Errorf("a deleted frame was not drawn again: %q", third)
	}
	changed := drew(t, env, path, "--rays", "2")
	if !strings.Contains(changed, "from an earlier version of this job") ||
		!strings.Contains(changed, fmt.Sprintf("drew %d frames and kept 0",
			frames)) {
		t.Errorf("a changed job did not draw again: %q", changed)
	}
}

// a frame whose bytes changed since it was written is
// drawn again, never kept; and a folder of pictures that is no run's is
// refused, its pictures left alone.
// anti-test of TestDrawFilmResumes
func TestDrawFilmKeepsOnlyWhatItWrote(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n"))
	env := envIn(t, context.Background(), directory)
	drew(t, env, path)
	folder := filepath.Join(directory, "film")
	written(t, folder, fmt.Sprintf(frameName, nothing), "not a picture")
	again := drew(t, env, path)
	if !strings.Contains(again, "drew 1 frames") {
		t.Errorf("a changed frame was kept: %q", again)
	}
	stranger := filepath.Join(directory, "stranger")
	if err := os.MkdirAll(stranger, folderMode); err != nil {
		t.Fatal(err)
	}
	written(t, stranger, "holiday.png", "someone's")
	refused := running(Tool(env), "draw", path, "--out", stranger)
	if refused.code == nothing ||
		!strings.Contains(refused.wrongSaid, "holds pictures") {
		t.Errorf("a stranger's folder was drawn into: %q",
			refused.wrongSaid)
	}
	if got := files(t, stranger); !slices.Equal(got,
		[]string{"holiday.png"}) {
		t.Errorf("the stranger's folder holds %v", got)
	}
}

// a title is drawn on the frames inside its span and on no other: of a
// two-second film at three a second, with a frame on each side of each
// end of the span, only the two frames inside it differ from the
// untitled film's.
func TestDrawTitlesInsideTheirSpan(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	twice := func(text string) string {
		return strings.Replace(text, "duration: {value: 1,",
			"duration: {value: 2,", one)
	}
	plain := written(t, directory, "plain.yaml",
		twice(filmJob(t, "outputs: {frames: plain}\n")))
	titled := written(t, directory, "titled.yaml", twice(filmJob(t,
		fmt.Sprintf("titles:\n  rows: 4\n  elements:\n    name:\n"+
			"      kind: text\n      words: widow\n      ink: ink\n"+
			"      across: 0.5\n      down: 0.5\n      align: centre\n"+
			"      from: {value: %g, unit: s}\n"+
			"      until: {value: %g, unit: s}\n"+
			"outputs: {frames: titled}\n",
			duration(t, "title-from").Seconds(),
			duration(t, "title-until").Seconds()))))
	drew(t, env, plain)
	drew(t, env, titled)
	at := func(folder string, index int) []byte {
		return pictureOf(t, filepath.Join(directory, folder,
			fmt.Sprintf(frameName, index))).Pix
	}
	rate := number(t, "fps")
	for index := range filmFrames(t) * pair {
		when := filmTime(index, rate)
		inside := when >= duration(t, "title-from") &&
			when < duration(t, "title-until")
		// anti-test of TestDrawTitlesInsideTheirSpan
		differs := !bytes.Equal(at("plain", index), at("titled", index))
		if differs != inside {
			t.Errorf("frame %d at %s: titled %v, inside the span %v",
				index, when, differs, inside)
		}
	}
}

// a stock changes the picture, and draws the same picture again on a
// second run.
func TestDrawStockChangesThePictureTheSameEachTime(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	plain := written(t, directory, "plain.yaml",
		stillJob(t, "outputs: {picture: plain.png}\n"))
	stocked := written(t, directory, "stocked.yaml", stillJob(t,
		"stock: {name: lantern, seed: 7}\n"+
			"outputs: {picture: stocked.png}\n"))
	drew(t, env, plain)
	drew(t, env, stocked)
	once := pictureOf(t, filepath.Join(directory, "stocked.png")).Pix
	if bytes.Equal(once, pictureOf(t, filepath.Join(directory,
		"plain.png")).Pix) {
		t.Error("the stock did not change the picture")
	}
	drew(t, env, stocked)
	if !bytes.Equal(once, pictureOf(t, filepath.Join(directory,
		"stocked.png")).Pix) {
		t.Error("the stock drew another picture the second time")
	}
}

// progress lines say each frame by its number and hold no escape bytes
// or carriage returns, so they read well in a pipe.
func TestDrawProgressIsPlain(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n"))
	said := drew(t, envIn(t, context.Background(), directory), path)
	frames := filmFrames(t)
	for index := range frames {
		line := fmt.Sprintf("frame %d of %d: draw ", index+one, frames)
		if !strings.Contains(said, line) {
			t.Errorf("no line %q in %q", line, said)
		}
	}
	if strings.ContainsAny(said, "\x1b\r") {
		t.Errorf("the progress holds escape bytes: %q", said)
	}
}

// a set file given alone is drawn as a still, to out/NAME.png under the
// working directory, the flags standing over its size and renderer.
func TestDrawASetAloneIsAStill(t *testing.T) {
	directory := t.TempDir()
	set, err := os.ReadFile(widow(t))
	if err != nil {
		t.Fatal(err)
	}
	path := written(t, directory, "widow.yaml", string(set))
	drew(t, envIn(t, context.Background(), directory), path,
		"--width", fmt.Sprint(number(t, "width")),
		"--height", fmt.Sprint(number(t, "height")),
		"--rays", fmt.Sprint(number(t, "rays")), "--renderer", "cpu")
	picture := pictureOf(t, filepath.Join(directory, outFolder,
		"widow.png"))
	if picture.Bounds().Dx() != number(t, "width") {
		t.Errorf("the set's still is %v", picture.Bounds())
	}
}

// a file that is not YAML, or not there, is refused by
// name, and a scene form the session does not draw yet is refused by
// the session's name for it.
// anti-test of TestDrawASetAloneIsAStill
// anti-test of TestJobOfReadsEachKind
func TestDrawRefusesWhatItCannotRead(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	for _, each := range []struct{ name, text, says string }{
		{"list.yaml", "- a\n- b\n", "is not a job, a plan or a set"},
		{"named.yaml", jobLine + "scene: {named: lake}\n", "named"},
	} {
		path := written(t, directory, each.name, each.text)
		got := running(Tool(env), "draw", path)
		if got.code == nothing || !strings.Contains(got.wrongSaid,
			each.says) {
			t.Errorf("%s gave %d: %q", each.name, got.code, got.wrongSaid)
		}
	}
	gone := running(Tool(env), "draw", "gone.yaml")
	if gone.code == nothing || !strings.Contains(gone.wrongSaid,
		"gone.yaml") {
		t.Errorf("a file not there gave %q", gone.wrongSaid)
	}
	two := running(Tool(env), "draw", "a.yaml", "b.yaml")
	if two.code == nothing || !strings.Contains(two.wrongSaid,
		"one file") {
		t.Errorf("two files gave %q", two.wrongSaid)
	}
}

// frames already rendered are developed, titled and written again: a
// film of a film's frames, through the clamp, is the same frames.
func TestDrawFramesAlreadyRendered(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	drew(t, env, written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n")))
	again := written(t, directory, "again.yaml", jobLine+
		"scene: {frames: film}\n"+fmt.Sprintf("fps: %d\n",
		number(t, "fps"))+"duration: {value: 1, unit: s}\n"+
		"outputs: {frames: again}\n")
	drew(t, env, again)
	name := fmt.Sprintf(frameName, nothing)
	if !bytes.Equal(pictureOf(t, filepath.Join(directory, "film",
		name)).Pix, pictureOf(t, filepath.Join(directory, "again",
		name)).Pix) {
		t.Error("a frame through the clamp came out changed")
	}
	// anti-test of TestDrawFramesAlreadyRendered
	longer := written(t, directory, "longer.yaml", jobLine+
		"scene: {frames: film}\n"+fmt.Sprintf("fps: %d\n",
		number(t, "fps"))+"duration: {value: 2, unit: s}\n"+
		"outputs: {frames: longer}\n")
	got := running(Tool(env), "draw", longer)
	if got.code == nothing || !strings.Contains(got.wrongSaid,
		"shorten the duration") {
		t.Errorf("too few plates gave %q", got.wrongSaid)
	}
}
