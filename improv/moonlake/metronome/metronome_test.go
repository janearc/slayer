package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/improv/moonlake/show"
)

// scoreText is a score with beats a second apart from one second, the
// first three measured and the fourth counted through.
const scoreText = `score:
  duration: 3
  beats:
    times: [1.0, 2.0, 3.0, 4.0]
    measured: [1, 1, 1, 0]
`

// cuesText is a sheet with a kill cue and a plain cue.
const cuesText = `cues:
  - {at_sec: 1.0, rig: 0}
  - {at_sec: 2.0, beams: 2}
`

// writeFiles writes the score and the cue sheet into a temporary
// directory and returns their paths.
func writeFiles(t *testing.T, score, cues string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	scorePath := filepath.Join(dir, "score.yaml")
	cuesPath := filepath.Join(dir, "cues.yaml")
	if err := os.WriteFile(scorePath, []byte(score), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cuesPath, []byte(cues), 0o644); err != nil {
		t.Fatal(err)
	}
	return scorePath, cuesPath
}

// runDraw runs the draw verb, declared as main declares it, with the
// given arguments. It returns the status, standard output and standard
// error.
func runDraw(t *testing.T, args ...string) (int, []byte, string) {
	t.Helper()
	tool := pzm.New("metronome", "check the score against its audio",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("raw RGB frames on stdout"),
		verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("draw", verb.NoShort, "write the visual metronome",
		"Writes RGB24 frames.", "Example_draw", facts, drawVerb,
		pzm.Value("score", verb.NoShort, "score YAML"),
		pzm.Value("cues", verb.NoShort, "cue sheet YAML"),
		pzm.Value("fps", verb.NoShort, "frames per second"),
		pzm.Value("width", verb.NoShort, "frame width"),
		pzm.Value("height", verb.NoShort, "frame height"))
	var out, errs bytes.Buffer
	status := tool.RunWith(append([]string{"draw"}, args...), strings.NewReader(""), &out, &errs)
	return status, out.Bytes(), errs.String()
}

// pixel reads one rgb24 pixel from frame number n of a stream of frames
// of the given width and height.
func pixel(data []byte, n, width, height, x, y int) [3]byte {
	at := (n*width*height + y*width + x) * 3
	return [3]byte{data[at], data[at+1], data[at+2]}
}

// TestDrawFrames draws six frames and checks the stream's size and the
// colours of the squares, the gap, and the cue band.
func TestDrawFrames(t *testing.T) {
	scorePath, cuesPath := writeFiles(t, scoreText, cuesText)
	status, out, errs := runDraw(t, "--score", scorePath, "--cues", cuesPath,
		"--fps", "2", "--width", "320", "--height", "180")
	if status != 0 {
		t.Fatalf("status %d: %s", status, errs)
	}
	const w, h = 320, 180
	if len(out) != 6*w*h*3 {
		t.Fatalf("got %d bytes, want %d", len(out), 6*w*h*3)
	}
	// Frame 0 is at 0 s, before the first beat: the squares rest and the
	// corner is the ground.
	if got := pixel(out, 0, w, h, 77, 94); got != [3]byte{55, 55, 60} {
		t.Errorf("resting beat one square = %v", got)
	}
	if got := pixel(out, 0, w, h, 132, 94); got != [3]byte{18, 48, 24} {
		t.Errorf("resting beat two square = %v", got)
	}
	if got := pixel(out, 0, w, h, 0, 0); got != [3]byte{8, 8, 10} {
		t.Errorf("ground = %v", got)
	}
	// Frame 2 is at 1 s, on the first beat: beat one is white.
	if got := pixel(out, 2, w, h, 77, 94); got != [3]byte{255, 255, 255} {
		t.Errorf("struck beat one square = %v", got)
	}
	// The kill cue fires at 1 s: the band is red at its brightest.
	if got := pixel(out, 2, w, h, 2, 135); got != [3]byte{255, 40, 40} {
		t.Errorf("kill band = %v", got)
	}
	// The plain cue fires at 2 s, frame 4: the band is blue.
	if got := pixel(out, 4, w, h, 2, 135); got != [3]byte{80, 160, 255} {
		t.Errorf("cue band = %v", got)
	}
	// Frame 0 has no cue showing.
	if got := pixel(out, 0, w, h, 2, 135); got != [3]byte{8, 8, 10} {
		t.Errorf("band before any cue = %v", got)
	}
}

// TestDrawDryRun checks that a dry run writes nothing.
func TestDrawDryRun(t *testing.T) {
	scorePath, cuesPath := writeFiles(t, scoreText, cuesText)
	status, out, errs := runDraw(t, "--dry-run", "--score", scorePath,
		"--cues", cuesPath, "--fps", "2", "--width", "320", "--height", "180")
	if status != 0 || len(out) != 0 {
		t.Fatalf("status %d, %d bytes, %s", status, len(out), errs)
	}
}

// TestDrawRefusals covers the arguments and files draw turns away.
func TestDrawRefusals(t *testing.T) {
	scorePath, cuesPath := writeFiles(t, scoreText, cuesText)
	badScore, _ := writeFiles(t, "score: [", cuesText)
	_, badCues := writeFiles(t, scoreText, "cues: [")
	size := []string{"--fps", "2", "--width", "320", "--height", "180"}
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no score", append([]string{"--cues", cuesPath}, size...), "draw needs score and cues"},
		{"no cues", append([]string{"--score", scorePath}, size...), "draw needs score and cues"},
		{"stray argument", append([]string{"--score", scorePath, "--cues", cuesPath, "extra"}, size...), "no arguments"},
		{"missing score file", append([]string{"--score", filepath.Join(t.TempDir(), "none.yaml"), "--cues", cuesPath}, size...), "show"},
		{"unreadable score", append([]string{"--score", badScore, "--cues", cuesPath}, size...), "the score"},
		{"unreadable cues", append([]string{"--score", scorePath, "--cues", badCues}, size...), "show"},
		{"fps not a number", []string{"--score", scorePath, "--cues", cuesPath, "--fps", "x", "--width", "320", "--height", "180"}, "positive fps"},
		{"width too small", []string{"--score", scorePath, "--cues", cuesPath, "--fps", "2", "--width", "319", "--height", "180"}, "positive fps"},
		{"height too small", []string{"--score", scorePath, "--cues", cuesPath, "--fps", "2", "--width", "320", "--height", "179"}, "positive fps"},
		{"fps zero", []string{"--score", scorePath, "--cues", cuesPath, "--fps", "0", "--width", "320", "--height", "180"}, "positive fps"},
	}
	for _, c := range cases {
		status, out, errs := runDraw(t, c.args...)
		if status != 1 || len(out) != 0 || !strings.Contains(errs, c.want) {
			t.Errorf("%s: status %d, %d bytes, stderr %q, want it to contain %q",
				c.name, status, len(out), errs, c.want)
		}
	}
}

// failingWriter fails every write.
type failingWriter struct{}

// Write always fails.
func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

// callWith builds the call for render by running a verb that hands its
// call to the test, with the output writer given.
func callWith(t *testing.T, out *failingWriter, args []string) error {
	t.Helper()
	var got error
	tool := pzm.New("metronome", "check the score against its audio",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("raw RGB frames on stdout"),
		verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("draw", verb.NoShort, "write the visual metronome",
		"Writes RGB24 frames.", "Example_draw", facts,
		func(call pzm.Call) error { got = drawVerb(call); return got },
		pzm.Value("score", verb.NoShort, "score YAML"),
		pzm.Value("cues", verb.NoShort, "cue sheet YAML"),
		pzm.Value("fps", verb.NoShort, "frames per second"),
		pzm.Value("width", verb.NoShort, "frame width"),
		pzm.Value("height", verb.NoShort, "frame height"))
	var errs bytes.Buffer
	tool.RunWith(append([]string{"draw"}, args...), strings.NewReader(""), out, &errs)
	return got
}

// TestDrawWriteErrors checks that a failing output is reported, both
// when the buffer fills mid-stream and when only the last flush fails.
func TestDrawWriteErrors(t *testing.T) {
	scorePath, cuesPath := writeFiles(t, scoreText, cuesText)
	cases := []struct{ name, fps string }{
		{"flush at the end", "2"},
		{"buffer full mid-stream", "4"},
	}
	for _, c := range cases {
		err := callWith(t, &failingWriter{}, []string{"--score", scorePath,
			"--cues", cuesPath, "--fps", c.fps, "--width", "320", "--height", "180"})
		if err == nil || !strings.Contains(err.Error(), "disk full") {
			t.Errorf("%s: err = %v, want disk full", c.name, err)
		}
	}
}

// newFrame makes a frame of the minimum size filled with the ground.
func newFrame() *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, 320, 180))
	fill(frame, frame.Bounds(), ground)
	return frame
}

// parsedScore parses the test score with the given extra header lines.
func parsedScore(t *testing.T, extra string) *show.Score {
	t.Helper()
	text := strings.Replace(scoreText, "score:\n", "score:\n"+extra, 1)
	score, err := show.ParseScore([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return score
}

// hasColour reports whether any pixel of the frame is the colour.
func hasColour(frame *image.RGBA, want color.RGBA) bool {
	for i := 0; i+3 < len(frame.Pix); i += 4 {
		if frame.Pix[i] == want.R && frame.Pix[i+1] == want.G && frame.Pix[i+2] == want.B {
			return true
		}
	}
	return false
}

// TestDrawFrameLabels checks the colour of the measured word before a
// beat, in the intro, on a measured beat, an estimated one and a counted
// one.
func TestDrawFrameLabels(t *testing.T) {
	green := color.RGBA{60, 255, 90, 255}
	faintInk := faint
	cases := []struct {
		name    string
		extra   string
		t       float64
		want    color.RGBA
		present bool
	}{
		{"measured is green", "", 1.5, green, true},
		{"estimated is green too", "  status: estimated\n", 1.5, green, true},
		{"counted through is faint", "", 4.5, faintInk, true},
		{"intro has no green word", "", 0.5, green, false},
		{"intro shows the drum time in faint", "", 0.5, faintInk, true},
	}
	for _, c := range cases {
		frame := newFrame()
		drawFrame(frame, c.t, parsedScore(t, c.extra), nil)
		// Green also fills the struck beat squares, so read only the
		// label rows, which sit above the squares.
		label := frame.SubImage(image.Rect(0, 40, 320, 70)).(*image.RGBA)
		if got := hasColour(label, c.want); got != c.present {
			t.Errorf("%s: colour present = %v, want %v", c.name, got, c.present)
		}
	}
}

// TestDrawFrameProgress checks the progress bar's length at half the song.
func TestDrawFrameProgress(t *testing.T) {
	frame := newFrame()
	drawFrame(frame, 1.5, parsedScore(t, ""), nil)
	bar := color.RGBA{90, 90, 100, 255}
	// The bar is two pixels tall at this size; half of 320 is 160.
	if got := frame.RGBAAt(159, 179); got != bar {
		t.Errorf("bar at 159 = %v", got)
	}
	if got := frame.RGBAAt(161, 179); got != ground {
		t.Errorf("past the bar = %v", got)
	}
}

// TestMix checks the blend and its clamping.
func TestMix(t *testing.T) {
	off, on := [3]float64{0, 100, 200}, [3]float64{100, 200, 0}
	cases := []struct {
		level float64
		want  color.RGBA
	}{
		{0, color.RGBA{0, 100, 200, 255}},
		{0.5, color.RGBA{50, 150, 100, 255}},
		{1, color.RGBA{100, 200, 0, 255}},
		{2, color.RGBA{100, 200, 0, 255}},
		{-1, color.RGBA{0, 100, 200, 255}},
	}
	for _, c := range cases {
		if got := mix(off, on, c.level); got != c.want {
			t.Errorf("mix level %g = %v, want %v", c.level, got, c.want)
		}
	}
}

// TestOutline checks that outline paints the edges and leaves the middle.
func TestOutline(t *testing.T) {
	frame := newFrame()
	c := color.RGBA{1, 2, 3, 255}
	outline(frame, image.Rect(10, 10, 30, 30), c)
	for _, p := range []image.Point{{10, 10}, {29, 29}, {20, 11}, {11, 20}, {28, 20}, {20, 28}} {
		if got := frame.RGBAAt(p.X, p.Y); got != c {
			t.Errorf("edge %v = %v", p, got)
		}
	}
	if got := frame.RGBAAt(20, 20); got != ground {
		t.Errorf("middle = %v", got)
	}
	if got := frame.RGBAAt(9, 9); got != ground {
		t.Errorf("outside = %v", got)
	}
}

// TestText checks that text is centred, scaled, and clipped to the frame.
func TestText(t *testing.T) {
	frame := newFrame()
	c := color.RGBA{250, 250, 250, 255}
	text(frame, "I", 160, 90, 2, c)
	count := 0
	for y := 0; y < 180; y++ {
		for x := 0; x < 320; x++ {
			if frame.RGBAAt(x, y) == c {
				count++
				if x < 160-7 || x > 160+7 || y < 90-13 || y > 90+13 {
					t.Fatalf("ink at %d,%d is outside the glyph box", x, y)
				}
			}
		}
	}
	// Every inked cell becomes a two by two block, so the count is a
	// multiple of four and not zero.
	if count == 0 || count%4 != 0 {
		t.Errorf("ink count %d", count)
	}
	// Words off the edge of the frame draw nothing and do not panic.
	off := newFrame()
	text(off, "wide", -100, -100, 1, c)
	if hasColour(off, c) {
		t.Error("text far outside the frame left ink")
	}
}
