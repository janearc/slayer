//go:build draw

package session

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// What the end-to-end section trials pick, each by the test author.
const (
	// seqWidth and seqHeight are a frame of 48 by 32 pixels: a fire
	// drawn close up on the cpu takes seconds a frame, and this is small.
	seqWidth  = 48
	seqHeight = 32
	// seqOver and seqSide are where the film's sections start, 16
	// pixels, over which it draws in sections of 16 pixels a side: three
	// across by two down.
	seqOver = 16
	seqSide = 16
	// seqFrames is how many frames the clip has: ten a second for a fifth
	// of a second.
	seqFrames = 2
	// seqPlan is the plan the trial writes: a clip with no camera of its
	// own, so the scene's camera stands still for every frame.
	seqPlan = "$schema: https://github.com/janearc/slayer/schemas/sequence-v1.json\n" +
		"scene: %s\noutput: %s\nfps: 10\nstart: {value: 0.13, unit: s}\n" +
		"duration: {value: 0.2, unit: s}\nprint: {knee: 0.1, top: 100, bloom: 0}\n"
	// seqStart is the first frame's moment, in seconds, and seqStep the
	// time between frames: what seqPlan says.
	seqStart = 130 * time.Millisecond
	seqStep  = 100 * time.Millisecond
	// seqSections is how many sections the frame has, three across by
	// two down, counted.
	seqSections = 6
	// seqCamera is a camera four metres over the tree, looking down,
	// which is where the flame is: the one the existing sequence test
	// draws fire with.
	seqCamera = "  cameras:\n    near:\n      orientation: {point_to: the tree, alpha: -90}\n" +
		"      location:\n        loc_relative: {object: the tree, standoff_x: 0.1, standoff_y: 0, standoff_z: 4}\n"
	// seqFilm is the film of the trial's scene, in a set file: the size,
	// one ray a pixel, and the sections as a placeholder says.
	seqFilm = "    width: %d\n    height: %d\n    rays: 1\n    exposure: 100000\n" +
		"    sections: {over: %d, size: %d, keep: %v}\n"
)

// sectionsSet is a set file of a recipe fire on the fixture, drawn on the
// cpu, with a film that draws in sections and keeps them or not.
func sectionsSet(t *testing.T, keep bool) string {
	t.Helper()
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "\n  renderer: gpu-native", "", 1)
	was := "    width: 24\n    height: 16\n    rays: 1\n    exposure: 100000\n"
	now := fmt.Sprintf(seqFilm, seqWidth, seqHeight, seqOver, seqSide, keep)
	changed := strings.Replace(text, was, now, 1)
	if changed == text || strings.Contains(changed, "gpu-native") {
		t.Fatal("the fixture's film or renderer was not changed")
	}
	changed = overTheTree(t, changed)
	if err = os.WriteFile(given.set, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	return given.set
}

// sequenceFrames runs the sequence verb on a scene and gives the bytes of
// each frame file, in order.
func sequencedFrames(t *testing.T, scene string) [][]byte {
	t.Helper()
	dir := t.TempDir()
	output := filepath.Join(dir, "frames")
	plan := filepath.Join(dir, "clip.yaml")
	text := fmt.Sprintf(seqPlan, scene, output)
	if err := os.WriteFile(plan, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, wrong bytes.Buffer
	args := []string{"sequence", "--plan", plan}
	if code := sequenceRun(args, &out, &wrong); code != sequenceZero {
		t.Fatalf("sequence failed %d: %s", code, wrong.String())
	}
	var frames [][]byte
	for at := range seqFrames {
		name := fmt.Sprintf("frame-%06d.png", at)
		bytes, err := os.ReadFile(filepath.Join(output, name))
		if err != nil {
			t.Fatal(err)
		}
		frames = append(frames, bytes)
	}
	return frames
}

// keptByTheKeeper draws the clip's frames through the sequence's own
// picture path with a keeper, and says how many sections it kept and
// drew again over the frames after the first.
func keptByTheKeeper(t *testing.T, scene string) (kept, drawn int) {
	t.Helper()
	given, err := parse([]string{"-set", scene}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	_, _, given, err = staged(given)
	if err != nil {
		t.Fatal(err)
	}
	defer given.live.closeBeds()
	given.kept = &keeper{}
	plan := sequencePlan{start: seqStart}
	for at := range seqFrames {
		_, _, _, err = sequencePicture(plan, given,
			seqStart+time.Duration(at)*seqStep)
		if err != nil {
			t.Fatal(err)
		}
	}
	return given.kept.kept, given.kept.drawn
}

// a film that keeps sections draws the same frame files, byte for byte,
// as one that keeps none, through the sequence verb and a fire that
// moves. The fire is on terrain, a steady lit surface whose bounds a
// section's rays always reach, so the keeper keeps none: the count is
// logged, and the test proves only that keeping changes no bit.
func TestAKeptSequenceIsTheSequenceDrawnWhole(t *testing.T) {
	whole := sequencedFrames(t, sectionsSet(t, false))
	scene := sectionsSet(t, true)
	kept := sequencedFrames(t, scene)
	for at := range whole {
		if !bytes.Equal(whole[at], kept[at]) {
			t.Errorf("frame %d differs between keep true and keep false", at)
		}
	}
	if bytes.Equal(whole[0], whole[seqFrames-1]) {
		t.Errorf("the first and last frames are the same: the fire did not " +
			"move, so nothing was tried")
	}
	got, drawn := keptByTheKeeper(t, scene)
	t.Logf("a fire on terrain: %d sections kept, %d drawn again, over the "+
		"frames after the first", got, drawn)
}

// a set with no fire is the same world in every frame: the keeper keeps
// every section of every frame after the first, and the frames are the
// same, byte for byte, as with keep false.
func TestAKeptSequenceOfAStillWorldKeepsEverySection(t *testing.T) {
	whole := sequencedFrames(t, stillSet(t, false))
	scene := stillSet(t, true)
	kept := sequencedFrames(t, scene)
	for at := range whole {
		if !bytes.Equal(whole[at], kept[at]) {
			t.Errorf("frame %d differs between keep true and keep false", at)
		}
	}
	got, drawn := keptByTheKeeper(t, scene)
	if want := (seqFrames - sequenceOne) * seqSections; got != want ||
		drawn != sequenceZero {
		t.Errorf("%d sections kept and %d drawn again, not %d and none", got,
			drawn, want)
	}
}

// stillSet is a set file with nothing alight on the fixture, drawn in
// sections that are kept or not.
func stillSet(t *testing.T, keep bool) string {
	t.Helper()
	path := onFixture(t, "alight: true", "alight: false")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	was := "    width: 24\n    height: 16\n    rays: 1\n    exposure: 100000\n"
	now := fmt.Sprintf(seqFilm, seqWidth, seqHeight, seqOver, seqSide, keep)
	changed := strings.Replace(string(raw), was, now, 1)
	if changed == string(raw) {
		t.Fatal("the fixture's film was not changed")
	}
	if err = os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// overTheTree is a set file whose one camera looks straight down on the
// tree that burns, from four metres above it, so that the flame fills
// the middle of the frame.
func overTheTree(t *testing.T, text string) string {
	t.Helper()
	start, end := strings.Index(text, "  cameras:"), strings.Index(text, "  set:")
	if start < 0 || end < start {
		t.Fatal("the fixture's cameras were not found")
	}
	return text[:start] + seqCamera + text[end:]
}
