package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/show"
	"github.com/janearc/slayer/improv/walk"
)

// The explaining pane, drawn to fill its pane: the beat's four squares
// and the bar, the six lamps in one row, and the sentence. Laid out on a
// pane 960 across and scaled to the pane's width. Picked by the
// untraced film's author, fable-team, 2026-11-12, by looking at it full
// size and at a third of that.
const (
	panelWidth    = 960.0
	panelMargin   = 40.0
	squareSide    = 92.0
	squareGap     = 18.0
	squareTop     = 36.0
	barSize       = 58.0
	lampTop       = 196.0
	lampRadius    = 40.0
	ringWidth     = 6.0
	lampLabelSize = 26.0
	lampLabelDrop = 76.0
	sentenceTop   = 352.0
	sentenceSize  = 34.0
	sentenceLead  = 46.0
	sentenceRows  = 4
	// offShare is how much of its colour a dark square or lamp keeps:
	// the explainer's
	offShare = 0.12
	// beatFall is how long a beat's square takes to fall to a third, and
	// beatLeast how lit the square of the beat stays: the explainer's
	beatFall  = 0.18
	beatLeast = 0.3
	// cueFall is how long a lamp's ring takes to fall to a third after
	// its cue, cueSpan how long it is drawn, strobeFall a strobe lamp's
	// fall and strobeBack how many frames back a flash is looked for:
	// the explainer's (explainer/rig.go)
	cueFall    = 0.35
	cueSpan    = 3 * cueFall
	strobeFall = 0.1
	strobeBack = 10
	// beatsPerBar is the song's bar. Defined by the song.
	beatsPerBar = 4
)

// lampKinds are the six lamps as the explainer shows them: the label,
// the hud theme's role it lights in, and the rig's part whose cues ring
// it (explainer/rig.go).
var lampKinds = []struct{ label, role, part string }{
	{"Lasers", "ok", show.PartLasers},
	{"Strobes", "ink", show.PartStrobes},
	{"Backlights", "warn", show.PartFloods},
	{"Searchlights", "border", show.PartSearch},
	{"Moon", "ink", show.PartMoon},
	{"Ship's beam", "ink", show.PartOverhead},
}

// explainerOf is the explainer's panel for moonlake, from the score, the
// cue sheet and the explainer's script: the same parts the explainer
// film shows, its lamps' levels read off the cue sheet's moments.
func explainerOf(score *show.Score, sheet *show.Sheet,
	script string) (*explain.Explainer, error) {
	words, err := explain.ReadScript(script, score)
	if err != nil {
		return nil, err
	}
	var lights []explain.Light
	for i, kind := range lampKinds {
		colour, err := explain.ThemeColour(kind.role)
		if err != nil {
			return nil, err
		}
		level := levelOf(sheet, i)
		part := kind.part
		ring := func(seconds float64) (float64, error) {
			return cueRing(sheet.Fired(), part, seconds), nil
		}
		light, err := explain.NewLight(kind.label, colour, level, ring)
		if err != nil {
			return nil, err
		}
		lights = append(lights, light)
	}
	clock := func(seconds float64) (explain.Beat, error) {
		beat := score.BeatAt(seconds)
		if !beat.On() {
			return explain.OffGrid(beatsPerBar)
		}
		return explain.NewBeat(beat.Bar(), beat.InBar(), beatsPerBar,
			beat.Since(), beat.Measured())
	}
	return explain.New(explain.Parts{Clock: clock, Lights: lights,
		Script: words})
}

// levelOf is lamp i's level at a moment, as the explainer reads it; the
// strobes' held over their fall, since a strobe lasts one frame.
func levelOf(sheet *show.Sheet, lamp int) explain.Reading {
	readings := []func(show.Moment) float64{show.Moment.Lasers, nil,
		show.Moment.Floods, show.Moment.Searchlights, show.Moment.Moon,
		show.Moment.Overhead}
	if lamp == 1 {
		return func(seconds float64) (float64, error) {
			brightest := 0.0
			for back := range strobeBack {
				ago := float64(back) / filmRate
				then, err := sheet.At(seconds-ago, filmRate)
				if err != nil {
					return 0, err
				}
				if then.Strobe() {
					brightest = math.Max(brightest,
						math.Exp(-ago/strobeFall))
				}
			}
			return brightest, nil
		}
	}
	return func(seconds float64) (float64, error) {
		at, err := sheet.At(seconds, filmRate)
		return readings[lamp](at), err
	}
}

// cueRing is a lamp's ring at a moment: full when a cue that sets its
// part fires, falling, gone after cueSpan.
func cueRing(fired []show.Fired, part string, seconds float64) float64 {
	brightest := 0.0
	for _, each := range fired {
		since := seconds - each.At()
		if each.Sets(part) && since >= 0 && since < cueSpan {
			brightest = math.Max(brightest, math.Exp(-since/cueFall))
		}
	}
	return brightest
}

// drawPanel draws the explaining pane for a moment into its place.
func drawPanel(frame *image.RGBA, place image.Rectangle,
	explainer *explain.Explainer, seconds float64) error {
	panel, err := explainer.PanelAt(seconds)
	if err != nil {
		return err
	}
	tones := map[string]color.RGBA{}
	for _, role := range []string{"ink", "ground", "ok"} {
		if tones[role], err = explain.ThemeColour(role); err != nil {
			return err
		}
	}
	walk.Fill(frame, place, tones["ground"])
	scale := float64(place.Dx()) / panelWidth
	at := func(x, y float64) image.Point {
		return place.Min.Add(image.Pt(int(x*scale), int(y*scale)))
	}
	if err := drawBeat(frame, panel.Beat(), tones, at, scale); err != nil {
		return err
	}
	if err := drawLamps(frame, panel, tones, at, scale); err != nil {
		return err
	}
	sentence, saying := panel.Says()
	if !saying {
		return nil
	}
	rows, err := walk.Wrapped(sentence.Words(), sentenceSize*scale,
		int((panelWidth-2*panelMargin)*scale))
	if err != nil {
		return err
	}
	for i, row := range rows[:min(len(rows), sentenceRows)] {
		if err := walk.Words(frame, row, at(panelMargin, sentenceTop+
			sentenceSize+float64(i)*sentenceLead), sentenceSize*scale,
			tones["ink"]); err != nil {
			return err
		}
	}
	return nil
}

// drawBeat draws the four squares, the beat's lit, and the bar.
func drawBeat(frame *image.RGBA, beat explain.Beat,
	tones map[string]color.RGBA, at func(x, y float64) image.Point,
	scale float64) error {
	for k := range beatsPerBar {
		on := tones["ok"]
		if k == 0 {
			on = tones["ink"]
		}
		level := 0.0
		if beat.On() && beat.InBar() == k+1 {
			level = math.Max(beatLeast, math.Exp(-beat.Since()/beatFall))
		}
		left := panelMargin + float64(k)*(squareSide+squareGap)
		square := image.Rectangle{Min: at(left, squareTop),
			Max: at(left+squareSide, squareTop+squareSide)}
		walk.Fill(frame, square, walk.Mix(walk.Mix(tones["ground"], on,
			offShare), on, level))
	}
	words := "Intro"
	if beat.On() {
		words = fmt.Sprintf("Bar %d", beat.Bar())
	}
	return walk.Words(frame, words, at(panelMargin+beatsPerBar*(squareSide+squareGap)+squareGap, squareTop+squareSide*0.72),
		barSize*scale, tones["ink"])
}

// drawLamps draws the six lamps in a row: each a disc as bright as its
// light, a ring when its cue fires, and its label under it.
func drawLamps(frame *image.RGBA, panel explain.Panel,
	tones map[string]color.RGBA, at func(x, y float64) image.Point,
	scale float64) error {
	lamps := panel.Lamps()
	column := (panelWidth - 2*panelMargin) / float64(len(lamps))
	for i, lamp := range lamps {
		x := panelMargin + column*(float64(i)+0.5)
		centre := at(x, lampTop)
		colour, err := explain.ThemeColour(lampKinds[i].role)
		if err != nil {
			return err
		}
		radius := int(lampRadius * scale)
		if lamp.Ring() > 0.02 {
			walk.Disc(frame, centre, radius+int(ringWidth*scale),
				walk.Mix(tones["ground"], tones["ink"], lamp.Ring()))
			walk.Disc(frame, centre, radius, tones["ground"])
		}
		walk.Disc(frame, centre, radius-1, walk.Mix(walk.Mix(
			tones["ground"], colour, offShare), colour, lamp.Level()))
		wide, err := walk.Measure(lamp.Label(), lampLabelSize*scale)
		if err != nil {
			return err
		}
		label := at(x, lampTop+lampLabelDrop)
		if err := walk.Words(frame, lamp.Label(), label.Sub(image.Pt(
			wide/2, 0)), lampLabelSize*scale, tones["ink"]); err != nil {
			return err
		}
	}
	return nil
}
