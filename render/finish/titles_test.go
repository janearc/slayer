package finish

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/hud"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/finish/mocks"
	"go.uber.org/mock/gomock"
)

// the picture the titles are drawn on: a portrait small enough to be
// quick, tall enough for twenty rows of cells ten pixels high.
const (
	pictureWide = 200
	pictureHigh = 400
	pictureRows = 40
)

// words is a job's titles: a text from one second until three, a bar
// that fills across four seconds, and a frame always.
const words = `titles:
  rows: 40
  elements:
    edge: {kind: frame, ink: border, left: 0, top: 0, right: 1, bottom: 1}
    hello:
      kind: text
      words: hello
      ink: ink
      ground: ground
      across: 0.1
      down: 0.5
      from: {value: 1, unit: s}
      until: {value: 3, unit: s}
    meter:
      kind: bar
      value: {from: 0, to: 1, start: 0, over: 4}
      fill: ok
      empty: dim
      left: 0.1
      top: 0.9
      right: 0.9
      bottom: 0.95
`

// grey is a picture of one grey, so what is drawn on it is seen.
func grey() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, pictureWide, pictureHigh))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 90, G: 90,
		B: 90, A: 255}), image.Point{}, draw.Src)
	return img
}

// titledAt is the grey picture with a job's titles drawn at a time.
func titledAt(t *testing.T, text string, at time.Duration) *image.RGBA {
	t.Helper()
	made, said, err := NewTitles(jobOf(t, text))
	if err != nil || !said {
		t.Fatalf("titles: said %v, error %v", said, err)
	}
	img := grey()
	if err := made.Draw(img, at); err != nil {
		t.Fatal(err)
	}
	return img
}

// differs is how many bytes of two pictures differ in a rectangle.
func differs(a, b *image.RGBA, in image.Rectangle) int {
	count := 0
	for y := in.Min.Y; y < in.Max.Y; y++ {
		for x := in.Min.X; x < in.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				count++
			}
		}
	}
	return count
}

// a text appears inside its span and not outside it. The text's cells
// are a rectangle by the 'hello' at a tenth across, half down.
func TestATextAppearsOnlyInsideItsSpan(t *testing.T) {
	place := image.Rect(20, 200, 70, 210)
	plain := grey()
	for at, want := range map[time.Duration]bool{0: false,
		time.Second - 1: false, time.Second: true, 2 * time.Second: true,
		3*time.Second - 1: true, 3 * time.Second: false,
		time.Hour: false} {
		seen := differs(titledAt(t, words, at), plain, place) > none
		if seen != want {
			t.Errorf("at %v the text is seen: %v", at, seen)
		}
	}
}

// a bar's value follows time: more of its box is the fill role's colour
// later, and none at nought.
func TestABarsValueFollowsTime(t *testing.T) {
	fill, err := hud.DefaultTheme().Colour("ok")
	if err != nil {
		t.Fatal(err)
	}
	last := -1
	for _, second := range []time.Duration{0, 1, 2, 4} {
		img := titledAt(t, words, second*time.Second)
		count := 0
		for x := 20; x < 180; x++ {
			if img.RGBAAt(x, 370) == fill {
				count++
			}
		}
		if count <= last {
			t.Errorf("at %vs the bar holds %d filled columns, %d before",
				second, count, last)
		}
		last = count
	}
	if last != 160 {
		t.Errorf("a full bar holds %d of its 160 columns", last)
	}
}

// the same bar, a second time through, is the same picture: drawing is
// a function of the time.
func TestTitlesAtOneTimeAreOnePicture(t *testing.T) {
	first := titledAt(t, words, 2*time.Second)
	second := titledAt(t, words, 2*time.Second)
	if string(first.Pix) != string(second.Pix) {
		t.Error("one time gave two pictures")
	}
}

// with no theme named the default theme is used: ink on the ground is
// the default's, to the byte.
func TestWithNoThemeTheDefaultIsUsed(t *testing.T) {
	img := titledAt(t, words, 2*time.Second)
	border, err := hud.DefaultTheme().Colour("border")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for x := range pictureWide {
		found = found || img.RGBAAt(x, 5) == border
	}
	if !found {
		t.Errorf("no pixel of the frame's top line is the default "+
			"border %v", border)
	}
}

// a theme the job names is read, and its colours are drawn.
func TestANamedThemeIsRead(t *testing.T) {
	directory := t.TempDir()
	sheet := ":root { --ink: #ff0000; --ground: #00ff00; }\n"
	if err := os.WriteFile(filepath.Join(directory, "red.css"),
		[]byte(sheet), private); err != nil {
		t.Fatal(err)
	}
	made := jobIn(t, directory, heading+`titles:
  theme: red.css
  rows: 40
  elements:
    word: {kind: text, words: x, ink: ink, ground: ground, across: 0.5,
      down: 0.5}
`)
	titles, _, err := NewTitles(made)
	if err != nil {
		t.Fatal(err)
	}
	img := grey()
	if err := titles.Draw(img, 0); err != nil {
		t.Fatal(err)
	}
	if got := img.RGBAAt(100, 200); got != (color.RGBA{G: 255, A: 255}) &&
		got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("a cell of the text is %v, not of the named theme", got)
	}
}

// the anti-test: a theme file that cannot be read is refused by its
// path, and a job with no titles says it has none.
func TestAThemeThatCannotBeReadAndNoTitles(t *testing.T) {
	directory := t.TempDir()
	made := jobIn(t, directory, heading+`titles:
  theme: missing.css
  elements:
    word: {kind: text, words: x, ink: ink, across: 0, down: 0}
`)
	_, _, err := NewTitles(made)
	if err == nil || !strings.Contains(err.Error(), "missing.css") {
		t.Errorf("a missing theme gave %v", err)
	}
	_, said, err := NewTitles(jobOf(t, ""))
	if said || err != nil {
		t.Errorf("a job with no titles gave said %v, %v", said, err)
	}
}

// a role the theme has not is refused by the title's name and the
// role's, when the titles are made.
func TestAnUnknownRoleIsRefusedByName(t *testing.T) {
	_, _, err := NewTitles(jobOf(t, `titles:
  elements:
    credit: {kind: text, words: x, ink: chartreuse, across: 0, down: 0}
`))
	if err == nil {
		t.Fatal("a role the theme has not was taken")
	}
	for _, word := range []string{"credit", "chartreuse"} {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("%q does not say %q", err, word)
		}
	}
	_, _, err = NewTitles(jobOf(t, `titles:
  elements:
    credit: {kind: text, words: x, ink: ink, ground: nonesuch,
      across: 0, down: 0}
`))
	if err == nil || !strings.Contains(err.Error(), "nonesuch") {
		t.Errorf("an unknown ground gave %v", err)
	}
}

// a title with no ink, or a bar with no fill, is refused by name. Job
// would refuse these first, so the titles are tried through a mock.
func TestATitleWithNoRoleIsRefused(t *testing.T) {
	ctrl := gomock.NewController(t)
	for _, kind := range []string{"text", "frame", "bar"} {
		title := mocks.NewTitled(ctrl)
		title.EXPECT().Name().Return("credit").AnyTimes()
		title.EXPECT().Kind().Return(kind).AnyTimes()
		title.EXPECT().Roles().Return("", "ground", "", "dim").AnyTimes()
		_, err := elementsOf([]titled{title}, hud.DefaultTheme())
		if err == nil || !strings.Contains(err.Error(), `"credit"`) {
			t.Errorf("a %s with no role gave %v", kind, err)
		}
	}
}

// a count is written out by its format at the time.
func TestACountIsWrittenOutAtTheTime(t *testing.T) {
	text := `titles:
  rows: 40
  elements:
    clock:
      kind: text
      count: {from: 0, to: 9, start: 0, over: 9}
      format: "%.0f"
      ink: ink
      ground: ground
      across: 0.1
      down: 0.5
`
	place := image.Rect(20, 200, 30, 210)
	if differs(titledAt(t, text, 0), titledAt(t, text, 9*time.Second),
		place) == none {
		t.Error("the count looked the same at nought and at nine")
	}
}

// the world's label is refused by name, with why: it needs the scene.
func TestAWorldLabelIsRefusedByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinned := mocks.NewTitled(ctrl)
	pinned.EXPECT().Name().Return("tag").AnyTimes()
	pinned.EXPECT().Kind().Return("label").AnyTimes()
	_, err := elementsOf([]titled{pinned}, hud.DefaultTheme())
	if err == nil {
		t.Fatal("a label was taken")
	}
	for _, word := range []string{`"tag"`, `"label"`, "scene"} {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("%q does not say %q", err, word)
		}
	}
}

// the anti-tests for text and bar, which job would refuse first: an
// alignment that is not one, and a bar that has no value.
func TestATextAndABarThatJobWouldNotMake(t *testing.T) {
	ctrl := gomock.NewController(t)
	text := mocks.NewTitled(ctrl)
	text.EXPECT().Name().Return("odd").AnyTimes()
	text.EXPECT().Kind().Return("text").AnyTimes()
	text.EXPECT().Roles().Return("ink", "", "", "").AnyTimes()
	text.EXPECT().At().Return(0.0, 0.0, "diagonal").AnyTimes()
	_, err := elementsOf([]titled{text}, hud.DefaultTheme())
	if err == nil || !strings.Contains(err.Error(), "diagonal") {
		t.Errorf("an odd alignment gave %v", err)
	}
	bar := mocks.NewTitled(ctrl)
	bar.EXPECT().Name().Return("empty").AnyTimes()
	bar.EXPECT().Kind().Return("bar").AnyTimes()
	bar.EXPECT().Roles().Return("", "", "ok", "").AnyTimes()
	bar.EXPECT().Value().Return(job.Ramp{}, false).AnyTimes()
	_, err = elementsOf([]titled{bar}, hud.DefaultTheme())
	if err == nil || !strings.Contains(err.Error(), "no value") {
		t.Errorf("a bar with no value gave %v", err)
	}
}

// each way a title's span is said, one end or both, is the hud's.
func TestSpansOfTitlesAreTheHuds(t *testing.T) {
	ctrl := gomock.NewController(t)
	second := time.Second
	cases := []struct {
		name          string
		from, until   *time.Duration
		at            time.Duration
		held, exactly bool
	}{
		{"neither", nil, nil, time.Hour, true, true},
		{"from only", &second, nil, 2 * second, true, true},
		{"from only, early", &second, nil, 0, false, true},
		{"until only", nil, &second, 0, true, true},
		{"until only, late", nil, &second, 2 * second, false, true},
		{"both", &second, &second, second, false, true},
	}
	for _, each := range cases {
		title := mocks.NewTitled(ctrl)
		title.EXPECT().From().DoAndReturn(func() (time.Duration, bool) {
			if each.from == nil {
				return 0, false
			}
			return *each.from, true
		}).AnyTimes()
		title.EXPECT().Until().DoAndReturn(func() (time.Duration, bool) {
			if each.until == nil {
				return 0, false
			}
			return *each.until, true
		}).AnyTimes()
		if spanOf(title).Holds(each.at) != each.held {
			t.Errorf("%s: held at %v is not %v", each.name, each.at,
				each.held)
		}
	}
}
