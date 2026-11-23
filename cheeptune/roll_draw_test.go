//go:build draw

package cheeptune

import (
	"image"
	"math"
	"os"
	"testing"
)

func TestTheRollIsDrawnFromWhatPlayed(t *testing.T) {
	s, err := ReadScore("@dusk")
	if err != nil {
		t.Fatal(err)
	}
	v, err := ReadVoicing("@dmg-echo")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Render(s, v)
	if err != nil {
		t.Fatal(err)
	}
	img, err := p.roll()
	if err != nil {
		t.Fatal(err)
	}
	secs := float64(len(p.Left)) / float64(p.Rate)
	if w := img.Bounds().Dx(); w < rollLeft+int(math.Ceil(secs*60))+rollRight {
		t.Errorf("width %d is short of %.2f s at sixty pixels a second",
			w, secs)
	}
	if os.Getenv("TUNE_KEEP") != "" {
		if err := p.WriteRoll("out/dusk-dmg-echo.png"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheRollPutsAnEchoWhereItSounded(t *testing.T) {
	// one note, and its echo three quarters of a beat later. the marks
	// the roll draws come from the ticks, so the echo's first mark has to
	// be that late, give or take the tick the driver started it on.
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["c5:1/2 -:7/2"]}`),
		voicing(lead+`, {"kind": "pulse", "echo": "lead",
			"delay": "3/4", "gain": 0.5, "duty": 0.5}`, ""))
	first := func(i int) float64 {
		for _, tk := range p.Ticks[i] {
			if tk.Vol > 0 {
				return tk.At
			}
		}
		return -1
	}
	want := 0.75 * p.Score.beatSeconds()
	if got := first(1) - first(0); math.Abs(got-want) > 1/tickRate {
		t.Errorf("the echo's first mark is %.3f s after the lead's, "+
			"want %.3f", got, want)
	}
	if _, err := p.roll(); err != nil {
		t.Fatal(err)
	}
}

func TestAShortPieceKeepsItsWholeTitle(t *testing.T) {
	// two seconds is 120 pixels of notes, far narrower than the title.
	// nothing may be drawn past the right edge, so the picture widens.
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["c4:4"]}`),
		voicing(lead, ""))
	img, err := p.roll()
	if err != nil {
		t.Fatal(err)
	}
	// the title sits in the top line; the right margin there must be
	// ground, not ink cut off mid-letter.
	if x, y, ok := inkInMargin(img, 0, rollLine); ok {
		t.Errorf("ink at %d,%d in the right margin: the title is cut", x, y)
	}
}

func TestAShortPieceKeepsItsWholeLegend(t *testing.T) {
	// four channels name themselves in the legend, which for a two-second
	// piece is wider than the notes and wider than the title.
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["c4:4"]},
			{"role": "bass", "bars": ["c3:4"]},
			{"role": "perc", "bars": ["k:4"]}`),
		voicing(lead+`,
			{"kind": "pulse", "echo": "lead", "delay": "3/4",
			 "gain": 0.5, "duty": 0.5},
			{"kind": "wave", "role": "bass", "table": "triangle"},
			{"kind": "noise", "role": "perc",
			 "hits": {"k": {"shift": 7, "divisor": 3, "decay": 0.12}}}`,
			""))
	img, err := p.roll()
	if err != nil {
		t.Fatal(err)
	}
	if x, y, ok := inkInMargin(img, rollLine, 2*rollLine); ok {
		t.Errorf("ink at %d,%d in the right margin: the legend is cut",
			x, y)
	}
}

// inkInMargin finds the first pixel in the right margin, between rows y0 and
// y1, that is not the ground.
func inkInMargin(img *image.RGBA, y0, y1 int) (x, y int, found bool) {
	w := img.Bounds().Dx()
	for y := y0; y < y1; y++ {
		for x := w - rollRight; x < w; x++ {
			if img.RGBAAt(x, y) != rollGround {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}
