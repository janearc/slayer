package main

import (
	"fmt"
	"image"
	"math"

	"github.com/janearc/slayer/improv/moonlake/rig"
	"github.com/janearc/slayer/improv/moonlake/show"
)

// A drawn pane's camera moves inside its still: its window travels from
// one part to another over a bar, starting on a bar line, eased; the two
// drawn panes never move at once. The moves are the script's, below.

// shot is where a window goes at a bar: in on a place of the rig, a
// share of the whole window wide, or out to the whole (a share of one).
type shot struct {
	bar   int
	on    func(rig.Layout) rig.Point
	share float64
}

// The places a window goes in on: the shore heads, the first drums'
// two beams; the ship's beam where it lands. From package rig.
var (
	onShore = func(show rig.Layout) rig.Point {
		heads := show.Heads()
		return heads[2].Origin.Add(heads[3].Origin).Scale(0.5)
	}
	onShip = func(show rig.Layout) rig.Point {
		_, foot := show.Ship(beamOut, beamApex)
		return foot
	}
)

// inShare is how much of the whole window a move in shows. Picked by
// the untraced film's author, 2026-11-12, by looking.
const inShare = 0.5

// aboveShots and ridgeShots are the two drawn panes' moves, by the
// explainer's script: in on the shore heads for the first drums (bar 1,
// "two beams, low"), out for the break (bar 9); in again when the
// strobes and backlights fire on every sixteenth (bar 68), out at the
// calm (bar 74).
//
// The ridge view swings in on the ship's beam the bar before it lands
// (bar 49; it lands at 50), out as it glides away (bar 61), in when it
// returns (bar 73), out when the drums drop (bar 81).
var (
	aboveShots = []shot{{1, onShore, inShare}, {9, nil, 1},
		{68, onShore, inShare}, {74, nil, 1}}
	ridgeShots = []shot{{49, onShip, inShare}, {61, nil, 1},
		{73, onShip, inShare}, {81, nil, 1}}
)

// move is a window's travel: from one part to another between two
// moments, in seconds.
type move struct {
	from, until float64
	start, end  image.Rectangle
}

// movesOf is a pane's moves from its shots, on the score's bar lines,
// each from where the last left the window.
func movesOf(pane view, shots []shot, score *show.Score,
	layout rig.Layout, aspect float64) ([]move, error) {
	var made []move
	at := pane.part
	for _, each := range shots {
		from, err := score.BarTime(each.bar, 1)
		if err != nil {
			return nil, err
		}
		until, err := score.BarTime(each.bar+1, 1)
		if err != nil {
			return nil, err
		}
		target := pane.part
		if each.on != nil {
			target, err = pane.around(each.on(layout), each.share, aspect)
			if err != nil {
				return nil, err
			}
		}
		made = append(made, move{from: from, until: until, start: at,
			end: target})
		at = target
	}
	return made, nil
}

// around is a window a share of the whole one wide, of a shape, centred
// on a place of the rig as far as the still allows.
func (pane view) around(at rig.Point, share, aspect float64) (
	image.Rectangle, error) {
	seen := pane.camera.Project(placeOf(at))
	if !seen.Ahead() {
		return image.Rectangle{}, fmt.Errorf("a move in on %v, which the "+
			"camera does not see", at)
	}
	width := int(float64(pane.part.Dx()) * share)
	height := int(float64(width) / aspect)
	bounds := pane.still.Rect
	left := min(max(int(seen.X())-width/2, bounds.Min.X), bounds.Max.X-width)
	top := min(max(int(seen.Y())-height/2, bounds.Min.Y), bounds.Max.Y-height)
	return image.Rect(left, top, left+width, top+height), nil
}

// partAt is the window at a moment: before a move, where the last one
// left it; during one, eased between its ends.
func partAt(home image.Rectangle, moves []move, seconds float64) image.Rectangle {
	part := home
	for _, each := range moves {
		if seconds < each.from {
			break
		}
		share := math.Min(1, (seconds-each.from)/(each.until-each.from))
		eased := share * share * (3 - 2*share)
		between := func(a, b int) int {
			return a + int(math.Round(float64(b-a)*eased))
		}
		part = image.Rect(between(each.start.Min.X, each.end.Min.X),
			between(each.start.Min.Y, each.end.Min.Y),
			between(each.start.Max.X, each.end.Max.X),
			between(each.start.Max.Y, each.end.Max.Y))
	}
	return part
}
