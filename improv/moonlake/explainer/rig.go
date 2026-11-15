package main

import (
	"image/color"
	"math"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/show"
)

// The numbers moonlake's parts work in. Picked by the explainer's
// author, fable-team, 2026-11-12, unless a line says otherwise.
const (
	// cueFlashFall is how long a lamp's ring takes to fall to a third
	// after its cue fires, in seconds
	cueFlashFall = 0.35
	// cueFlashSpan is how long after its cue a lamp's ring is drawn, in
	// seconds: past three falls it is under a twentieth
	cueFlashSpan = 3 * cueFlashFall
	// strobeHold is how long a strobe lamp's flash takes to fall to a
	// third, in seconds: a strobe lasts one frame, which a viewer misses
	strobeHold = 0.1
	// strobeLookBack is how many frames back a strobe's flash is looked
	// for: four falls of strobeHold at 24 a second
	strobeLookBack = 10
	// groundShare is how far the panel's ground leans toward the beams'
	// colour with the rig full; the ground is held legible whatever it is
	groundShare = 0.6
	// rigFull is the rig level at which the ground leans its whole share:
	// the first drums' 0.15 leans a quarter of it
	rigFull = 0.6
	// beatsPerBar is moonlake's bar. Defined by the song.
	beatsPerBar = 4
)

// moonlake is the show's parts for package explain: its score and cue
// sheet, read.
type moonlake struct {
	score *show.Score
	sheet *show.Sheet
	fired []show.Fired
}

// clock is which bar and beat a moment is, from the score's grid.
func (of moonlake) clock(seconds float64) (explain.Beat, error) {
	beat := of.score.BeatAt(seconds)
	if !beat.On() {
		return explain.OffGrid(beatsPerBar)
	}
	return explain.NewBeat(beat.Bar(), beat.InBar(), beatsPerBar,
		beat.Since(), beat.Measured())
}

// lamp is one of moonlake's lamps: its label, the theme role it lights
// in, the rig's part whose cues ring it, and its level from a moment.
type lamp struct {
	label, role, part string
	level             func(of moonlake, seconds float64) (float64, error)
}

// reading is a lamp level read off the moment at that time.
func reading(read func(show.Moment) float64) func(moonlake, float64) (
	float64, error) {
	return func(of moonlake, seconds float64) (float64, error) {
		at, err := of.sheet.At(seconds, frameRate)
		return read(at), err
	}
}

// lamps are moonlake's six, in the order the panel shows them.
var lamps = []lamp{
	{"Lasers", "ok", show.PartLasers, reading(show.Moment.Lasers)},
	{"Strobes", "ink", show.PartStrobes, moonlake.strobeFlash},
	{"Backlights", "warn", show.PartFloods, reading(show.Moment.Floods)},
	{"Searchlights", "border", show.PartSearch,
		reading(show.Moment.Searchlights)},
	{"Moon", "ink", show.PartMoon, reading(show.Moment.Moon)},
	{"Ship's beam", "ink", show.PartOverhead, reading(show.Moment.Overhead)},
}

// lights is moonlake's lamps for the panel.
func (of moonlake) lights() ([]explain.Light, error) {
	var made []explain.Light
	for _, each := range lamps {
		colour, err := explain.ThemeColour(each.role)
		if err != nil {
			return nil, err
		}
		level := func(seconds float64) (float64, error) {
			return each.level(of, seconds)
		}
		part := each.part
		ring := func(seconds float64) (float64, error) {
			return cueFlash(of.fired, part, seconds), nil
		}
		light, err := explain.NewLight(each.label, colour, level, ring)
		if err != nil {
			return nil, err
		}
		made = append(made, light)
	}
	return made, nil
}

// strobeFlash is the strobes' lamp: a strobe fires for one frame, so the
// lamp holds each flash and lets it fall by strobeHold.
func (of moonlake) strobeFlash(seconds float64) (float64, error) {
	brightest := 0.0
	for back := range strobeLookBack {
		ago := float64(back) / frameRate
		then, err := of.sheet.At(seconds-ago, frameRate)
		if err != nil {
			return 0, err
		}
		if then.Strobe() {
			brightest = math.Max(brightest, math.Exp(-ago/strobeHold))
		}
	}
	return brightest, nil
}

// cueFlash is a lamp's ring at a moment: full when a cue that sets its
// part fires, falling by cueFlashFall, gone after cueFlashSpan, and
// nought before the cue.
func cueFlash(fired []show.Fired, part string, seconds float64) float64 {
	brightest := 0.0
	for _, each := range fired {
		since := seconds - each.At()
		if !each.Sets(part) || since < 0 || since >= cueFlashSpan {
			continue
		}
		brightest = math.Max(brightest, math.Exp(-since/cueFlashFall))
	}
	return brightest
}

// ground is the panel's ground at a moment: leaning toward the beams'
// colour this beat as far as the rig is up, held legible.
func (of moonlake) ground(seconds float64) (color.RGBA, error) {
	at, err := of.sheet.At(seconds, frameRate)
	if err != nil {
		return color.RGBA{}, err
	}
	share := groundShare * math.Min(1, at.Rig()/rigFull)
	return explain.Tinted(at.Colour(), share)
}
