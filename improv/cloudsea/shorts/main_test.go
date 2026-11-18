package main

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/improv/walk"
)

// barSlack is how near a bar's start must be to its arithmetic. Picked:
// float64 rounding.
const barSlack = 1e-9

// steady is a score of three bars of four at 60 a minute: a second a
// beat, four a bar.
const steady = `{"name": "steady", "tempo": 60, "meter": 4, "loop": false,
 "parts": [{"role": "lead", "bars": ["c4:4", "c4:4", "c4:4"]}]}`

// TestBarStart: the first bar starts at nought and the third eight
// seconds on; a bar before the first or past the last is refused.
func TestBarStart(t *testing.T) {
	score, err := cheeptune.ParseScore([]byte(steady))
	if err != nil {
		t.Fatal(err)
	}
	for bar, want := range map[int]float64{1: 0, 3: 8} {
		got, err := barStart(score, bar)
		if err != nil || math.Abs(got-want) > barSlack {
			t.Errorf("bar %d: %v, %v; want %v", bar, got, err, want)
		}
	}
	for _, bar := range []int{0, 4, -1} {
		if _, err := barStart(score, bar); !errors.Is(err, errAsked) {
			t.Errorf("bar %d was taken: %v", bar, err)
		}
	}
}

// shortScript is a script over package walk's marker fixture with one short
// of two seconds whose sound line is given.
const shortScript = `set: set.yaml
places: places.csv
shorts:
  - name: where
    title: Where the markers are
%s
    shots:
      - camera: down
        picture: down.png
        seconds: 1
        lit: show
        name: The markers
        sentence: Six stones.
      - set: set.yaml
        camera: above
        picture: above.png
        seconds: 1
        lit: inspection
        name: From above
        sentence: Same, above.
`

// writeScript copies the marker fixture into a new folder with a script
// whose sound line is given, and returns the script's path.
func writeScript(t *testing.T, sound string) string {
	t.Helper()
	fixture := filepath.Join("..", "..", "walk", "testdata", "marker")
	dir := t.TempDir()
	for _, name := range []string{"set.yaml", "places.csv", "down.png", "above.png"} {
		data, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "script.yaml")
	if err := os.WriteFile(path, []byte(strings.Replace(shortScript, "%s", sound, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// readShort reads the script and returns its short named where.
func readShort(t *testing.T, path string) walk.Short {
	t.Helper()
	script, err := walk.ReadScript(path, "")
	if err != nil {
		t.Fatal(err)
	}
	short, found := script.Short("where")
	if !found {
		t.Fatal("the script has no short where")
	}
	return short
}

// TestSoundOf checks that the two channels of a piece are taken in turn.
func TestSoundOf(t *testing.T) {
	piece := &cheeptune.Piece{Rate: 44100, Left: []float32{0.1, 0.2, 0.3}, Right: []float32{-0.1, -0.2, -0.3}}
	sound, err := soundOf(piece)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{0.1, -0.1, 0.2, -0.2, 0.3, -0.3}
	got := sound.Samples()
	if sound.Rate() != 44100 || sound.Length() != 3 || len(got) != len(want) {
		t.Fatalf("sound: rate %d length %d", sound.Rate(), sound.Length())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sample %d: got %v want %v", i, got[i], want[i])
		}
	}
	if _, err := soundOf(&cheeptune.Piece{}); err == nil {
		t.Error("a piece with no rate was taken")
	}
}

// TestWritePNG checks that a frame is written and read back, and that a path
// that cannot be created is refused.
func TestWritePNG(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 3, 1))
	frame.SetRGBA(2, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	path := filepath.Join(t.TempDir(), "out.png")
	if err := writePNG(path, frame); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	read, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := read.At(2, 0).RGBA()
	if read.Bounds().Dx() != 3 || r>>8 != 10 || g>>8 != 20 || b>>8 != 30 {
		t.Errorf("read back %v, %v %v %v", read.Bounds(), r>>8, g>>8, b>>8)
	}
	if err := writePNG(filepath.Join(t.TempDir(), "gone", "out.png"), frame); err == nil {
		t.Error("wrote into a folder that is not there")
	}
}

// TestRunFrameAndRefusals checks that one frame is written at the short's
// size and that a missing script, a missing short and a frame past the end
// are refused, the missing short by name.
func TestRunFrameAndRefusals(t *testing.T) {
	script := writeScript(t, "    sound: {bar: 2}")
	out := filepath.Join(t.TempDir(), "frame.png")
	if err := run(asked{script: script, short: "where", frame: 30, png: out}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil || config.Width != 1080 || config.Height != 1920 {
		t.Errorf("frame is %dx%d: %v", config.Width, config.Height, err)
	}
	if err := run(asked{script: script, short: "where", frame: 48, png: out}); err == nil {
		t.Error("a frame past the end was written")
	}
	if err := run(asked{script: filepath.Join(t.TempDir(), "gone"), short: "where", frame: -1}); err == nil {
		t.Error("a missing script was read")
	}
	err = run(asked{script: script, short: "there", frame: -1})
	if !errors.Is(err, errAsked) || !strings.Contains(err.Error(), `"there"`) {
		t.Errorf("missing short: %v", err)
	}
	if err := run(asked{script: script, short: "where", frame: -1}); !errors.Is(err, errAsked) {
		t.Errorf("sound with no score: %v", err)
	}
}
