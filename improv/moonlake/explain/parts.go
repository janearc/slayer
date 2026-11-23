package explain

import (
	"fmt"
	"image/color"
)

// Beat is where a moment falls in a show's bars: which bar, which beat
// of it, how many beats a bar holds, how long since the beat struck, and
// whether it was heard or only counted on through a break.
type Beat struct {
	on, measured       bool
	bar, inBar, perBar int
	since              float64
}

// NewBeat is a moment on a show's grid: bar from one, inBar from one to
// perBar, since from nought. Anything else is refused by name.
func NewBeat(bar, inBar, perBar int, since float64, measured bool) (Beat,
	error) {
	if bar < 1 || perBar < 1 || perBar > mostBeats || inBar < 1 ||
		inBar > perBar || !(since >= 0) {
		return Beat{}, fmt.Errorf("%w: bar %d beat %d of %d, %v s since; "+
			"bars and beats count from one, a bar holds 1 to %d beats, "+
			"and since is nought or more", errExplain, bar, inBar, perBar,
			since, mostBeats)
	}
	return Beat{on: true, measured: measured, bar: bar, inBar: inBar,
		perBar: perBar, since: since}, nil
}

// OffGrid is a moment before a show's grid starts, in a show of perBar
// beats a bar.
func OffGrid(perBar int) (Beat, error) {
	if perBar < 1 || perBar > mostBeats {
		return Beat{}, fmt.Errorf("%w: a bar of %d beats; a bar holds 1 "+
			"to %d", errExplain, perBar, mostBeats)
	}
	return Beat{perBar: perBar}, nil
}

// On is whether the moment is on the grid.
func (of Beat) On() bool { return of.on }

// Bar is the bar, from one; nought off the grid.
func (of Beat) Bar() int { return of.bar }

// InBar is the beat's place in its bar, from one; nought off the grid.
func (of Beat) InBar() int { return of.inBar }

// PerBar is how many beats the show's bars hold.
func (of Beat) PerBar() int { return of.perBar }

// Since is how long ago the beat struck, in seconds.
func (of Beat) Since() float64 { return of.since }

// Measured is whether the beat was heard, not counted on.
func (of Beat) Measured() bool { return of.measured }

// Reading is a level a show reads at a moment, 0 to 1.
type Reading func(seconds float64) (float64, error)

// Light is one lamp on the panel: its name, its colour lit, how bright
// it is at a moment, and how bright its ring, which a show flashes when
// a cue that sets the light fires.
type Light struct {
	label       string
	colour      color.RGBA
	level, ring Reading
}

// NewLight is a lamp. A label is needed; a light with no ring reading
// never rings.
func NewLight(label string, colour color.RGBA, level, ring Reading) (
	Light, error) {
	if label == "" || level == nil {
		return Light{}, fmt.Errorf("%w: a light needs a label and a level",
			errExplain)
	}
	if ring == nil {
		ring = func(float64) (float64, error) { return 0, nil }
	}
	return Light{label: label, colour: colour, level: level, ring: ring},
		nil
}

// Parts is what a show supplies for its explainer: the clock, which bar
// and beat a moment is; the lights its lamps show, one to six; the
// ground behind the panel at a moment, or nil for the hud theme's own;
// and the script.
type Parts struct {
	Clock  func(seconds float64) (Beat, error)
	Lights []Light
	Ground func(seconds float64) (color.RGBA, error)
	Script *Script
}
