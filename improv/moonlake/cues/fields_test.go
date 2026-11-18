package cues

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestEveryCueFieldShowsInTheFrame sets each field of the sheet in one
// cue and checks the frame at beat 2 of a half-second grid against
// values worked by hand: beat 2 is the place 2.0, one second in.
func TestEveryCueFieldShowsInTheFrame(t *testing.T) {
	show := parsed(t, `cues:
  - at_sec: 0
    beams: 3
    palette: [cyan, amber]
    sweep_bars: 2
    strobes: grid8
    duty: 0.5
    pulse: "x.x."
    decay: 2
    split: true
    floods: ones
    spots: 0.5
    spot_bars: 4
    search: 0.75
    search_bars: 8
    search_beat: 0.5
    search_moves: [chase, fan]
    search_move_bars: 1
    figures: [scan, cone]
    figure_bars: 1
    banks: [offset]
    banks_bars: 1
    master: 0.5
    overhead: 0.3
    exposure: -0.2
    moon: 0.4
    moon_alt: 20
    moon_pulse: 0
    mist: 0.3
`)
	got := frameAt(t, show, 1, 24)
	closeTo := func(name string, value, want float64) {
		t.Helper()
		if math.Abs(value-want) > 1e-12 {
			t.Errorf("%s = %v, want %v", name, value, want)
		}
	}
	if got.Beams != 3 {
		t.Errorf("beams = %d", got.Beams)
	}
	// the place 2 is beat index 2, so the palette's first colour is on
	if got.Colour != colours()["cyan"] {
		t.Errorf("colour = %v, want cyan", got.Colour)
	}
	closeTo("sweep", got.Sweep, 2.0/4/2)
	// a sixteenth-pair of 4 at duty 0.5 begins on an even step: white
	if got.White != 1 || got.Magenta != 0 {
		t.Errorf("strobes white %d magenta %d", got.White, got.Magenta)
	}
	closeTo("gate", got.Gate, 1)
	if got.Parity != 0 {
		t.Errorf("parity = %d, want 0 for hit 4", got.Parity)
	}
	// the right heads play a sixteenth behind: step 6 struck a sixteenth ago
	closeTo("gate right", got.GateRight, math.Exp(-1.0/2))
	closeTo("flood left", got.FloodLeft, math.Exp(-1/floodFlash))
	if got.FloodRight != 0 {
		t.Errorf("flood right = %v", got.FloodRight)
	}
	closeTo("spots", got.Spots, 0.5)
	// 2/4/4 = 0.125 of the way through the spots' bars
	closeTo("spot aim", got.SpotAim, 2*(1-math.Abs(2*0.125-1)))
	closeTo("search", got.Search, 0.75)
	closeTo("search phase", got.SearchPhase, 2.0/4/8)
	if got.SearchMove != "chase" {
		t.Errorf("search move = %q", got.SearchMove)
	}
	closeTo("search move phase", got.SearchMovePhase, 0.5)
	closeTo("search gate", got.SearchGate, 1)
	if got.Figure != "scan" || got.Banks != "offset" {
		t.Errorf("figure %q banks %q", got.Figure, got.Banks)
	}
	closeTo("figure phase", got.FigurePhase, 0.5)
	closeTo("master", got.Master, 0.5)
	closeTo("overhead", got.Overhead, 0.3)
	closeTo("exposure", got.Exposure, -0.2)
	closeTo("moon", got.Moon, 0.4)
	closeTo("moon alt", got.MoonAlt, 20)
	closeTo("mist", got.Mist, 0.3)
	if got.Bar != "1.3" {
		t.Errorf("bar = %q", got.Bar)
	}
}

// TestSearchGateFallsBetweenBeats checks that with search_beat set the
// searchlights' gate is struck on a beat and has fallen half a beat on.
func TestSearchGateFallsBetweenBeats(t *testing.T) {
	show := parsed(t, `cues: [{at_sec: 0, search_beat: 0.5}]`)
	// half a beat past beat 2, at 1.25 s
	got := frameAt(t, show, 1.25, 24).SearchGate
	want := 1 - 0.5 + 0.5*math.Exp(-0.5/0.35)
	if math.Abs(got-want) > 1e-12 {
		t.Errorf("search gate = %v, want %v", got, want)
	}
}

// TestLeadStrobesFollowHotAndHotter checks the lead's loudness cuts: a
// note of 90 is silent above both cuts, white between them and magenta
// at or above the upper one.
func TestLeadStrobesFollowHotAndHotter(t *testing.T) {
	for _, each := range []struct {
		sheet          string
		white, magenta int
	}{
		{`cues: [{at_sec: 0, strobes: lead, hot: 95, hotter: 99}]`, 0, 0},
		{`cues: [{at_sec: 0, strobes: lead, hot: 85, hotter: 95}]`, 1, 0},
		{`cues: [{at_sec: 0, strobes: lead, hot: 70, hotter: 85}]`, 0, 1},
		{`cues: [{at_sec: 0, strobes: lattice}]`, -1, -1},
	} {
		got := frameAt(t, parsed(t, each.sheet), 1.125, 8)
		if got.White != each.white || got.Magenta != each.magenta {
			t.Errorf("%s: white %d magenta %d, want %d %d", each.sheet,
				got.White, got.Magenta, each.white, each.magenta)
		}
	}
}

// TestFloodsLeadSideFollowsTheNoteCount checks that the lead mode fires
// the sides in turn by the note's place in the lead, and only for notes
// that are loud enough and under half a second old.
func TestFloodsLeadSideFollowsTheNoteCount(t *testing.T) {
	lead := []note{{AtSec: 1, Vel: 90}, {AtSec: 1.2, Vel: 100}, {AtSec: 1.3, Vel: 50},
		{AtSec: 0.4, Vel: 100}}
	left, right := flooded("lead", 1.25, 0, nil, lead, 80)
	if want := math.Exp(-0.25 / floodFlash); math.Abs(left-want) > 1e-12 {
		t.Errorf("left = %v, want %v", left, want)
	}
	if want := math.Exp(-0.05 / floodFlash); math.Abs(right-want) > 1e-12 {
		t.Errorf("right = %v, want %v", right, want)
	}
	// the quiet note at 1.3 has not struck yet, and the old one is past
	if left, right := flooded("lead", 1.1, 0, nil, lead, 95); right != 0 || left != 0 {
		t.Errorf("under the cut: %v and %v", left, right)
	}
}

// TestGridFloodsAreDarkBeforeTheGrid checks a place before the grid
// fires no flood, in each grid mode.
func TestGridFloodsAreDarkBeforeTheGrid(t *testing.T) {
	for _, mode := range []string{"grid16", "grid8", "ones"} {
		if left, right := flooded(mode, 0, -1, steady(4), nil, 0); left != 0 || right != 0 {
			t.Errorf("%s before the grid: %v and %v", mode, left, right)
		}
	}
}

// TestLoadReadsBothFiles checks Load against files written to a
// temporary directory, and refuses a file that is not yaml.
func TestLoadReadsBothFiles(t *testing.T) {
	dir := t.TempDir()
	scorePath := filepath.Join(dir, "score.yaml")
	cuesPath := filepath.Join(dir, "cues.yaml")
	if err := os.WriteFile(scorePath, []byte(reuseScore), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cuesPath, []byte(`cues: [{at_sec: 1, rig: 0.9}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	show, err := Load(scorePath, cuesPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := frameAt(t, show, 0.5, 24).Rig; got != 0 {
		t.Errorf("before the cue rig = %v", got)
	}
	if got := frameAt(t, show, 1, 24).Rig; got != 0.9 {
		t.Errorf("at the cue rig = %v", got)
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("cues: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(scorePath, bad); err == nil {
		t.Error("Load accepted malformed cues")
	}
	if _, err := Load(bad, cuesPath); err == nil {
		t.Error("Load accepted a malformed score")
	}
}

// TestGridStrobesAlternateWhiteAndMagenta checks that the odd
// subdivisions of a grid strobe burn magenta and the even ones white.
func TestGridStrobesAlternateWhiteAndMagenta(t *testing.T) {
	for _, each := range []struct {
		mode           string
		seconds        float64
		white, magenta int
	}{
		{"grid8", 0, 1, 0},
		{"grid8", 0.25, 0, 1}, // place 0.5, subdivision 1
		{"grid16", 0.125, 0, 1},
		{"grid16", 0.25, 1, 0}, // place 0.5, subdivision 2
	} {
		show := parsed(t, `cues: [{at_sec: 0, strobes: `+each.mode+`, duty: 0.5}]`)
		got := frameAt(t, show, each.seconds, 24)
		if got.White != each.white || got.Magenta != each.magenta {
			t.Errorf("%s at %v: white %d magenta %d, want %d %d", each.mode,
				each.seconds, got.White, got.Magenta, each.white, each.magenta)
		}
	}
}
