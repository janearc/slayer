package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/rig"
	"github.com/janearc/slayer/improv/moonlake/show"
	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

// childEnv, when set to one, makes the test binary run the command's
// main and exit, so the paths that stop the process can be tried in a
// child.
const childEnv = "UNTRACED_TEST_CHILD"

// TestMain runs the command itself in a child, and the tests otherwise.
func TestMain(m *testing.M) {
	if os.Getenv(childEnv) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// beatsEvery is a score's yaml with count beats half a second apart from
// the first second, a bar every two seconds.
func beatsEvery(count int) string {
	var out strings.Builder
	out.WriteString("score:\n  duration: 400\n  beats:\n    times: [")
	for i := 0; i < count; i++ {
		if i > 0 {
			out.WriteString(", ")
		}
		fmt.Fprintf(&out, "%g", 1+0.5*float64(i))
	}
	out.WriteString("]\n")
	return out.String()
}

// writtenText writes text to a file in a temporary folder and says where.
func writtenText(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "written.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// testScore is the score of 400 beats, so bars one to a hundred exist.
func testScore(t *testing.T) *show.Score {
	t.Helper()
	scoreText := beatsEvery(400)
	score, err := show.ReadScore(writtenText(t, scoreText))
	if err != nil {
		t.Fatal(err)
	}
	return score
}

// testSheet is a cue sheet that sets the lasers at bar one, the strobes
// and the searchlights at bar two and the ship's light at bar three.
func testSheet(t *testing.T, score *show.Score) *show.Sheet {
	t.Helper()
	sheet, err := show.ReadSheet(writtenText(t, "cues:\n"+
		"  - {at_sec: 0, master: 1, moon: 1, moon_alt: 6}\n"+
		"  - {at: \"1.1\", rig: 1, beams: 4}\n"+
		"  - {at: \"2.1\", strobes: grid16, search: 1}\n"+
		"  - {at: \"3.1\", overhead: 1}\n"), score)
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

// testScript writes an explainer script of two sentences and says where.
func testScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "explain.yaml")
	text := "says:\n  - {at: \"1.1\", says: \"Two beams, low.\"}\n" +
		"  - {at: \"3.1\", says: \"The ship comes in.\"}\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestMovesFromShots: each shot is a move on the score's bar line, the
// first from the whole window, each from where the last left it, and an
// out shot back to the whole window.
func TestMovesFromShots(t *testing.T) {
	score := testScore(t)
	pane := overhead(t)
	moves, err := movesOf(pane, aboveShots, score, layout, 16.0/9)
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != len(aboveShots) {
		t.Fatalf("%d moves for %d shots", len(moves), len(aboveShots))
	}
	// bar 1 starts at the first beat, 1 s; bar 2 is two seconds on
	if moves[0].from != 1 || moves[0].until != 3 {
		t.Errorf("the first move runs %v to %v s, want 1 to 3", moves[0].from,
			moves[0].until)
	}
	if moves[1].from != 17 || moves[1].until != 19 {
		t.Errorf("the second move runs %v to %v s, want 17 to 19",
			moves[1].from, moves[1].until)
	}
	in := moves[0].end
	if moves[0].start != pane.part || in.Dx() != 100 || in.Dy() != 56 {
		t.Errorf("the first move is %v to %v, want 100 by 56", moves[0].start,
			in)
	}
	if !in.In(pane.still.Rect) {
		t.Errorf("the window %v is off the still %v", in, pane.still.Rect)
	}
	if moves[1].start != in || moves[1].end != pane.part {
		t.Errorf("the second move is %v to %v, want %v to the whole", moves[1].start,
			moves[1].end, in)
	}
}

// TestMovesRefusals: a bar the score lacks and a place the camera does
// not see are each an error naming what is wrong.
func TestMovesRefusals(t *testing.T) {
	score := testScore(t)
	pane := overhead(t)
	if _, err := movesOf(pane, []shot{{bar: 500, share: 1}}, score, layout, 1); err == nil ||
		!strings.Contains(err.Error(), "bar 500") {
		t.Errorf("a bar off the grid gave %v", err)
	}
	if _, err := movesOf(pane, []shot{{bar: 100, share: 1}}, score, layout, 1); err == nil {
		t.Error("a shot whose next bar is off the grid was taken")
	}
	behind := func(rig.Layout) rig.Point { return rig.Point{0, 0, 9000} }
	if _, err := movesOf(pane, []shot{{bar: 2, on: behind, share: 0.5}}, score,
		layout, 1); err == nil ||
		!strings.Contains(err.Error(), "does not see") {
		t.Errorf("a place behind the camera gave %v", err)
	}
}

// TestAroundCentresAndKeepsToTheStill: a window on a place is centred on
// it, and one that would run off the still is moved back on.
func TestAroundCentresAndKeepsToTheStill(t *testing.T) {
	pane := overhead(t)
	x, y, _ := pane.onPane(rig.Point{}, pane.still.Rect.Size())
	got, err := pane.around(rig.Point{}, 0.5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if want := image.Rect(int(x)-50, int(y)-50, int(x)+50, int(y)+50); got != want {
		t.Errorf("the window is %v, want %v", got, want)
	}
	// near the corner of the still, the window is kept inside it
	corner := rig.Point{2900, 2900, 0}
	got, err = pane.around(corner, 0.5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !got.In(pane.still.Rect) || got.Dx() != 100 {
		t.Errorf("a window at the corner is %v", got)
	}
}

// TestCueRingFallsAfterItsCue: the ring is full when a cue that sets the
// part fires, falls by a third in a third of a second and a bit, is gone
// after three of those, and is none for another part or before the cue.
func TestCueRingFallsAfterItsCue(t *testing.T) {
	fired := testSheet(t, testScore(t)).Fired()
	// the lasers' cue is at bar 1, 1 s
	for _, c := range []struct {
		part    string
		seconds float64
		want    float64
	}{
		{show.PartLasers, 1, 1},
		{show.PartLasers, 1 + cueFall, math.Exp(-1)},
		{show.PartLasers, 1 + cueSpan + 0.01, 0},
		{show.PartLasers, 0.5, 0},
		{show.PartOverhead, 1, 0},
		{show.PartStrobes, 3, 1},
	} {
		if got := cueRing(fired, c.part, c.seconds); !within(got, c.want) {
			t.Errorf("the %s ring at %v s is %v, want %v", c.part, c.seconds,
				got, c.want)
		}
	}
}

// TestExplainerLampsFollowTheSheet: the six lamps are the panel's, in
// its order, the lasers' lit once their cue has and dark before, the
// strobes' full on a firing frame, and each lamp's ring follows its cue.
func TestExplainerLampsFollowTheSheet(t *testing.T) {
	score := testScore(t)
	sheet := testSheet(t, score)
	explainer, err := explainerOf(score, sheet, testScript(t))
	if err != nil {
		t.Fatal(err)
	}
	before, err := explainer.PanelAt(0.5)
	if err != nil {
		t.Fatal(err)
	}
	labels := []string{"Lasers", "Strobes", "Backlights", "Searchlights",
		"Moon", "Ship's beam"}
	for i, lamp := range before.Lamps() {
		if lamp.Label() != labels[i] {
			t.Errorf("lamp %d is %q, want %q", i, lamp.Label(), labels[i])
		}
		if i != 4 && lamp.Level() != 0 {
			t.Errorf("the %s is at %v before any cue", lamp.Label(), lamp.Level())
		}
	}
	// the lasers' cue fires at 1 s: the lamp is lit and its ring full
	after, err := explainer.PanelAt(1.5)
	if err != nil {
		t.Fatal(err)
	}
	lasers := after.Lamps()[0]
	if lasers.Level() <= 0 || lasers.Ring() <= 0 {
		t.Errorf("the lasers lamp is %v with ring %v after its cue",
			lasers.Level(), lasers.Ring())
	}
	// find a frame where a strobe fires: its lamp is then full
	fired := -1.0
	for frame := 0; frame < 24*10; frame++ {
		seconds := 3 + float64(frame)/filmRate
		moment, err := sheet.At(seconds, filmRate)
		if err != nil {
			t.Fatal(err)
		}
		if moment.Strobe() {
			fired = seconds
			break
		}
	}
	if fired < 0 {
		t.Fatal("no strobe fired in ten seconds of grid16 strobes")
	}
	panel, err := explainer.PanelAt(fired)
	if err != nil {
		t.Fatal(err)
	}
	if got := panel.Lamps()[1].Level(); got != 1 {
		t.Errorf("the strobes lamp on a firing frame is %v, want 1", got)
	}
	sentence, saying := after.Says()
	if !saying || sentence.Words() != "Two beams, low." {
		t.Errorf("the panel says %q (%v)", sentence.Words(), saying)
	}
}

// TestExplainerOfRefusals: a missing script is an error.
func TestExplainerOfRefusals(t *testing.T) {
	score := testScore(t)
	sheet := testSheet(t, score)
	missingScript := filepath.Join(t.TempDir(), "none.yaml")
	if _, err := explainerOf(score, sheet,
		missingScript); err == nil {
		t.Error("a missing script was read")
	}
}

// TestTonesAreTheLampsColours: each kind of fixture is the colour of
// its lamp's theme role, as light.
func TestTonesAreTheLampsColours(t *testing.T) {
	tones, err := tonesOf()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		got  [3]float64
		role string
	}{{"lasers", tones.lasers, "ok"}, {"strobes", tones.strobes, "ink"},
		{"floods", tones.floods, "warn"}, {"search", tones.search, "border"},
		{"ship", tones.ship, "ink"}} {
		want, err := explain.ThemeColour(c.role)
		if err != nil {
			t.Fatal(err)
		}
		if c.got != [3]float64{float64(want.R) / 255, float64(want.G) / 255,
			float64(want.B) / 255} {
			t.Errorf("the %s tone is %v, want the %s role %v", c.name, c.got,
				c.role, want)
		}
	}
}

// TestCaptionTypeIsNeverUnderTheSmallest: a pane's name is set at the
// smallest type at any width, since the house's least is over the
// picked size.
func TestCaptionTypeIsNeverUnderTheSmallest(t *testing.T) {
	for _, width := range []int{200, 960, 1920} {
		got := captionTypeFor(image.Rect(0, 0, width, 100))
		if got != smallestType {
			t.Errorf("at %d across the type is %v, want %v", width, got,
				smallestType)
		}
	}
}

// TestDrawPanelLightsTheBeatsSquare: at a beat the first square is the
// ink colour in full, the last is a dim ok, and the pane is filled with
// the ground.
func TestDrawPanelLightsTheBeatsSquare(t *testing.T) {
	score := testScore(t)
	explainer, err := explainerOf(score, testSheet(t, score), testScript(t))
	if err != nil {
		t.Fatal(err)
	}
	pane := image.Rect(0, 0, 960, 540)
	picture := blank(960, 540)
	// the second bar's first beat falls at 3 s
	if err := drawPanel(picture, pane, explainer, 3); err != nil {
		t.Fatal(err)
	}
	ink, _ := explain.ThemeColour("ink")
	ground, _ := explain.ThemeColour("ground")
	ok, _ := explain.ThemeColour("ok")
	first := picture.RGBAAt(int(panelMargin+squareSide/2), int(squareTop+squareSide/2))
	last := picture.RGBAAt(int(panelMargin+3*(squareSide+squareGap)+squareSide/2),
		int(squareTop+squareSide/2))
	if first != ink {
		t.Errorf("the beat's square is %v, want the ink %v", first, ink)
	}
	if last == ok || last == ground {
		t.Errorf("an unlit square is %v, want a dim ok between %v and %v", last,
			ground, ok)
	}
	if corner := picture.RGBAAt(959, 539); corner != ground {
		t.Errorf("the pane's corner is %v, want the ground %v", corner, ground)
	}
	// before the grid begins, the bar says intro and no square is lit
	early := blank(960, 540)
	if err := drawPanel(early, pane, explainer, 0.2); err != nil {
		t.Fatal(err)
	}
	if got := early.RGBAAt(int(panelMargin+squareSide/2), int(squareTop+squareSide/2)); got == ink {
		t.Error("a square was lit before the grid began")
	}
	// a pane half the width scales the squares to match
	small := blank(480, 270)
	if err := drawPanel(small, image.Rect(0, 0, 480, 270), explainer, 3); err != nil {
		t.Fatal(err)
	}
	if got := small.RGBAAt(int(panelMargin/2+squareSide/4), int(squareTop/2+squareSide/4)); got != ink {
		t.Errorf("the half size square is %v, want the ink %v", got, ink)
	}
}

// TestEdgeNameStepsPastTheNamesTaken: a name is set at the nearer side
// level with its fixture, and one that would overlap an earlier name is
// stepped below it.
func TestEdgeNameStepsPastTheNamesTaken(t *testing.T) {
	place := image.Rect(0, 0, 400, 300)
	picture := blank(400, 300)
	size := 28.0
	left, err := edgeName(picture, place, named{"lasers", image.Pt(50, 100)}, size, nil)
	if err != nil {
		t.Fatal(err)
	}
	if left.Min.X != captionInset || left.Min.Y != 74 || left.Dy() != 52 {
		t.Errorf("the first name's box is %v, want it at 12, 74 and 52 high", left)
	}
	again, err := edgeName(picture, place, named{"strobes", image.Pt(50, 100)}, size,
		[]image.Rectangle{left})
	if err != nil {
		t.Fatal(err)
	}
	if again.Min.Y != left.Max.Y+captionInset/2 {
		t.Errorf("the second name is at %d, want %d", again.Min.Y,
			left.Max.Y+captionInset/2)
	}
	right, err := edgeName(picture, place, named{"ship", image.Pt(350, 100)}, size, nil)
	if err != nil {
		t.Fatal(err)
	}
	if right.Max.X != 400-captionInset {
		t.Errorf("a name on the right ends at %d, want %d", right.Max.X,
			400-captionInset)
	}
	if lit(picture) == 0 {
		t.Error("the names and their leaders drew nothing")
	}
}

// writeTraced writes the four passes of a camera, each of one value,
// at a size, into a directory.
func writeTraced(t *testing.T, dir, camera string, width, height int, values map[string]float32) {
	t.Helper()
	for name, value := range values {
		picture := frame.New(width, height)
		for i := range picture.Pix {
			picture.Pix[i].R, picture.Pix[i].G, picture.Pix[i].B = value, 2*value, 3*value
		}
		if err := raw.Write(filepath.Join(dir, camera+"-"+name+".pfm"), picture); err != nil {
			t.Fatal(err)
		}
	}
}

// TestTracedOfReadsAndRefuses: the four passes are read into one traced
// camera, and a pass of another size, or one missing, is refused.
func TestTracedOfReadsAndRefuses(t *testing.T) {
	dir := t.TempDir()
	values := map[string]float32{"nodisc": 0.5, "back-left": 1, "back-right": 2,
		"overhead": 4}
	writeTraced(t, dir, "north", 2, 3, values)
	got, err := tracedOf(dir, "north")
	if err != nil {
		t.Fatal(err)
	}
	if got.size != image.Pt(2, 3) || len(got.plate) != 18 || len(got.passes) != 3 {
		t.Fatalf("read size %v, %d plate numbers, %d passes", got.size,
			len(got.plate), len(got.passes))
	}
	if got.plate[0] != 0.5 || got.plate[1] != 1 || got.plate[2] != 1.5 ||
		got.passes["overhead"][17] != 12 || got.passes["back-right"][3] != 2 {
		t.Errorf("read the wrong numbers: plate %v", got.plate[:3])
	}
	if _, err := tracedOf(dir, "ridge"); err == nil {
		t.Error("a camera with no passes was read")
	}
	writeTraced(t, dir, "ridge", 2, 3, values)
	writeTraced(t, dir, "ridge", 3, 3, map[string]float32{"overhead": 1})
	if _, err := tracedOf(dir, "ridge"); err == nil ||
		!strings.Contains(err.Error(), "one size") {
		t.Errorf("passes of two sizes gave %v", err)
	}
}

// TestTracedPanesNeedPasses: with no folder there are no traced panes;
// with one, both cameras are read; a camera lacking is an error.
func TestTracedPanesNeedPasses(t *testing.T) {
	above, moon, err := tracedPanes(asked{})
	if above != nil || moon != nil || err != nil {
		t.Errorf("no passes gave %v %v %v", above, moon, err)
	}
	dir := t.TempDir()
	values := map[string]float32{"nodisc": 1, "back-left": 1, "back-right": 1,
		"overhead": 1}
	writeTraced(t, dir, "north", 1, 1, values)
	if _, _, err := tracedPanes(asked{passes: dir}); err == nil {
		t.Error("a ridge with no passes was read")
	}
	writeTraced(t, dir, "ridge", 1, 1, values)
	above, moon, err = tracedPanes(asked{passes: dir})
	if err != nil || above == nil || moon == nil {
		t.Errorf("both cameras gave %v %v %v", above, moon, err)
	}
	if _, _, err := tracedPanes(asked{passes: filepath.Join(dir, "none")}); err == nil {
		t.Error("a missing folder was read")
	}
}

// TestAveragedTakesTheMeanOfWhatItCovers: a four by four down to two by
// two is each block's mean; a window past the light's edge counts its
// missing pixels as dark.
func TestAveragedTakesTheMeanOfWhatItCovers(t *testing.T) {
	light := make([]float64, 4*4*tracedChannel)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			for c := 0; c < tracedChannel; c++ {
				light[(y*4+x)*tracedChannel+c] = float64(x+4*y) * float64(c+1)
			}
		}
	}
	got := averaged(light, image.Pt(4, 4), image.Rect(0, 0, 4, 4), image.Pt(2, 2))
	// the first block is 0, 1, 4 and 5: mean 2.5, in three channels
	if got[0] != 2.5 || got[1] != 5 || got[2] != 7.5 {
		t.Errorf("the first block is %v, want 2.5 5 7.5", got[:3])
	}
	// the last block is 10, 11, 14 and 15: 12.5
	if got[3*tracedChannel] != 12.5 {
		t.Errorf("the last block is %v, want 12.5", got[3*tracedChannel])
	}
	off := averaged(light, image.Pt(4, 4), image.Rect(2, 2, 6, 6), image.Pt(2, 2))
	if off[0] != 12.5 || off[3*tracedChannel] != 0 {
		t.Errorf("a window past the edge is %v and %v, want 12.5 and 0", off[0],
			off[3*tracedChannel])
	}
	same := averaged(light, image.Pt(4, 4), image.Rect(1, 1, 3, 3), image.Pt(4, 4))
	if same[0] != 5 || same[tracedChannel] != 5 || same[2*tracedChannel] != 6 {
		t.Errorf("an enlargement is %v %v %v, want 5 5 6", same[0], same[tracedChannel],
			same[2*tracedChannel])
	}
}

// TestTracedAtOverheadAndExposure: the ship's light adds to the plate by
// its level, an exposure cue brightens the print, and a pass not in the
// passes is refused.
func TestTracedAtOverheadAndExposure(t *testing.T) {
	pane := traced{size: image.Pt(1, 1), plate: []float64{1e-8, 1e-8, 1e-8},
		passes: map[string][]float64{"overhead": {0, 0, 0},
			"back-left": {0, 0, 0}, "back-right": {0, 0, 0}}}
	whole, one := image.Rect(0, 0, 1, 1), image.Pt(1, 1)
	plain, err := pane.at(cues{master: 1}, whole, one)
	if err != nil {
		t.Fatal(err)
	}
	brighter, err := pane.at(cues{master: 1, exposure: 1}, whole, one)
	if err != nil {
		t.Fatal(err)
	}
	if brighter.RGBAAt(0, 0).G <= plain.RGBAAt(0, 0).G {
		t.Errorf("a stop more exposure is %v, no brighter than %v",
			brighter.RGBAAt(0, 0), plain.RGBAAt(0, 0))
	}
	pane.passes["overhead"] = []float64{1e-6, 1e-6, 1e-6}
	lit, err := pane.at(cues{master: 1, overhead: 1}, whole, one)
	if err != nil {
		t.Fatal(err)
	}
	if lit.RGBAAt(0, 0).B <= plain.RGBAAt(0, 0).B {
		t.Errorf("the ship's light adds nothing: %v against %v",
			lit.RGBAAt(0, 0), plain.RGBAAt(0, 0))
	}
	delete(pane.passes, "overhead")
	if _, err := pane.at(cues{overhead: 1}, whole, one); err == nil {
		t.Error("a missing overhead pass was summed")
	}
}

// TestReadPNGIsRGBAAtNought: a picture with a non-zero origin is read
// with its top left at nought, and a missing or a broken file is refused.
func TestReadPNGIsRGBAAtNought(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.png")
	source := image.NewRGBA(image.Rect(0, 0, 3, 2))
	source.SetRGBA(2, 1, color.RGBA{9, 8, 7, 255})
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, source); err != nil {
		t.Fatal(err)
	}
	file.Close()
	got, err := readPNG(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Rect != image.Rect(0, 0, 3, 2) || got.RGBAAt(2, 1) != (color.RGBA{9, 8, 7, 255}) {
		t.Errorf("read %v with %v at 2 1", got.Rect, got.RGBAAt(2, 1))
	}
	if _, err := readPNG(filepath.Join(dir, "none.png")); err == nil {
		t.Error("a missing picture was read")
	}
	os.WriteFile(filepath.Join(dir, "bad.png"), []byte("not a picture"), 0o644)
	if _, err := readPNG(filepath.Join(dir, "bad.png")); err == nil ||
		!strings.Contains(err.Error(), "bad.png") {
		t.Errorf("a broken picture gave %v", err)
	}
}

// TestSongSpanOfAMissingSong refuses a song that is not there.
func TestSongSpanOfAMissingSong(t *testing.T) {
	missingSong := filepath.Join(t.TempDir(), "none.wav")
	if _, err := songSpan(missingSong, 0, 1); err == nil {
		t.Error("a missing song was read")
	}
}

// child runs the command in a child process with arguments, and says its
// exit status and what it wrote to stderr.
func child(t *testing.T, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), childEnv+"=1")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), stderr.String()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0, stderr.String()
}

// TestCommandSaysWhyItStops: a layout that is neither wide nor tall and
// a missing score each stop with status one and a message that names
// the command.
func TestCommandSaysWhyItStops(t *testing.T) {
	status, stderr := child(t, "-layout", "square")
	if status != 1 || !strings.Contains(stderr, "untraced:") ||
		!strings.Contains(stderr, "say wide or tall") {
		t.Errorf("a square layout: status %d, stderr %q", status, stderr)
	}
	missingScore := filepath.Join(t.TempDir(), "none.yaml")
	status, stderr = child(t, "-score", missingScore)
	if status != 1 || !strings.Contains(stderr, "untraced:") {
		t.Errorf("a missing score: status %d, stderr %q", status, stderr)
	}
}

// TestPartsOfRefusals: each source that is wrong stops partsOf at its
// own step, with an error from it.
func TestPartsOfRefusals(t *testing.T) {
	dir := t.TempDir()
	score := filepath.Join(dir, "score.yaml")
	scoreText := beatsEvery(400)
	os.WriteFile(score, []byte(scoreText), 0o644)
	cues := filepath.Join(dir, "cues.yaml")
	os.WriteFile(cues, []byte("cues:\n  - {at_sec: 0, master: 1}\n"), 0o644)
	places := filepath.Join(dir, "places.csv")
	os.WriteFile(places, []byte("of,kind,east,north,height\n"), 0o644)
	good := asked{layout: "wide", score: score, cues: cues, places: places,
		script: testScript(t), stills: dir}
	for _, c := range []struct {
		name   string
		change func(*asked)
		want   string
	}{
		{"layout", func(a *asked) { a.layout = "round" }, "say wide or tall"},
		{"score", func(a *asked) { a.score = filepath.Join(dir, "none") }, "no such file"},
		{"cues", func(a *asked) { a.cues = filepath.Join(dir, "none") }, "no such file"},
		{"places", func(a *asked) { a.places = filepath.Join(dir, "none") }, "no such file"},
		{"bad places", func(a *asked) { a.places = score }, "places table"},
		{"script", func(a *asked) { a.script = filepath.Join(dir, "none") }, "no such file"},
		{"stills", func(a *asked) {}, "walk-north.yaml"},
	} {
		given := good
		c.change(&given)
		if _, err := partsOf(given); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: gave %v, want it to mention %q", c.name, err, c.want)
		}
	}
}

// TestDrawnViewsNeedTheSets: a recipe folder with no ridge set, or with a
// set but no still, is an error naming the file.
func TestDrawnViewsNeedTheSets(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := drawnViews(asked{stills: dir, layout: "wide"}, walk.Places{}); err == nil {
		t.Error("a folder with no sets was read")
	}
	if _, err := viewOf(dir, "x.yaml", "cam", "x.png", walk.Places{}); err == nil ||
		!strings.Contains(err.Error(), "x.yaml") {
		t.Errorf("a missing set gave %v", err)
	}
}
