package explain

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/show"
)

// The tests place moments with show's score, a real Placer: sixteen
// beats half a second apart from 2 s, bars one to four.
const grid = `score:
  duration: 20
  beats:
    times: [2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0, 5.5, 6.0, 6.5, 7.0, 7.5,
            8.0, 8.5, 9.0, 9.5]
`

// script is a script over grid: two shorts, three sentences, a chant
// line.
const script = `shorts:
  all: {from: {at: "1.1"}, until: {at: "4.4"}}
  start: {from: {at_sec: 2}, until: {at: "2.1"}}
says:
  - {at: "1.1", says: "First."}
  - {at: "2.1", says: "Second."}
  - {at_sec: 6.2, says: "Third."}
chant:
  - {from_sec: 3.0, until_sec: 4.7, words: "one two three"}
`

// gridScore is grid, read.
func gridScore(t *testing.T) *show.Score {
	t.Helper()
	score, err := show.ParseScore([]byte(grid))
	if err != nil {
		t.Fatal(err)
	}
	return score
}

// gridScript is script, read against grid.
func gridScript(t *testing.T) *Script {
	t.Helper()
	read, err := ParseScript([]byte(script), gridScore(t))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// the sentence standing at a moment is the script's latest at or before
// it, and none before the first; the shorts are where the script says.
func TestTheSentenceStandingIsTheLatest(t *testing.T) {
	read := gridScript(t)
	for seconds, want := range map[float64]string{
		2.0: "First.", 3.99: "First.", 4.0: "Second.", 6.19: "Second.",
		6.2: "Third.", 100: "Third.",
	} {
		said, ok := read.Says(seconds)
		if !ok || said.Words() != want {
			t.Errorf("at %v the script says %q (%v), not %q", seconds,
				said.Words(), ok, want)
		}
	}
	if said, ok := read.Says(1.99); ok {
		t.Errorf("before the first sentence it says %q", said.Words())
	}
	if first, _ := read.Says(2); first.From() != 2.0 {
		t.Errorf("the first sentence is from %v", first.From())
	}
	if from, until, ok := read.Short("all"); !ok || from != 2.0 ||
		until != 9.5 {
		t.Errorf("the short all is %v to %v (%v)", from, until, ok)
	}
	if names := read.Shorts(); len(names) != 2 || names[0] != "all" {
		t.Errorf("the shorts are %v", names)
	}
}

// the anti-test: a sentence or a short at a place the show lacks is
// refused by name, and so are a script out of order, an empty short, an
// empty chant line, and a script with no placer.
func TestABadScriptIsRefusedByName(t *testing.T) {
	score := gridScore(t)
	for raw, says := range map[string]string{
		`says: [{at: "9.1", says: "Late."}]`:                       `"9.1"`,
		`says: [{at: "2.1", says: "B."}, {at: "1.1", says: "A."}]`: "in order",
		`shorts: {x: {from: {at: "2.1"}, until: {at: "1.1"}}}`:     "empty",
		`shorts: {x: {from: {at: "7.1"}, until: {at: "1.1"}}}`:     `"7.1"`,
		`shorts: {x: {from: {at: "1.1"}, until: {at: "7.1"}}}`:     `"7.1"`,
		`chant: [{from_sec: 3, until_sec: 2, words: "no"}]`:        "empty",
		`says: [`: "the script",
	} {
		_, err := ParseScript([]byte(raw), score)
		if err == nil || !errors.Is(err, ErrExplain) ||
			!strings.Contains(err.Error(), says) {
			t.Errorf("%s\ngave %v, not a refusal naming %s", raw, err, says)
		}
	}
	if _, err := ParseScript([]byte(script), nil); err == nil {
		t.Error("a script with no placer was read")
	}
	if _, err := ReadScript("testdata/none.yaml", score); err == nil {
		t.Error("a missing script was read")
	}
}

// the chant's line is up from its start to its end, whole.
func TestTheChantIsUpWhileItIsSung(t *testing.T) {
	read := gridScript(t)
	for _, seconds := range []float64{3.0, 4.0, 4.69} {
		if words, ok := read.Chant(seconds); !ok || words != "one two three" {
			t.Errorf("at %v the chant is %q (%v)", seconds, words, ok)
		}
	}
}

// the anti-test: before its start and from its end, nothing is up.
func TestTheChantIsDownOutsideItsLine(t *testing.T) {
	read := gridScript(t)
	for _, seconds := range []float64{2.99, 4.7, 9.1} {
		if words, ok := read.Chant(seconds); ok {
			t.Errorf("at %v the chant shows %q", seconds, words)
		}
	}
}

// a sentence too long for the time it stands inside a short is named,
// with what it needs and what it has; one that fits is not.
func TestASentenceTooLongIsNamed(t *testing.T) {
	read, err := ParseScript([]byte(`shorts:
  quick: {from: {at: "1.1"}, until: {at: "1.3"}}
  slow: {from: {at: "1.1"}, until: {at: "4.4"}}
says:
  - {at: "1.1", says: "This sentence is far too long to read in one second."}
  - {at: "1.3", says: "Fine."}`), gridScore(t))
	if err != nil {
		t.Fatal(err)
	}
	slow, err := read.TooFast("quick")
	if err != nil || len(slow) != 1 ||
		!strings.Contains(slow[0].Error(), "far too long") {
		t.Errorf("too fast in quick: %v (%v)", slow, err)
	}
	if fine, _ := read.TooFast("slow"); len(fine) != 1 {
		t.Errorf("in slow the long one stands a second too: %v", fine)
	}
	if _, err := read.TooFast("none"); err == nil {
		t.Error("a short the script lacks was checked")
	}
}

// a beat is placed in its bar; one outside it, or a bar of no beats, is
// refused by name.
func TestABeatIsPlacedInItsBar(t *testing.T) {
	beat, err := NewBeat(25, 2, 4, 0.1, true)
	if err != nil || !beat.On() || beat.Bar() != 25 || beat.InBar() != 2 ||
		beat.PerBar() != 4 || beat.Since() != 0.1 || !beat.Measured() {
		t.Errorf("bar 25 beat 2: %+v (%v)", beat, err)
	}
	off, err := OffGrid(3)
	if err != nil || off.On() || off.PerBar() != 3 || off.Bar() != 0 {
		t.Errorf("off the grid: %+v (%v)", off, err)
	}
	for _, bad := range [][3]int{{0, 1, 4}, {1, 0, 4}, {1, 5, 4},
		{1, 1, 0}, {1, 1, mostBeats + 1}} {
		if _, err := NewBeat(bad[0], bad[1], bad[2], 0, true); err == nil {
			t.Errorf("bar %d beat %d of %d was taken", bad[0], bad[1], bad[2])
		}
	}
	if _, err := NewBeat(1, 1, 4, -1, true); err == nil {
		t.Error("a beat struck in the future was taken")
	}
	if _, err := OffGrid(0); err == nil {
		t.Error("a bar of no beats was taken")
	}
}

// steady is a reading that is always one level.
func steady(level float64) Reading {
	return func(float64) (float64, error) { return level, nil }
}

// broken is a reading that always fails.
func broken(float64) (float64, error) { return 0, errors.New("broken") }

// explainer is an explainer over grid and script with two lights, a
// clock on grid and a red-leaning ground.
func explainer(t *testing.T, level Reading) *Explainer {
	t.Helper()
	score := gridScore(t)
	green := color.RGBA{154, 217, 138, 255}
	one, err := NewLight("One", green, level, nil)
	if err != nil {
		t.Fatal(err)
	}
	two, err := NewLight("Two", green, steady(0.5), steady(1))
	if err != nil {
		t.Fatal(err)
	}
	clock := func(seconds float64) (Beat, error) {
		beat := score.BeatAt(seconds)
		if !beat.On() {
			return OffGrid(4)
		}
		return NewBeat(beat.Bar(), beat.InBar(), 4, beat.Since(),
			beat.Measured())
	}
	ground := func(float64) (color.RGBA, error) {
		return Tinted([3]float64{1, 0.04, 0.04}, 1)
	}
	made, err := New(Parts{Clock: clock, Lights: []Light{one, two},
		Ground: ground, Script: gridScript(t)})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// the panel at a moment holds the show's beat, its lights as their
// readings say, its ground, the chant and the sentence.
func TestThePanelHoldsWhatTheShowSays(t *testing.T) {
	made := explainer(t, steady(2))
	panel, err := made.PanelAt(3.3)
	if err != nil {
		t.Fatal(err)
	}
	lamps := panel.Lamps()
	if len(lamps) != 2 || lamps[0].Label() != "One" ||
		lamps[0].Level() != 1 || lamps[0].Ring() != 0 ||
		lamps[1].Level() != 0.5 || lamps[1].Ring() != 1 {
		t.Errorf("lamps: %+v", lamps)
	}
	if words, singing := panel.Chant(); !singing || words != "one two three" {
		t.Errorf("at 3.3 s the chant is %q", words)
	}
	if said, saying := panel.Says(); !saying || said.Words() != "First." {
		t.Errorf("the sentence is %q", said.Words())
	}
	if beat := panel.Beat(); beat.InBar() != 3 {
		t.Errorf("at 3.3 s the beat is %d", beat.InBar())
	}
	if Legible(panel.Ground()) != nil {
		t.Errorf("the ground %v is not legible", panel.Ground())
	}
	if made.Script() == nil {
		t.Error("the explainer lost its script")
	}
}

// the anti-test: an explainer needs a clock, a script and one to six
// lights, a light needs a label and a level, and a reading that fails
// fails the panel.
func TestAnExplainerNeedsItsParts(t *testing.T) {
	light, _ := NewLight("One", color.RGBA{}, steady(1), nil)
	clock := func(float64) (Beat, error) { return OffGrid(4) }
	read := gridScript(t)
	for name, parts := range map[string]Parts{
		"no clock":  {Lights: []Light{light}, Script: read},
		"no script": {Clock: clock, Lights: []Light{light}},
		"no lights": {Clock: clock, Script: read},
		"seven lights": {Clock: clock, Script: read,
			Lights: []Light{light, light, light, light, light, light, light}},
	} {
		if _, err := New(parts); err == nil {
			t.Errorf("%s: an explainer was made", name)
		}
	}
	if _, err := NewLight("", color.RGBA{}, steady(1), nil); err == nil {
		t.Error("a light with no label was made")
	}
	if _, err := NewLight("One", color.RGBA{}, nil, nil); err == nil {
		t.Error("a light with no level was made")
	}
	if _, err := explainer(t, broken).PanelAt(3); err == nil {
		t.Error("a failing reading made a panel")
	}
	plain, err := New(Parts{Clock: clock, Lights: []Light{light},
		Script: read})
	if err != nil {
		t.Fatal(err)
	}
	panel, _ := plain.PanelAt(3)
	if theme, _ := ThemeColour("ground"); panel.Ground() != theme {
		t.Errorf("with no ground given the panel's is %v", panel.Ground())
	}
}

// a ground tinted toward any colour keeps the ink in the band, apart in
// hue or greyed; none share leaves the theme's ground.
func TestATintedGroundIsLegible(t *testing.T) {
	for _, tint := range [][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		{1, 1, 1}, {1, 0.55, 0.08}, {1, 0.25, 0.85}, {0, 0, 0}} {
		for _, share := range []float64{0, 0.3, 0.6, 1} {
			ground, err := Tinted(tint, share)
			if err != nil {
				t.Fatal(err)
			}
			if err := Legible(ground); err != nil {
				t.Errorf("%v at %v: %v", tint, share, err)
			}
		}
	}
	theme, _ := ThemeColour("ground")
	if none, _ := Tinted([3]float64{1, 0, 0}, 0); Contrast(none, theme) > 1.02 {
		t.Errorf("no tint gives %v, not the theme's %v", none, theme)
	}
}

// the anti-test: a ground too light, too dark or too near the ink's hue
// while saturated is named for the rule it breaks.
func TestAnIllegibleGroundIsNamed(t *testing.T) {
	for ground, says := range map[color.RGBA]string{
		{128, 128, 128, 255}: "band",
		{0, 0, 0, 255}:       "band",
		{70, 40, 0, 255}:     "degrees",
	} {
		if err := Legible(ground); err == nil ||
			!strings.Contains(err.Error(), says) {
			t.Errorf("%v gave %v, not %q", ground, err, says)
		}
	}
	if _, err := ThemeColour("mauve"); err == nil {
		t.Error("a theme role that is not one was given")
	}
}

// a composed frame is 1080 by 1920, the film whole at its top and the
// panel's ground under it.
func TestAComposedFrameHoldsTheFilmWhole(t *testing.T) {
	made := explainer(t, steady(1))
	film := image.NewRGBA(image.Rect(0, 0, 1080, 793))
	red := color.RGBA{200, 10, 10, 255}
	for i := range film.Pix {
		film.Pix[i] = []uint8{red.R, red.G, red.B, red.A}[i%4]
	}
	for _, seconds := range []float64{1, 3.3, 4.3} {
		frame, err := made.Compose(seconds, film)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Bounds().Size() != frameSize {
			t.Fatalf("the frame is %v", frame.Bounds().Size())
		}
		if frame.RGBAAt(0, 0) != red || frame.RGBAAt(1079, 792) != red {
			t.Errorf("the film's corners are %v and %v", frame.RGBAAt(0, 0),
				frame.RGBAAt(1079, 792))
		}
		panel, _ := made.PanelAt(seconds)
		if frame.RGBAAt(5, 800) != panel.Ground() {
			t.Errorf("under the film is %v", frame.RGBAAt(5, 800))
		}
	}
}

// the anti-test: a film's picture of the wrong size is refused by name,
// and so is a sentence too long for the panel's rows.
func TestComposeRefusesAWrongPicture(t *testing.T) {
	made := explainer(t, steady(1))
	for _, size := range []image.Point{{720, 528}, {1080, 1200}} {
		_, err := made.Compose(3, image.NewRGBA(image.Rectangle{Max: size}))
		if err == nil || !strings.Contains(err.Error(), "1080") {
			t.Errorf("a %v picture gave %v", size, err)
		}
	}
	if _, err := made.Compose(3, nil); err == nil {
		t.Error("no picture was composed")
	}
	long := strings.Repeat("a long sentence that runs on ", 12)
	panel := Panel{sentence: Sentence{words: long}, saying: true}
	tones, _ := colours()
	frame := image.NewRGBA(image.Rectangle{Max: frameSize})
	if err := drawSentence(frame, panel, tones); err == nil ||
		!strings.Contains(err.Error(), "shorten") {
		t.Errorf("a sentence of %d letters gave %v", len(long), err)
	}
}
