package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// waltz is a score at 70 beats a minute in three: a beat is 6/7 s and
// a bar 18/7 s. A lead of two notes and a drum bar of kick and snare.
const waltz = `{
	"name": "waltz", "tempo": 70, "meter": 3, "loop": false,
	"parts": [
		{"role": "lead", "bars": ["c5:2 e5", "g5:3"]},
		{"role": "perc", "bars": ["k s s@pp", "k:3"]}
	]
}`

// march is the same tune's tempo in four, for refusing a mixed meter.
const march = `{
	"name": "march", "tempo": 70, "meter": 4, "loop": false,
	"parts": [{"role": "lead", "bars": ["c5:4"]}]
}`

// writeScore writes a score into a temporary directory of the test's.
func writeScore(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "score.json")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestPositionThree checks bar, beat and sixteenth in three.
func TestPositionThree(t *testing.T) {
	cases := map[float64]string{
		0: "1.1.1", 0.25: "1.1.2", 1: "1.2.1", 2.75: "1.3.4", 3: "2.1.1",
		21: "8.1.1", 21.75: "8.1.4", 22.5: "8.2.3",
		20.9999999999: "8.1.1",
	}
	for beats, want := range cases {
		if got := readout(beats, 3); got != want {
			t.Errorf("%v beats in three: got %s, want %s", beats, got, want)
		}
	}
}

// TestPositionFour checks bar, beat and sixteenth in four.
func TestPositionFour(t *testing.T) {
	cases := map[float64]string{
		0: "1.1.1", 3.75: "1.4.4", 4: "2.1.1", 5.5: "2.2.3", 317.25: "80.2.2",
	}
	for beats, want := range cases {
		if got := readout(beats, 4); got != want {
			t.Errorf("%v beats in four: got %s, want %s", beats, got, want)
		}
	}
}

// TestSyncAtEighteen checks the brief's sync point through the score's
// own clock: 18 s at 70 in three is seven bars gone, so 8.1.1.
func TestSyncAtEighteen(t *testing.T) {
	lanes, score, err := readLanes([]string{"LEAD=" +
		writeScore(t, waltz) + "#lead"})
	if err != nil || len(lanes) != 1 {
		t.Fatalf("read: %v", err)
	}
	if got := readout(score.BeatsAt(18), score.Meter); got != "8.1.1" {
		t.Errorf("at 18 s: got %s, want 8.1.1", got)
	}
}

// TestPercussionRows checks a drum lane gets a row a hit, kick first.
func TestPercussionRows(t *testing.T) {
	lanes, _, err := readLanes([]string{"DRUMS=" +
		writeScore(t, waltz) + "#perc"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(lanes[0].hits, ","); got != "k,s" {
		t.Errorf("rows: got %s, want k,s", got)
	}
}

// TestFrameDraws draws one frame and checks the playhead's column is lit
// and the plate is left as it was.
func TestFrameDraws(t *testing.T) {
	path := writeScore(t, waltz)
	lanes, score, err := readLanes([]string{"LEAD=" + path + "#lead",
		"DRUMS=" + path + "#perc"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := newFilm(score, lanes, 540, 540)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), f.plate.Pix...)
	img := image.NewRGBA(f.plate.Bounds())
	f.frame(img, 1)
	got := img.RGBAAt(f.playheadX(), f.roll.Min.Y+f.roll.Dy()/2)
	if got != sunny {
		t.Errorf("playhead: got %v, want %v", got, sunny)
	}
	if string(before) != string(f.plate.Pix) {
		t.Error("drawing a frame changed the plate")
	}
}

// TestSpecRefusals is the anti-test: specs that cannot be lanes.
func TestSpecRefusals(t *testing.T) {
	path := writeScore(t, waltz)
	for _, spec := range []string{
		path, "LEAD" + path, "=" + path + "#lead", "LEAD=" + path,
		"LEAD=" + path + "#", "LEAD=" + path + "#pad",
		"LEAD=" + path + "#nothing", "LEAD=/no/such.json#lead",
	} {
		if _, _, err := readLanes([]string{spec}); err == nil {
			t.Errorf("%q: accepted, want refused", spec)
		}
	}
}

// TestMixedMeterRefused is the anti-test: one clock serves every lane,
// so scores in different meters are refused.
func TestMixedMeterRefused(t *testing.T) {
	_, _, err := readLanes([]string{"A=" + writeScore(t, waltz) + "#lead",
		"B=" + writeScore(t, march) + "#lead"})
	if err == nil {
		t.Error("three and four together: accepted, want refused")
	}
}

// TestLengthRefusals is the anti-test for the film's length flag.
func TestLengthRefusals(t *testing.T) {
	for _, flag := range []string{"x", "0", "-3"} {
		if _, err := length(flag, 1); err == nil {
			t.Errorf("%q: accepted, want refused", flag)
		}
	}
	if got, err := length("", 4); err != nil || got != 4 {
		t.Errorf("empty: got %v %v, want the fallback 4", got, err)
	}
}

// Example_draw is the draw verb's example, named by its declaration.
func Example_draw() {
	_ = drawVerb
}
