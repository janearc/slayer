package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/improv/moonlake/show"
	"github.com/janearc/slayer/improv/walk"
)

// markerDir is the fixture set that package walk's own tests draw from.
var markerDir = filepath.Join("..", "..", "walk", "testdata", "marker")

// scoreFile is a score of sixteen beats half a second apart from 2 s, that
// is four bars of four beats.
var scoreFile = filepath.Join("..", "show", "testdata", "score.yaml")

// shortScript is a script over the marker fixture with one short of a
// second a shot, 48 frames in all, whose sound line is given.
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
	dir := t.TempDir()
	for _, name := range []string{"set.yaml", "places.csv", "down.png", "above.png"} {
		data, err := os.ReadFile(filepath.Join(markerDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "script.yaml")
	text := strings.Replace(shortScript, "%s", sound, 1)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeWav writes a mono 16-bit wav of a steady tone, rate samples a second
// for the given seconds, and returns its path.
func writeWav(t *testing.T, seconds int) string {
	t.Helper()
	const rate = 44100
	count := rate * seconds
	data := make([]byte, 44+2*count)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(36+2*count))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], rate)
	binary.LittleEndian.PutUint32(data[28:], 2*rate)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(2*count))
	for i := 0; i < count; i++ {
		sample := int16(8000 * math.Sin(2*math.Pi*440*float64(i)/rate))
		binary.LittleEndian.PutUint16(data[44+2*i:], uint16(sample))
	}
	path := filepath.Join(t.TempDir(), "song.wav")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// readPNG reads a png file and returns its size.
func readPNG(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	picture, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

// TestBarTime checks bar starts from the score's grid: bars from one are the
// score's own, and bars before it are counted back at the length of bar one.
func TestBarTime(t *testing.T) {
	score, err := show.ReadScore(scoreFile)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[int]float64{3: 6, 1: 2, 0: 0, -1: -2}
	for bar, want := range cases {
		got, err := barTime(score, bar)
		if err != nil || math.Abs(got-want) > 1e-9 {
			t.Errorf("bar %d: got %v, %v; want %v", bar, got, err, want)
		}
	}
	if _, err := barTime(score, 5); err == nil {
		t.Error("a bar the score lacks was taken")
	}
}

// TestBarTimeNeedsSecondBar checks that a score of one bar cannot count
// bars back from bar one, and that bar one itself is still found.
func TestBarTimeNeedsSecondBar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "score.yaml")
	text := "score:\n  duration: 4\n  tracks:\n    lead:\n      - {at_sec: 1.0, vel: 90}\n  beats:\n    times: [1.0, 1.5]\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	score, err := show.ReadScore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := barTime(score, 1); err != nil || got != 1 {
		t.Errorf("bar 1: %v, %v", got, err)
	}
	if _, err := barTime(score, 0); err == nil {
		t.Error("counted back from bar one with no bar two")
	}
}

// TestWritePNG checks that a picture is written and read back, and that a
// path that cannot be created is refused.
func TestWritePNG(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 2, 1))
	picture.SetRGBA(1, 0, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	path := filepath.Join(t.TempDir(), "out.png")
	if err := writePNG(path, picture); err != nil {
		t.Fatal(err)
	}
	read := readPNG(t, path)
	r, g, b, _ := read.At(1, 0).RGBA()
	if read.Bounds().Dx() != 2 || r>>8 != 200 || g>>8 != 100 || b>>8 != 50 {
		t.Errorf("read back %v, %v %v %v", read.Bounds(), r>>8, g>>8, b>>8)
	}
	if err := writePNG(filepath.Join(t.TempDir(), "gone", "out.png"), picture); err == nil {
		t.Error("wrote into a folder that is not there")
	}
}

// TestRunFrame checks that one frame of a short is written as a png of the
// short's size, and that a frame past the end is refused.
func TestRunFrame(t *testing.T) {
	script := writeScript(t, "    sound: {bar: 1}")
	out := filepath.Join(t.TempDir(), "frame.png")
	if err := run(asked{script: script, short: "where", frame: 30, png: out}); err != nil {
		t.Fatal(err)
	}
	if size := readPNG(t, out).Bounds().Size(); size != image.Pt(1080, 1920) {
		t.Errorf("frame is %v", size)
	}
	if err := run(asked{script: script, short: "where", frame: 48, png: out}); err == nil {
		t.Error("a frame past the end was written")
	}
}

// TestRunRefusals checks that a missing script, a short the script lacks
// and a film that is not there are each refused.
func TestRunRefusals(t *testing.T) {
	script := writeScript(t, "    sound: {bar: 1}")
	missing := filepath.Join(t.TempDir(), "gone")
	cases := map[string]asked{
		"no script":  {script: missing, short: "where", frame: -1},
		"no short":   {script: script, short: "there", frame: -1},
		"no film":    {film: missing, png: filepath.Join(t.TempDir(), "x.png")},
		"no score":   {script: script, short: "where", frame: -1, score: missing},
		"empty film": {film: t.TempDir()},
	}
	for name, given := range cases {
		if err := run(given); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	err := run(asked{script: script, short: "there", frame: -1})
	if err == nil || !strings.Contains(err.Error(), `"there"`) {
		t.Errorf("the missing short is not named: %v", err)
	}
}

// TestSoundFor checks the song under a short: none for a silent short, the
// song's span for a sounded one, and each way it can be refused.
func TestSoundFor(t *testing.T) {
	silent, err := readShort(t, writeScript(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if sound, err := soundFor(silent, asked{}); err != nil || sound.Rate() != 0 {
		t.Errorf("silent short: %v, %v", sound.Rate(), err)
	}
	song := writeWav(t, 8)
	sounded, err := readShort(t, writeScript(t, "    sound: {bar: 2}"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := soundFor(sounded, asked{score: scoreFile, song: song})
	if err != nil {
		t.Fatal(err)
	}
	if got.Rate() != 44100 || got.Duration() < 1990*time.Millisecond || got.Duration() > 2010*time.Millisecond {
		t.Errorf("span is %v at %d", got.Duration(), got.Rate())
	}
	cases := map[string]asked{
		"no score": {score: filepath.Join(t.TempDir(), "gone"), song: song},
		"no song":  {score: scoreFile, song: filepath.Join(t.TempDir(), "gone")},
	}
	for name, given := range cases {
		if _, err := soundFor(sounded, given); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	late, err := readShort(t, writeScript(t, "    sound: {bar: 9}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := soundFor(late, asked{score: scoreFile, song: song}); err == nil {
		t.Error("a bar the score lacks was taken")
	}
	early, err := readShort(t, writeScript(t, "    sound: {bar: -1}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := soundFor(early, asked{score: scoreFile, song: song}); err == nil {
		t.Error("a span before the song was taken")
	}
	short, err := readShort(t, writeScript(t, "    sound: {bar: 4}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := soundFor(short, asked{score: scoreFile, song: writeWav(t, 3)}); err == nil {
		t.Error("a span past the song's end was taken")
	}
}

// TestRunWriteFailsOnBadPath checks that a film that cannot be written is
// reported as an error.
func TestRunWriteFailsOnBadPath(t *testing.T) {
	script := writeScript(t, "")
	given := asked{script: script, short: "where", frame: -1,
		mp4: filepath.Join(t.TempDir(), "gone", "short.mp4")}
	if err := run(given); err == nil {
		t.Error("wrote a film into a folder that is not there")
	}
}

// readShort reads the script and returns its short named where.
func readShort(t *testing.T, path string) (walk.Short, error) {
	t.Helper()
	script, err := walk.ReadScript(path, "")
	if err != nil {
		return walk.Short{}, err
	}
	short, found := script.Short("where")
	if !found {
		t.Fatal("the script has no short where")
	}
	return short, nil
}
