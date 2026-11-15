package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/show"
)

// near is how close two numbers must be to be the same here.
const near = 1e-9

// ours is moonlake's own score, cue sheet and script, read.
func ours(t *testing.T) (moonlake, *explain.Script) {
	t.Helper()
	score, err := show.ReadScore("../score.yaml")
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := show.ReadSheet("../cues.yaml", score)
	if err != nil {
		t.Fatal(err)
	}
	script, err := explain.ReadScript("../explain.yaml", score)
	if err != nil {
		t.Fatal(err)
	}
	return moonlake{score: score, sheet: sheet, fired: sheet.Fired()}, script
}

// small is a show of sixteen beats half a second apart from 2 s, its
// strobes cued on at bar 2 and its rig at bar 3.
func small(t *testing.T) moonlake {
	t.Helper()
	score, err := show.ParseScore([]byte(`score:
  beats:
    times: [2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0, 5.5, 6.0, 6.5, 7.0, 7.5,
            8.0, 8.5, 9.0, 9.5]
    measured: [1, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1]`))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := show.ParseSheet([]byte(`cues:
  - {at: "2.1", strobes: grid8, duty: 0.1}
  - {at: "3.1", rig: 1}`), score)
	if err != nil {
		t.Fatal(err)
	}
	return moonlake{score: score, sheet: sheet, fired: sheet.Fired()}
}

// every short in moonlake's script can be read: each sentence that comes
// up inside it stands long enough at the reading rate.
func TestEveryShortCanBeRead(t *testing.T) {
	_, script := ours(t)
	names := script.Shorts()
	if len(names) != 5 {
		t.Errorf("the script has shorts %v", names)
	}
	for _, name := range names {
		slow, err := script.TooFast(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, each := range slow {
			t.Error(each)
		}
	}
}

// the chant short holds the first two sung lines, and the drop short
// the third.
func TestTheChantShortHoldsTheFirstTwoLines(t *testing.T) {
	_, script := ours(t)
	from, until, _ := script.Short("chant")
	if until-from < 30 || until-from > 45 {
		t.Errorf("the chant short is %v s long", until-from)
	}
	for _, seconds := range []float64{1, 22} {
		if _, ok := script.Chant(seconds); !ok || seconds >= until {
			t.Errorf("at %v s no line is sung inside the chant short", seconds)
		}
	}
	from, until, _ = script.Short("drop")
	if _, ok := script.Chant(126); !ok || 126 < from || 126 >= until {
		t.Error("the reprise is not sung inside the drop short")
	}
}

// a lamp's ring is full when its cue fires, falls after, and is gone
// after its span.
func TestARingFlashesWhenItsCueFires(t *testing.T) {
	show := small(t)
	if got := cueFlash(show.fired, "strobes", 4.0); !(math.Abs(got-1) <= near) {
		t.Errorf("on its cue the strobes' ring is %v", got)
	}
	later := 4.0 + cueFlashFall
	if got := cueFlash(show.fired, "strobes", later); !(math.Abs(got-math.Exp(-1)) <= near) {
		t.Errorf("a fall after, the ring is %v", got)
	}
	if got := cueFlash(show.fired, "strobes", 4.01+cueFlashSpan); got != 0 {
		t.Errorf("after its span the ring is %v", got)
	}
}

// the anti-test: a ring is dark before its cue, and a cue for another
// part leaves it alone.
func TestARingIsDarkBeforeItsCue(t *testing.T) {
	show := small(t)
	if got := cueFlash(show.fired, "strobes", 3.99); got != 0 {
		t.Errorf("before its cue the strobes' ring is %v", got)
	}
	if got := cueFlash(show.fired, "strobes", 6.0); got != 0 {
		t.Errorf("the lasers' cue lit the strobes' ring to %v", got)
	}
	if got := cueFlash(show.fired, "lasers", 6.0); got != 1 {
		t.Errorf("the lasers' cue lit their ring to %v", got)
	}
}

// the strobes' lamp holds a strobe's one-frame flash a few frames, and
// is dark where no strobe fired.
func TestTheStrobeLampHoldsItsFlash(t *testing.T) {
	show := small(t)
	// grid8 duty 0.1: a strobe burns 4.25 s to 4.275 s
	if level, _ := show.strobeFlash(4.3); !(level > 0.3) {
		t.Errorf("two frames after a strobe its lamp is %v", level)
	}
	if level, _ := show.strobeFlash(3.9); level != 0 {
		t.Errorf("before the strobes were cued their lamp is %v", level)
	}
}

// the clock is the score's: bar and beat, carried beats marked, and off
// the grid before it.
func TestTheClockIsTheScores(t *testing.T) {
	show := small(t)
	beat, err := show.clock(4.6)
	if err != nil || beat.Bar() != 2 || beat.InBar() != 2 ||
		beat.Measured() || beat.PerBar() != 4 {
		t.Errorf("at 4.6 s: %+v (%v)", beat, err)
	}
	if off, _ := show.clock(1); off.On() {
		t.Error("before the grid the clock is on it")
	}
}

// the panel's ground stays legible under every palette colour the cue
// sheet uses, at every share the rig can give it: tested by arithmetic.
func TestTheGroundIsLegibleForEveryPalette(t *testing.T) {
	show, _ := ours(t)
	names := show.sheet.Palettes()
	if len(names) < 4 {
		t.Fatalf("the sheet's palettes are %v", names)
	}
	for _, name := range names {
		colour, ok := showColour(name)
		if !ok {
			t.Fatalf("lightshow does not know %q", name)
		}
		for _, share := range []float64{0, 0.25, 0.5, groundShare} {
			ground, err := explain.Tinted(colour, share)
			if err != nil {
				t.Fatal(err)
			}
			if err := explain.Legible(ground); err != nil {
				t.Errorf("%s at %v: %v", name, share, err)
			}
		}
	}
}

// showColour is a palette name's colour.
func showColour(name string) ([3]float64, bool) { return show.PaletteColour(name) }

// the anti-test: with the rig dark the ground is the theme's own, and
// with the rig up it is not.
func TestTheGroundLeansOnlyWithTheRig(t *testing.T) {
	show, _ := ours(t)
	theme, err := explain.ThemeColour("ground")
	if err != nil {
		t.Fatal(err)
	}
	dark, _ := show.ground(30)
	if explain.Contrast(dark, theme) > 1.02 {
		t.Errorf("before the drums the ground is %v, not the theme's %v",
			dark, theme)
	}
	if lit, _ := show.ground(110); lit == theme {
		t.Errorf("with the rig up the ground is still the theme's %v", lit)
	}
	lights, err := show.lights()
	if err != nil || len(lights) != 6 {
		t.Errorf("moonlake has %d lights (%v)", len(lights), err)
	}
}
