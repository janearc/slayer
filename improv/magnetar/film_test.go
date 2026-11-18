package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// library is where the repository keeps the scores and cut sheets the
// film's plan names.
const library = "../../cheeptune/library"

// writePNG writes a solid picture of a size to a path.
func writePNG(t *testing.T, path string, size image.Point, ink color.RGBA) {
	t.Helper()
	picture := image.NewRGBA(image.Rectangle{Max: size})
	for i := 0; i < len(picture.Pix); i += 4 {
		picture.Pix[i], picture.Pix[i+1], picture.Pix[i+2],
			picture.Pix[i+3] = ink.R, ink.G, ink.B, ink.A
	}
	var held bytes.Buffer
	if err := png.Encode(&held, picture); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, held.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeWAV writes a stereo sixteen-bit tone, a second long at 44100
// samples a second, to a path.
func writeWAV(t *testing.T, path string) {
	t.Helper()
	const rate, seconds = 44100, 1
	var data bytes.Buffer
	for i := 0; i < rate*seconds; i++ {
		sample := int16(8000 * math.Sin(2*math.Pi*220*float64(i)/rate))
		for range 2 {
			if err := binary.Write(&data, binary.LittleEndian, sample); err != nil {
				t.Fatal(err)
			}
		}
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

// filmFixture is a folder holding a plan, a tone and a segment of three
// pictures the size of the film's frame, with the paths to each.
type filmFixture struct {
	dir, plan, frames, out string
}

// newFilmFixture builds the folder: a plan that reads the library's
// glass-acid score and cuts, a tone for its sound and one segment of
// three pictures; kicking and clips are the plan's own words for those
// two keys.
func newFilmFixture(t *testing.T, kicking, clips string) filmFixture {
	t.Helper()
	dir := t.TempDir()
	fixture := filmFixture{dir: dir, plan: filepath.Join(dir, "plan.yaml"),
		frames: filepath.Join(dir, "frames"), out: filepath.Join(dir, "out.mp4")}
	if err := os.MkdirAll(filepath.Join(fixture.frames, "one"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		writePNG(t, filepath.Join(fixture.frames, "one",
			fmt.Sprintf("%02d.png", i)), filmSize,
			color.RGBA{R: uint8(60 * i), G: 90, B: 120, A: 255})
	}
	writeWAV(t, filepath.Join(dir, "tone.wav"))
	text := fmt.Sprintf(`tune:
  score: %s/glass-acid.json
  cuts: %s/glass-acid.cuts.json
  sound: %s/tone.wav
  kicking: %s
lifts: [breaks]
clips: %q
intro:
  clip: angry
  words: ""
  frames: 0
segments:
  - {frames: one, count: 3}
lines:
  - {from: 0.0, until: 0.05, say: "a line"}
`, library, library, dir, kicking, clips)
	if err := os.WriteFile(fixture.plan, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return fixture
}

// A plan the film cannot be made from is refused, each at the step that
// fails and with the cause.
func TestFilmRefusesAPlanThatCannotBeMade(t *testing.T) {
	broken := func(t *testing.T, fixture filmFixture, from, to string) {
		t.Helper()
		raw, err := os.ReadFile(fixture.plan)
		if err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(string(raw), from, to, 1)
		if changed == string(raw) {
			t.Fatalf("the plan has no %q to change", from)
		}
		if err := os.WriteFile(fixture.plan, []byte(changed), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name   string
		break_ func(t *testing.T, fixture filmFixture)
		want   string
	}{
		{"a part the cuts do not have", func(t *testing.T, f filmFixture) {
			broken(t, f, "[spreads, breaks]", "[nowhere]")
		}, "nowhere"},
		{"a sound that is not there", func(t *testing.T, f filmFixture) {
			broken(t, f, "tone.wav", "missing.wav")
		}, "missing.wav"},
		{"a score that is not there", func(t *testing.T, f filmFixture) {
			broken(t, f, "glass-acid.json", "no-such-score.json")
		}, "no-such-score.json"},
		{"a cut sheet that is not there", func(t *testing.T, f filmFixture) {
			broken(t, f, "glass-acid.cuts.json", "no-such.cuts.json")
		}, "no-such.cuts.json"},
		{"a segment folder that is not there", func(t *testing.T, f filmFixture) {
			broken(t, f, "{frames: one", "{frames: two")
		}, "two"},
		{"a segment of the wrong size", func(t *testing.T, f filmFixture) {
			broken(t, f, "count: 3", "count: 4")
		}, "holds 3 pictures, the plan says 4"},
		{"a picture that is not a picture", func(t *testing.T, f filmFixture) {
			if err := os.WriteFile(filepath.Join(f.frames, "one", "01.png"),
				[]byte("not a png"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "01.png"},
		{"a clips folder with no clips", func(t *testing.T, f filmFixture) {
			broken(t, f, `clips: ""`, fmt.Sprintf("clips: %q", f.dir))
		}, "host:"},
	}
	for _, c := range cases {
		fixture := newFilmFixture(t, "[spreads, breaks]", "")
		c.break_(t, fixture)
		err := film(fixture.plan, fixture.frames, fixture.out, &strings.Builder{})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: film gave %v, want a refusal naming %q", c.name,
				err, c.want)
		}
	}
}

// A plan that is missing or is not yaml is refused by the film command
// and by run.
func TestFilmRefusesAPlanThatIsMissingOrNotYAML(t *testing.T) {
	dir := t.TempDir()
	if err := film(filepath.Join(dir, "none.yaml"), dir,
		filepath.Join(dir, "out.mp4"), &strings.Builder{}); !os.IsNotExist(err) {
		t.Errorf("a missing plan gave %v, want a missing-file error", err)
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("tune: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"magnetar", "film", bad, dir,
		filepath.Join(dir, "out.mp4")}, &strings.Builder{}); err == nil {
		t.Error("a plan that is not yaml was read")
	}
}

// A folder's pictures are its png files in name order; anything else in
// it is left out, and a folder that is not there is refused.
func TestPicturesInListsPNGsInNameOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.png", "a.png", "c.txt", "10.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := picturesIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "10.png"), filepath.Join(dir, "a.png"),
		filepath.Join(dir, "b.png")}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("picturesIn gave %v, want %v", got, want)
	}
	if _, err := picturesIn(filepath.Join(dir, "absent")); err == nil {
		t.Error("a folder that is not there was listed")
	}
}

// A picture is read as RGBA with its own pixels; one that is missing or
// is no png is refused, naming the path.
func TestReadPNGGivesThePixelsAndRefusesWhatIsNoPicture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.png")
	writePNG(t, path, image.Pt(4, 3), color.RGBA{R: 10, G: 20, B: 30, A: 255})
	got, err := readPNG(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 3 {
		t.Errorf("the picture is %v, want 4 by 3", got.Bounds())
	}
	if pixel := got.RGBAAt(3, 2); pixel != (color.RGBA{R: 10, G: 20, B: 30, A: 255}) {
		t.Errorf("the last pixel is %v, want 10 20 30 opaque", pixel)
	}
	if _, err := readPNG(filepath.Join(dir, "absent.png")); err == nil {
		t.Error("a missing picture was read")
	}
	bad := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(bad, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPNG(bad); err == nil || !strings.Contains(err.Error(), bad) {
		t.Errorf("a file that is no png gave %v, want an error naming it", err)
	}
}

// A short's middle frame is written as a picture, and the command says
// where.
func TestRunWritesTheMiddleFrameAsAPicture(t *testing.T) {
	pictures := stills(t, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	out := filepath.Join(t.TempDir(), "field.png")
	var said strings.Builder
	err := run([]string{"magnetar", "magnetar-walk.yaml", pictures, "field",
		out}, &said)
	if err != nil {
		t.Fatal(err)
	}
	if said.String() != out+"\n" {
		t.Errorf("the command said %q, want %q", said.String(), out+"\n")
	}
	file, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	made, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if made.Bounds().Dx() == 0 || made.Bounds().Dy() == 0 {
		t.Errorf("the picture is %v, want a frame", made.Bounds())
	}
}

// The command refuses a script that is not there, a picture the short
// needs and a path it cannot write.
func TestRunRefusesWhatItCannotRead(t *testing.T) {
	dir := t.TempDir()
	if err := run([]string{"magnetar", filepath.Join(dir, "none.yaml"), dir,
		"field", filepath.Join(dir, "x.png")}, &strings.Builder{}); err == nil {
		t.Error("a script that is not there was read")
	}
	if err := run([]string{"magnetar", "magnetar-walk.yaml", dir, "field",
		filepath.Join(dir, "x.png")}, &strings.Builder{}); err == nil ||
		!strings.Contains(err.Error(), "magnetar-field.png") {
		t.Errorf("a short with no still gave %v, want a refusal naming it", err)
	}
	pictures := stills(t, color.RGBA{A: 255})
	if err := run([]string{"magnetar", "magnetar-walk.yaml", pictures, "field",
		filepath.Join(dir, "no-such-folder", "x.png")},
		&strings.Builder{}); err == nil {
		t.Error("a picture was written to a folder that is not there")
	}
}

// stills is a folder holding every still the walk script names, each
// tall enough for the script's window, in an ink.
func stills(t *testing.T, ink color.RGBA) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"field", "shells", "far", "beside", "world"} {
		writePNG(t, filepath.Join(dir, "magnetar-"+name+".png"),
			image.Pt(216, 512), ink)
	}
	return dir
}

// A short named that the script has not is refused once its pictures
// are found.
func TestRunRefusesAShortTheScriptHasNot(t *testing.T) {
	err := run([]string{"magnetar", "magnetar-walk.yaml",
		stills(t, color.RGBA{A: 255}), "sideways",
		filepath.Join(t.TempDir(), "x.png")}, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), `"sideways"`) {
		t.Errorf("a short the script has not gave %v, want a refusal "+
			"naming it", err)
	}
}

// A movie that cannot be started, or a still that holds a header and
// not its pixels, is refused with the cause.
func TestRunRefusesAMovieItCannotStartAndAStillItCannotDecode(t *testing.T) {
	dir := t.TempDir()
	err := run([]string{"magnetar", "magnetar-walk.yaml",
		stills(t, color.RGBA{A: 255}), "field",
		filepath.Join(dir, "no-such-folder", "x.mp4")}, &strings.Builder{})
	if err == nil {
		t.Error("a movie was written to a folder that is not there")
	}
	pictures := stills(t, color.RGBA{A: 255})
	field := filepath.Join(pictures, "magnetar-field.png")
	whole, err := os.ReadFile(field)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(field, whole[:len(whole)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	err = run([]string{"magnetar", "magnetar-walk.yaml", pictures, "field",
		filepath.Join(dir, "x.png")}, &strings.Builder{})
	if err == nil {
		t.Error("a truncated still was drawn")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "x.png")); statErr == nil {
		t.Error("a picture was written from a still that could not be read")
	}
}
