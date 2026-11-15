package walk

import (
	"errors"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// goodScript is a script over the marker fixture that reads.
const goodScript = `set: set.yaml
places: places.csv
shorts:
  - name: where
    title: Where the markers are
    sound: {bar: 2}
    shots:
      - camera: down
        picture: down.png
        seconds: 3
        lit: show
        name: The markers
        figures:
          - {value: "6", unit: "", says: boulders, from: set.yaml things}
        labels:
          - {thing: close m2, words: a marker}
          - {at: [263450, 2148300, 1073.6], words: the lookout, side: left}
        sentence: Six boulders glow alone.
      - set: set.yaml
        camera: above
        picture: above.png
        seconds: 2
        lit: inspection
        window: {from: [0, 0, 0.5], to: [0.2, 0.1, 0.4]}
        name: From above
        labels:
          - {thing: close m1, words: here, side: right}
        scale: {thing: close m1}
        sentence: The same, from above.
`

// scriptWith writes a script into a copy of the marker fixture, with
// one text replaced, and reads it.
func scriptWith(t *testing.T, old, new string) (Script, error) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"set.yaml", "places.csv", "down.png",
		"above.png", "odd.png"} {
		source, err := os.ReadFile(filepath.Join(markerDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), source,
			0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := strings.Replace(goodScript, old, new, 1)
	if old != "" && text == goodScript {
		t.Fatalf("%q is not in the script", old)
	}
	path := filepath.Join(dir, "script.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return ReadScript(path, "")
}

// TestReadScript reads the good script and checks what it holds.
func TestReadScript(t *testing.T) {
	script, err := scriptWith(t, "", "")
	if err != nil {
		t.Fatal(err)
	}
	short, found := script.Short("where")
	if !found || len(script.Shorts()) != 1 {
		t.Fatalf("the short is not there: %v", script.Shorts())
	}
	bar, sounded := short.Bar()
	if short.Seconds() != 5 || short.Frames() != 5*filmRate ||
		!sounded || bar != 2 || short.Title() != "Where the markers are" ||
		short.Name() != "where" {
		t.Errorf("the short is %+v", short)
	}
	shots := short.Shots()
	if shots[0].Camera() != "down" || shots[0].Lit() != "show" ||
		shots[1].tag != lighting["inspection"] ||
		len(shots[0].Labels()) != 2 ||
		shots[1].Seconds() != 2 || shots[0].Sentence() == "" {
		t.Errorf("the shots are %+v", shots)
	}
	figure := shots[0].figures[0]
	if figure.Value() != "6" || figure.From() == "" || figure.Says() == "" ||
		figure.Unit() != "" {
		t.Errorf("the figure is %+v", figure)
	}
	if _, found := script.Short("there"); found {
		t.Error("a short the script lacks was found")
	}
	if label := shots[0].Labels()[0]; label.Words() != "a marker" ||
		!label.At().On() {
		t.Errorf("the label is %+v", label)
	}
}

// TestReadScriptRefuses: each wrong thing is refused by its name.
func TestReadScriptRefuses(t *testing.T) {
	long := strings.Repeat("word ", 20)
	for _, wrong := range []struct{ old, new, named string }{
		{"camera: down", "camera: sideways", "sideways"},
		{"thing: close m2", "thing: close m9", "close m9"},
		{", from: set.yaml things", "", "from"},
		{"Six boulders glow alone.", long, "to read"},
		{"lit: show", "lit: bright", "bright"},
		{"[0.2, 0.1, 0.4]", "[0.8, 0.1, 0.4]", "off its"},
		{"[0, 0, 0.5]", "[0, 0.5]", "left, top and width"},
		{"side: left", "side: under", "under"},
		{"{thing: close m1, words: here", "{toward: [270, 5], words: here",
			"does not see"},
		{"thing: close m2,", "thing: close m2, toward: [1, 2],", "one of"},
		{"[263450, 2148300, 1073.6]", "[263450, 2148300]", "three numbers"},
		{"scale: {thing: close m1}", "scale: {toward: [10, 10]}", "sky"},
		{"sound: {bar: 2}", "sound: {}", "no bar"},
		{"seconds: 2", "seconds: 0", "time"},
		{"picture: down.png", "picture: set.yaml", "not a png"},
		{"picture: down.png", "picture: gone.png", "gone.png"},
		{"picture: above.png", "picture: odd.png", "draw it again"},
		{"set: set.yaml\nplaces", "set: gone.yaml\nplaces", "gone.yaml"},
		{"places: places.csv", "places: {unit: m}", "places table"},
		{"lit: show", "lit: show\n        tag: my own words", "one of the two"},
		{"lit: show", "tag: " + strings.Repeat("x", tagMost+1), "at most"},
		{"camera: down\n", "eye: {from: [0, 0, 0]}\n", "name one"},
		{"{thing: close m2, words: a marker}",
			"{pixel: [1, 2, 3], words: a marker}", "column and a row"},
		{"scale: {thing: close m1}", "scale: {pixels: 10}", "outright"},
		{"- set: set.yaml", "- set: lost.yaml", "lost.yaml"},
		{"places: places.csv", "places: down.png", "places table"},
		{"title: Where the markers are", "title: ''", "a title"},
		{"shorts:", "shorts: []\nnothing:", "no shorts"},
	} {
		_, err := scriptWith(t, wrong.old, wrong.new)
		if !errors.Is(err, ErrWalk) || !strings.Contains(err.Error(),
			wrong.named) {
			t.Errorf("%q for %q: %v", wrong.new, wrong.old, err)
		}
	}
}

// TestPanelFits refuses a sentence of too many rows and a figure wider
// than its column; a short one fits.
func TestPanelFits(t *testing.T) {
	if err := panelFits("short", []writtenFigure{{Value: "1", Unit: "m",
		Says: "tall"}}, false); err != nil {
		t.Errorf("a short panel: %v", err)
	}
	if err := panelFits(strings.Repeat("words go here ", 30),
		nil, true); !errors.Is(err, ErrWalk) {
		t.Errorf("a long sentence: %v", err)
	}
	wide := writtenFigure{Value: "384,400,000", Unit: "km", Says: "x"}
	if err := panelFits("s", []writtenFigure{wide, wide,
		wide}, false); !errors.Is(err, ErrWalk) {
		t.Errorf("a wide figure: %v", err)
	}
	if _, err := figuresOf(make([]writtenFigure, mostFigures+1)); err == nil {
		t.Error("too many figures were taken")
	}
}

// TestFrame composes frames of the good short: the frame's size, the
// ground in the panel, a dip at the cut and none at the start, and a
// frame past the end refused.
func TestFrame(t *testing.T) {
	script, err := scriptWith(t, "", "")
	if err != nil {
		t.Fatal(err)
	}
	short, _ := script.Short("where")
	kept := NewStills()
	tones, _ := colours()
	cut := 3 * filmRate
	for _, i := range []int{0, cut - 1, cut, short.Frames() - 1} {
		frame, err := short.Frame(i, kept)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Bounds().Size() != frameSize {
			t.Fatalf("frame %d is %v", i, frame.Bounds())
		}
		if frame.RGBAAt(frameSize.X-2, frameSize.Y-2) != tones["ground"] {
			t.Errorf("frame %d: the panel's corner is not the ground", i)
		}
	}
	if now := short.momentOf(0); now.dip != 0 || now.shot != 0 {
		t.Errorf("the start dips: %+v", now)
	}
	if now := short.momentOf(cut); now.dip != 1 || now.shot != 1 {
		t.Errorf("the cut does not dip fully: %+v", now)
	}
	if now := short.momentOf(short.Frames() - 1); now.dip != 0 {
		t.Errorf("the end dips: %+v", now)
	}
	if _, err := short.Frame(short.Frames(), kept); err == nil {
		t.Error("a frame past the end was made")
	}
	if _, err := kept.of(still{path: "gone.png"}); err == nil {
		t.Error("a still that is not there was read")
	}
}

// TestSongSpan cuts a span, low, faded at both ends, and refuses one
// past the song's end.
func TestSongSpan(t *testing.T) {
	const rate = 44100
	samples := make([]float32, 10*rate*stereo)
	for i := range samples {
		samples[i] = 1
	}
	song, err := movie.NewSound(rate, samples)
	if err != nil {
		t.Fatal(err)
	}
	span, err := SongSpan(song, 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	cut := span.Samples()
	if span.Length() != 5*rate || cut[0] != 0 || cut[len(cut)-1] != 0 {
		t.Errorf("the span is %d long, from %v to %v", span.Length(), cut[0],
			cut[len(cut)-1])
	}
	if middle := cut[(5*rate/2)*stereo]; middle != soundLevel {
		t.Errorf("the middle is %v, not %v", middle, soundLevel)
	}
	for _, wrong := range [][2]float64{{6, 5}, {-1, 2}, {0, 0}} {
		if _, err := SongSpan(song, wrong[0], wrong[1]); err == nil {
			t.Errorf("%v s from %v s was cut", wrong[1], wrong[0])
		}
	}
}

// TestReadScriptPictures finds the stills and a shot's own set in a
// folder of pictures apart from the script, and refuses a script that
// is not there or not YAML.
func TestReadScriptPictures(t *testing.T) {
	dir, pictures := t.TempDir(), t.TempDir()
	copies := map[string]string{"set.yaml": dir, "places.csv": dir,
		"down.png": pictures, "above.png": pictures}
	for name, into := range copies {
		source, err := os.ReadFile(filepath.Join(markerDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(into, name), source,
			0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := strings.Replace(goodScript, "- set: set.yaml", "- set: s.yaml", 1)
	source, _ := os.ReadFile(filepath.Join(markerDir, "set.yaml"))
	if err := os.WriteFile(filepath.Join(pictures, "s.yaml"), source,
		0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "script.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadScript(path, pictures); err != nil {
		t.Errorf("the pictures apart: %v", err)
	}
	if _, err := ReadScript(path, ""); !errors.Is(err, ErrWalk) {
		t.Errorf("the pictures not beside the script were found: %v", err)
	}
	if _, err := ReadScript(filepath.Join(dir, "gone.yaml"),
		""); !errors.Is(err, ErrWalk) {
		t.Errorf("a script not there: %v", err)
	}
	if err := os.WriteFile(path, []byte("shorts: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadScript(path, ""); !errors.Is(err, ErrWalk) {
		t.Errorf("a script that is not yaml: %v", err)
	}
}

// sceneScript is a script with no set: one shot with an eye over places
// given in kilometres, one with no camera at all.
const sceneScript = `places: {unit: km, at: {the star: [0, 0, 0],
  the hoop: [500, 0, 0]}}
shorts:
  - name: star
    title: A star
    shots:
      - picture: down.png
        eye: {from: [0, 627, 2457], look-at: [0, 0, 0], up: [0, 1, 0],
          wide: 40}
        seconds: 2
        lit: drawing
        name: The star
        labels:
          - {thing: the star, words: the star}
          - {at: [500, 0, 0], words: the hoop, side: right}
          - {pixel: [10, 20], words: where it appears}
        scale: {thing: the star}
        sentence: A star.
      - picture: above.png
        seconds: 2
        tag: Computed, not seen
        name: Bent light
        labels:
          - {pixel: [100, 50], words: the ring appears here}
        scale: {pixels: 100, length: "10", unit: km}
        sentence: Light bends.
`

// sceneWith writes the scene script, with one text replaced, beside the
// marker stills, and reads it.
func sceneWith(t *testing.T, old, new string) (Script, error) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"down.png", "above.png"} {
		source, err := os.ReadFile(filepath.Join(markerDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), source,
			0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := strings.Replace(sceneScript, old, new, 1)
	if old != "" && text == sceneScript {
		t.Fatalf("%q is not in the script", old)
	}
	path := filepath.Join(dir, "script.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return ReadScript(path, "")
}

// TestSceneScript reads a script with no set: an eye's pinhole projects
// the places, the scale bar is in the scene's unit, pixel labels are
// taken as given, and its frames compose.
func TestSceneScript(t *testing.T) {
	script, err := sceneWith(t, "", "")
	if err != nil {
		t.Fatal(err)
	}
	short, _ := script.Short("star")
	shots := short.Shots()
	labels := shots[0].Labels()
	star := labels[0].At()
	if math.Abs(star.X()-319.5) > 1e-9 || math.Abs(star.Y()-179.5) > 1e-9 {
		t.Errorf("the star, looked at, is at %v, %v, not the middle",
			star.X(), star.Y())
	}
	if pixel := labels[2].At(); pixel.X() != 10 || pixel.Y() != 20 {
		t.Errorf("a pixel label is at %+v", pixel)
	}
	bar, err := shots[0].view.ScaleBar(*shots[0].bar, 200)
	if err != nil || !strings.HasSuffix(bar.Words(), " km") {
		t.Errorf("the bar says %q: %v", bar.Words(), err)
	}
	if shots[0].tag != lighting["drawing"] ||
		shots[1].tag != "Computed, not seen" ||
		shots[1].outright.words != "10 km" {
		t.Errorf("the tags and bar are %q, %q, %+v", shots[0].tag,
			shots[1].tag, shots[1].outright)
	}
	kept := NewStills()
	for _, i := range []int{0, short.Frames() - 1} {
		if _, err := short.Frame(i, kept); err != nil {
			t.Fatal(err)
		}
	}
}

// TestSceneScriptRefuses: what a script with no set cannot say is
// refused by name.
func TestSceneScriptRefuses(t *testing.T) {
	for _, wrong := range []struct{ old, new, named string }{
		{"  - picture: above.png", "  - picture: above.png\n" +
			"        camera: down", "names no set"},
		{"{pixel: [100, 50], words: the ring appears here}",
			"{thing: the star, words: the star}", "no camera"},
		{"{at: [500, 0, 0], words: the hoop, side: right}",
			"{toward: [10, 10], words: up}", "on a set only"},
		{"{thing: the star, words: the star}",
			"{thing: the moon, words: the moon}", "the moon"},
		{"places: {unit: km,", "places: {", "unit, at"},
		{"the hoop: [500, 0, 0]", "the hoop: [500, 0]", "x, y"},
		{"up: [0, 1, 0]", "up: [0, 0, 0]", "an up across"},
		{"wide: 40", "wide: 0", "an angle"},
		{"look-at: [0, 0, 0]", "look-at: [0, 0]", "each x, y and z"},
		{"scale: {thing: the star}", "scale: {thing: the moon}", "the moon"},
		{"scale: {pixels: 100, length: \"10\", unit: km}",
			"scale: {thing: the star}", "no camera"},
	} {
		_, err := sceneWith(t, wrong.old, wrong.new)
		if !errors.Is(err, ErrWalk) || !strings.Contains(err.Error(),
			wrong.named) {
			t.Errorf("%q for %q: %v", wrong.new, wrong.old, err)
		}
	}
}

// TestSceneNoPlaces: a script with no set and no places takes pixel
// labels; an eye in it has no unit and is refused.
func TestSceneNoPlaces(t *testing.T) {
	text := "places: {unit: km, at: {the star: [0, 0, 0],\n" +
		"  the hoop: [500, 0, 0]}}\n"
	_, err := sceneWith(t, text, "")
	if !errors.Is(err, ErrWalk) || !strings.Contains(err.Error(), "unit") {
		t.Errorf("an eye with no unit: %v", err)
	}
	if got := grouped(10000); got != "10,000" {
		t.Errorf("ten thousand is %q", got)
	}
	if got := grouped(2.5); got != "2.5" {
		t.Errorf("two and a half is %q", got)
	}
}

// TestHost lays a host's clip into its corner, looped, and narrows the
// sentence for it; a host folder with no frames is refused.
func TestHost(t *testing.T) {
	dir := t.TempDir()
	host := filepath.Join(dir, "host")
	if err := os.Mkdir(host, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "host: host\n" + sceneScript
	path := filepath.Join(dir, "script.yaml")
	for _, name := range []string{"down.png", "above.png"} {
		source, _ := os.ReadFile(filepath.Join(markerDir, name))
		if err := os.WriteFile(filepath.Join(dir, name), source,
			0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadScript(path, ""); !errors.Is(err, ErrWalk) {
		t.Errorf("a host with no frames: %v", err)
	}
	clip := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range clip.Pix {
		clip.Pix[i] = 0xff
	}
	file, err := os.Create(filepath.Join(host, "000.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, clip); err != nil {
		t.Fatal(err)
	}
	file.Close()
	script, err := ReadScript(path, "")
	if err != nil {
		t.Fatal(err)
	}
	short, _ := script.Short("star")
	frame, err := short.Frame(short.Frames()-1, NewStills())
	if err != nil {
		t.Fatal(err)
	}
	middle := hostBox.Min.Add(hostBox.Size().Div(2))
	if got := frame.RGBAAt(middle.X, middle.Y); got.R != 0xff ||
		got.G != 0xff || got.B != 0xff {
		t.Errorf("the host's corner is %v, not the clip", got)
	}
	if sentenceWidth(true) >= sentenceWidth(false) {
		t.Error("the sentence is not narrowed for a host")
	}
}
