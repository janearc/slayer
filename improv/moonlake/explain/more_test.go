package explain

import (
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadScriptReadsAFile checks a script written to a temporary file
// reads as the same script as its text does.
func TestReadScriptReadsAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "explain.yaml")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	read, err := ReadScript(path, gridScore(t))
	if err != nil {
		t.Fatal(err)
	}
	if from, until, ok := read.Short("all"); !ok || from != 2.0 || until != 9.5 {
		t.Errorf("the short all is %v to %v (%v)", from, until, ok)
	}
	if said, ok := read.Says(4); !ok || said.Words() != "Second." {
		t.Errorf("at 4 s the script says %q (%v)", said.Words(), ok)
	}
	if err := os.WriteFile(path, []byte("says: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadScript(path, gridScore(t)); !errors.Is(err, ErrExplain) {
		t.Errorf("a broken file gave %v", err)
	}
}

// TestPanelFailsWhereItsPartsFail checks that a failing clock, ground,
// level or ring each fails the panel and that Compose passes it on.
func TestPanelFailsWhereItsPartsFail(t *testing.T) {
	light, err := NewLight("One", color.RGBA{}, steady(1), nil)
	if err != nil {
		t.Fatal(err)
	}
	ringing, err := NewLight("Two", color.RGBA{}, steady(1), broken)
	if err != nil {
		t.Fatal(err)
	}
	good := func(float64) (Beat, error) { return OffGrid(4) }
	failing := errors.New("clock failed")
	for name, parts := range map[string]Parts{
		"clock": {Clock: func(float64) (Beat, error) { return Beat{}, failing },
			Lights: []Light{light}, Script: gridScript(t)},
		"ground": {Clock: good, Lights: []Light{light}, Script: gridScript(t),
			Ground: func(float64) (color.RGBA, error) { return color.RGBA{}, failing }},
		"ring": {Clock: good, Lights: []Light{ringing}, Script: gridScript(t)},
	} {
		made, err := New(parts)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := made.PanelAt(3); err == nil {
			t.Errorf("%s: the panel was made", name)
		}
		film := image.NewRGBA(image.Rect(0, 0, 1080, 600))
		if _, err := made.Compose(3, film); err == nil {
			t.Errorf("%s: a frame was composed", name)
		}
	}
}

// TestClampHoldsLevelsToZeroAndOne checks both ends, the inside and a
// number that is not one.
func TestClampHoldsLevelsToZeroAndOne(t *testing.T) {
	for _, each := range []struct{ level, want float64 }{
		{-3, 0}, {0, 0}, {0.25, 0.25}, {1, 1}, {7, 1}, {math.NaN(), 0},
	} {
		if got := clamp(each.level); got != each.want {
			t.Errorf("clamp(%v) = %v, want %v", each.level, got, each.want)
		}
	}
}

// TestPanelClampsAReadingThatIsNotANumber checks a lamp whose level is
// not a number is dark rather than propagating the value.
func TestPanelClampsAReadingThatIsNotANumber(t *testing.T) {
	light, err := NewLight("One", color.RGBA{}, steady(math.NaN()), steady(5))
	if err != nil {
		t.Fatal(err)
	}
	made, err := New(Parts{Clock: func(float64) (Beat, error) { return OffGrid(4) },
		Lights: []Light{light}, Script: gridScript(t)})
	if err != nil {
		t.Fatal(err)
	}
	panel, err := made.PanelAt(3)
	if err != nil {
		t.Fatal(err)
	}
	if lamp := panel.Lamps()[0]; lamp.Level() != 0 || lamp.Ring() != 1 {
		t.Errorf("level %v and ring %v, want 0 and 1", lamp.Level(), lamp.Ring())
	}
}

// TestContrastDoesNotDependOnTheOrder checks that black on white and
// white on black give the same ratio, 21 with the flare: (1+0.05)/(0+0.05).
func TestContrastDoesNotDependOnTheOrder(t *testing.T) {
	black, white := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}
	for _, got := range []float64{Contrast(black, white), Contrast(white, black)} {
		if math.Abs(got-21) > 1e-9 {
			t.Errorf("contrast = %v, want 21", got)
		}
	}
}

// TestSaturationIsTheSpreadOverTheBrightest checks black has none, a
// grey has none, and pure red and an even mix are worked by hand.
func TestSaturationIsTheSpreadOverTheBrightest(t *testing.T) {
	for _, each := range []struct {
		colour color.RGBA
		want   float64
	}{
		{color.RGBA{0, 0, 0, 255}, 0},
		{color.RGBA{128, 128, 128, 255}, 0},
		{color.RGBA{255, 0, 0, 255}, 1},
		{color.RGBA{200, 100, 50, 255}, 0.75},
	} {
		if got := saturation(each.colour); math.Abs(got-each.want) > 1e-12 {
			t.Errorf("saturation(%v) = %v, want %v", each.colour, got, each.want)
		}
	}
}

// TestComposeNamesASentenceThatDoesNotFit checks a frame whose standing
// sentence is too long for the panel is refused by Compose itself.
func TestComposeNamesASentenceThatDoesNotFit(t *testing.T) {
	long := strings.Repeat("a long sentence that runs on ", 12)
	read, err := ParseScript([]byte(`says: [{at_sec: 2, says: "`+long+`"}]`), gridScore(t))
	if err != nil {
		t.Fatal(err)
	}
	light, err := NewLight("One", color.RGBA{}, steady(1), nil)
	if err != nil {
		t.Fatal(err)
	}
	made, err := New(Parts{Clock: func(float64) (Beat, error) { return OffGrid(4) },
		Lights: []Light{light}, Script: read})
	if err != nil {
		t.Fatal(err)
	}
	film := image.NewRGBA(image.Rect(0, 0, 1080, 600))
	if _, err := made.Compose(3, film); err == nil || !strings.Contains(err.Error(), "shorten") {
		t.Errorf("Compose gave %v", err)
	}
}
