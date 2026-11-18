package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/walk"
)

// library is where the repository keeps the scores and cut sheets.
const library = "../../../cheeptune/library"

// scriptText is a walk script of one short of two shots, the first two
// frames long and the second six, with a sentence over it.
const scriptText = `places:
  unit: km
  at:
    the star: [0, 0, 0]
shorts:
  - name: pair
    title: two shots
    shots:
      - picture: first.png
        eye: {from: [0, 45.74, 215.19], look-at: [0, 0, 0], up: [0, 1, 0],
          wide: 40}
        seconds: 0.0833
        lit: drawing
        window: {from: [0, 0.21875, 1], to: [0, 0.21875, 1]}
        name: the star
      - picture: second.png
        eye: {from: [0, 45.74, 215.19], look-at: [0, 0, 0], up: [0, 1, 0],
          wide: 40}
        seconds: 0.25
        lit: drawing
        window: {from: [0, 0.21875, 1], to: [0, 0.21875, 1]}
        name: the star again
        sentence: Hi.
`

// writeTone writes a stereo sixteen-bit tone of 44100 samples a second,
// a second long, to a path.
func writeTone(t *testing.T, path string) {
	t.Helper()
	const rate = 44100
	var data bytes.Buffer
	for i := 0; i < rate; i++ {
		sample := int16(8000 * math.Sin(2*math.Pi*220*float64(i)/rate))
		binary.Write(&data, binary.LittleEndian, sample)
		binary.Write(&data, binary.LittleEndian, sample)
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(36+data.Len()))
	out.WriteString("WAVEfmt ")
	for _, field := range []any{uint32(16), uint16(1), uint16(2),
		uint32(rate), uint32(rate * 4), uint16(4), uint16(16)} {
		binary.Write(&out, binary.LittleEndian, field)
	}
	out.WriteString("data")
	binary.Write(&out, binary.LittleEndian, uint32(data.Len()))
	out.Write(data.Bytes())
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture is a folder holding the script, its stills and a tone, with
// the flags that run the short.
type fixture struct {
	dir  string
	args []string
}

// newFixture builds the folder and the flags that make the film from it.
func newFixture(t *testing.T) fixture {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "script.yaml")
	if err := os.WriteFile(script, []byte(scriptText), 0o644); err != nil {
		t.Fatal(err)
	}
	picture := image.NewRGBA(image.Rect(0, 0, 216, 512))
	for i := 0; i < len(picture.Pix); i += 4 {
		picture.Pix[i], picture.Pix[i+1], picture.Pix[i+3] = 90, 140, 255
	}
	var held bytes.Buffer
	if err := png.Encode(&held, picture); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first.png", "second.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), held.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeTone(t, filepath.Join(dir, "tone.wav"))
	return fixture{dir: dir, args: []string{"-script", script, "-pictures", dir,
		"-short", "pair", "-out", filepath.Join(dir, "out.mp4"),
		"-score", library + "/glass-acid.json",
		"-sheet", library + "/glass-acid.cuts.json",
		"-film", filepath.Join(dir, "tone.wav"),
		"-kicks", "spreads, breaks", "-intro", "hello there",
		"-intro-seconds", "0.0833"}}
}

// with is the fixture's flags with one flag's value changed.
func (f fixture) with(flag, value string) []string {
	args := append([]string(nil), f.args...)
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			args[i+1] = value
			return args
		}
	}
	return append(args, flag, value)
}

// Each thing that can be wrong with the inputs stops the command with a
// message naming it, and no film is written.
func TestRunRefusesInputsThatCannotMakeAFilm(t *testing.T) {
	f := newFixture(t)
	empty := t.TempDir()
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"a script that is not there", f.with("-script",
			filepath.Join(f.dir, "none.yaml")), "none.yaml"},
		{"a short the script has not", f.with("-short", "other"),
			`no short is called "other"`},
		{"a score that is not there", f.with("-score",
			filepath.Join(f.dir, "none.json")), "none.json"},
		{"a sheet that is not there", f.with("-sheet",
			filepath.Join(f.dir, "none.cuts.json")), "none.cuts.json"},
		{"a film that is not there", f.with("-film",
			filepath.Join(f.dir, "none.wav")), "none.wav"},
		{"a kicking part the sheet has not", f.with("-kicks", "nowhere"),
			"nowhere"},
		{"a clips folder with no clips", f.with("-clips", empty), "host:"},
		{"a folder that cannot hold the film", f.with("-out",
			filepath.Join(f.dir, "no-such-folder", "out.mp4")), "movie:"},
	}
	for _, c := range cases {
		var out, problems bytes.Buffer
		if code := run(c.args, &out, &problems); code != failed ||
			!strings.Contains(problems.String(), c.want) {
			t.Errorf("%s: exit %d, error %q, want a failure naming %q",
				c.name, code, problems.String(), c.want)
		}
		if out.Len() != 0 {
			t.Errorf("%s: the command said %q", c.name, out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(f.dir, "out.mp4")); err == nil {
		t.Error("a film was written from inputs that could not make one")
	}
}

// The host speaks over each shot that has a sentence, and only over
// those, from where the shot starts to where it ends.
func TestSpeakingSpansTheShotsWithASentence(t *testing.T) {
	f := newFixture(t)
	script, err := walk.ReadScript(filepath.Join(f.dir, "script.yaml"), f.dir)
	if err != nil {
		t.Fatal(err)
	}
	short, found := script.Short("pair")
	if !found {
		t.Fatal("the script has no short pair")
	}
	spans := speaking(short)
	if len(spans) != 1 {
		t.Fatalf("the host speaks over %d spans, want 1: %v", len(spans), spans)
	}
	if math.Abs(spans[0].From-0.0833) > 1e-9 ||
		math.Abs(spans[0].Until-0.3333) > 1e-9 {
		t.Errorf("the host speaks from %v to %v, want 0.0833 to 0.3333",
			spans[0].From, spans[0].Until)
	}
}

// A lone comma list has no names in it when it is all blanks.
func TestNamesOfBlanksIsEmpty(t *testing.T) {
	if got := names(" , ,"); len(got) != 0 {
		t.Errorf("blanks read as %q, want none", got)
	}
	if got := set(" , "); len(got) != 0 {
		t.Errorf("blanks make the set %v, want empty", got)
	}
}
