package main

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/improv/moonlake/rig"
)

// The drawing's numbers. Picked by the untraced film's author,
// fable-team, 2026-11-12, by looking at the panes, unless a line says.
const (
	// raySteps is how many pieces a beam is drawn in, so one that passes
	// behind the camera is cut where it does
	raySteps = 64
	// markRadius and markSteps are a fixture's ring, in pane pixels
	markRadius = 5.0
	markSteps  = 40
	// beamReach is how far a laser beam runs if it meets no water, and
	// searchReach a searchlight's drawn length, in metres; lightshow's
	// beamReach is the first
	beamReach   = 4000.0
	searchReach = 1500.0
	// searchCount is how many searchlights stand: lightshow's
	searchCount = 8
	// beamShare is how bright a laser line is drawn at full gate, and
	// searchShare a searchlight's, strobeRadius a strobe's flash and
	// floodRadius a backlight's, in pane pixels
	beamShare    = 0.9
	searchShare  = 0.6
	strobeRadius = 40.0
	floodRadius  = 30.0
	// shipEdges is how many lines draw the ship's cone, shipShare how
	// bright each; beamOut, beamApex and beamRadius are lightshow's
	// BEAM_OUT, BEAM_APEX and BEAM_RADIUS defaults, which the film ran on
	shipEdges  = 24
	shipShare  = 0.35
	beamOut    = 900.0
	beamApex   = 450.0
	beamRadius = 90.0
	// shown is the least level a light is drawn at: lightshow's
	shown = 0.01
	// fewBeams is how many beams a head draws as lines; past it, a fan
	// is a wedge with that many lines through it; sheetShare is how
	// bright the wedge is laid
	fewBeams   = 6
	sheetShare = 0.12
	// chaseDim is how bright a chase's other lights are: lightshow's
	chaseDim = 0.12
)

// The fixtures' colours, as lightshow lights them: the searchlights a
// cool white, the white strobe and flood, the magenta gel, the ship's
// beam's core.
var (
	searchColour  = [3]float64{0.85, 0.9, 1}
	whiteColour   = [3]float64{1, 1, 1}
	magentaColour = [3]float64{1, 0.25, 0.85}
	shipColour    = [3]float64{0.82, 0.9, 1.0}
	darkColour    = [3]float64{0.55, 0.55, 0.55}
)

// drawRig draws the whole rig on a pane at a moment: every fixture
// marked at its place, and what the cues have on, where the show sends
// it.
func drawRig(picture *image.RGBA, pane view, state cues, layout rig.Layout,
	seconds float64, tones lampTones) []named {
	names := pane.marks(picture, layout, tones, firing(state, seconds))
	pane.lasers(picture, state, layout, seconds)
	pane.searchlights(picture, state, layout)
	pane.strobes(picture, state, layout, seconds)
	pane.floods(picture, state, layout)
	pane.ship(picture, state, layout)
	return names
}

// lampTones are the panel's lamps' colours, as light: each kind of
// fixture is marked in its lamp's colour, so panel and drawings read as
// one legend.
type lampTones struct {
	lasers, strobes, floods, search, ship [3]float64
}

// The kinds of fixture as the panel's lamps show them: each kind's
// mark, its name, and the hud theme's role its lamp lights in.
var (
	laserMark  = shape{corners: 3, turn: -90}
	searchMark = shape{corners: 4, turn: 0}
	strobeMark = shape{corners: 4, turn: 45}
	floodMark  = shape{}
	shipMark   = shape{corners: 6, turn: 0}
)

// marks marks every fixture, lit or dark, each kind in its shape and its
// lamp's colour, and names each kind once beside its fixtures.
func (pane view) marks(picture *image.RGBA, layout rig.Layout,
	tones lampTones, lit levels) []named {
	var heads, strobes, searches, floods []rig.Point
	for _, head := range layout.Heads() {
		heads = append(heads, head.Origin)
	}
	strobePlaces, floodPlaces := layout.Strobes(), layout.Backlights()
	strobes = append(strobes, strobePlaces[:]...)
	for k := range searchCount {
		searches = append(searches, layout.Searchlight(k))
	}
	floods = append(floods, floodPlaces[:]...)
	top, _ := layout.Ship(beamOut, beamApex)
	groups := []struct {
		name    string
		places  []rig.Point
		outline shape
		colour  [3]float64
		level   func(int) float64
	}{
		{"lasers", heads, laserMark, tones.lasers, lit.head},
		{"strobes", strobes, strobeMark, tones.strobes, lit.strobe},
		{"searchlights", searches, searchMark, tones.search, lit.search},
		{"backlights", floods, floodMark, tones.floods, lit.flood},
		{"ship", []rig.Point{top}, shipMark, tones.ship, lit.ship},
	}
	var names []named
	for _, group := range groups {
		for k, place := range group.places {
			pane.mark(picture, place, group.outline, group.colour)
			pane.pulse(picture, place, group.colour, group.level(k))
		}
		if x, y, seen := pane.middle(group.places, picture.Rect.Size()); seen {
			names = append(names, named{group.name, image.Pt(int(x),
				int(y))})
		}
	}
	return names
}

// named is a kind of fixture's name and where on its pane to write it.
type named struct {
	name string
	at   image.Point
}

// middle is where a kind's name goes: at the first of its places the
// pane sees, and whether it sees any.
func (pane view) middle(places []rig.Point, size image.Point) (float64,
	float64, bool) {
	for _, place := range places {
		x, y, ahead := pane.onPane(place, size)
		if ahead && x >= 0 && y >= 0 && x < float64(size.X) &&
			y < float64(size.Y) {
			return x, y, true
		}
	}
	return 0, 0, false
}

// lasers draws each head's beams as lightshow throws them: the figure's
// ways about the head's aim swung by the mirrors, each bank at its
// gate, a split pulse lighting half, to the water or their reach; a rig
// of nought is the blackout.
func (pane view) lasers(picture *image.RGBA, state cues, layout rig.Layout,
	seconds float64) {
	if state.rig == 0 {
		return
	}
	swing, tilt, spread := rig.Sweep(seconds, state.sweep)
	// a rig cued above nought scales the beams, as lightshow does
	scale := 1.0
	if state.rig > 0 {
		scale = state.rig
	}
	gates := state.gates()
	for _, head := range layout.Heads() {
		gate := gates[head.Bank]
		if gate < shown {
			continue
		}
		ways, _, split := rig.FigureWays(state.figure,
			head.Aim+head.Mirror*swing, tilt+head.Lift, spread, state.beams,
			state.figurePhase)
		var ends []rig.Point
		for b, way := range ways {
			if split && state.parity >= 0 && (b+head.Flip)%2 != state.parity {
				continue
			}
			ends = append(ends, layout.BeamEnd(head.Origin, way, beamReach))
		}
		pane.fan(picture, head.Origin, ends, state.colour, gate*scale)
	}
}

// searchlights draws each searchlight's beam on its move, a chase's
// others dim.
func (pane view) searchlights(picture *image.RGBA, state cues,
	layout rig.Layout) {
	level := state.search * state.searchGate
	if level < shown {
		return
	}
	running := rig.Running(searchCount, state.searchMovePhase)
	for k := range searchCount {
		bright := level
		if state.searchMove == "chase" && k != running {
			bright *= chaseDim
		}
		origin := layout.Searchlight(k)
		way := layout.SearchWay(state.searchMove, k, searchCount,
			state.searchPhase, state.searchMovePhase)
		pane.ray(picture, origin, origin.Add(way.Scale(searchReach)),
			searchColour, searchShare*bright)
	}
}

// strobes flashes each strobe that fires in the moment's strobe frame.
func (pane view) strobes(picture *image.RGBA, state cues,
	layout rig.Layout, seconds float64) {
	frame := rig.StrobeFrame(seconds)
	places := layout.Strobes()
	for i, forced := range []int{state.white, state.magenta} {
		if !rig.StrobeFires(uint64(i+1), forced, frame) {
			continue
		}
		colour := []([3]float64){whiteColour, magentaColour}[i]
		x, y, ahead := pane.onPane(places[i], picture.Rect.Size())
		if ahead {
			spot(picture, x, y, strobeRadius, colour, 1)
		}
	}
}

// floods lights the two backlights by their flashes: white and magenta.
func (pane view) floods(picture *image.RGBA, state cues,
	layout rig.Layout) {
	places := layout.Backlights()
	for i, level := range []float64{state.floodLeft, state.floodRight} {
		if level < shown {
			continue
		}
		colour := []([3]float64){whiteColour, magentaColour}[i]
		x, y, ahead := pane.onPane(places[i], picture.Rect.Size())
		if ahead {
			spot(picture, x, y, floodRadius, colour, math.Min(1, level))
		}
	}
}

// ship draws the ship's beam as its cone: from its apex over the water
// to a ring where it lands, as bright as the overhead cue.
func (pane view) ship(picture *image.RGBA, state cues, layout rig.Layout) {
	if state.overhead < shown {
		return
	}
	top, foot := layout.Ship(beamOut, beamApex)
	for k := range shipEdges {
		turn := 2 * math.Pi * float64(k) / shipEdges
		edge := foot.Add(rig.Point{beamRadius * math.Cos(turn),
			beamRadius * math.Sin(turn), 0})
		pane.ray(picture, top, edge, shipColour,
			shipShare*math.Min(1, state.overhead))
	}
}

// fan draws a head's beams: one or a few as lines; more as a sheet of
// light, a translucent wedge between neighbouring beams with a few of
// them drawn brighter through it, so a fan of many reads as one and
// does not crawl as it moves.
func (pane view) fan(picture *image.RGBA, origin rig.Point,
	ends []rig.Point, colour [3]float64, level float64) {
	if len(ends) <= fewBeams {
		for _, end := range ends {
			pane.ray(picture, origin, end, colour, beamShare*level)
		}
		return
	}
	size := picture.Rect.Size()
	x0, y0, ahead := pane.onPane(origin, size)
	for k := 1; ahead && k < len(ends); k++ {
		x1, y1, one := pane.onPane(ends[k-1], size)
		x2, y2, other := pane.onPane(ends[k], size)
		if one && other {
			wedge(picture, [3][2]float64{{x0, y0}, {x1, y1}, {x2, y2}},
				colour, sheetShare*level)
		}
	}
	step := max(1, (len(ends)-1)/(fewBeams-1))
	for k := 0; k < len(ends); k += step {
		pane.ray(picture, origin, ends[k], colour, beamShare*level)
	}
}

// wedge adds light over a triangle of a picture.
func wedge(picture *image.RGBA, corners [3][2]float64, colour [3]float64,
	share float64) {
	low, high := corners[0], corners[0]
	for _, at := range corners[1:] {
		low = [2]float64{math.Min(low[0], at[0]), math.Min(low[1], at[1])}
		high = [2]float64{math.Max(high[0], at[0]), math.Max(high[1], at[1])}
	}
	bounds := picture.Rect
	for y := max(int(low[1]), bounds.Min.Y); y <= min(int(high[1]),
		bounds.Max.Y-1); y++ {
		for x := max(int(low[0]), bounds.Min.X); x <= min(int(high[0]),
			bounds.Max.X-1); x++ {
			if inside(corners, float64(x)+0.5, float64(y)+0.5) {
				lighten(picture, x, y, colour, share)
			}
		}
	}
}

// inside is whether a point is within a triangle, either way round.
func inside(corners [3][2]float64, x, y float64) bool {
	side := func(a, b [2]float64) float64 {
		return (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
	}
	one := side(corners[0], corners[1])
	two := side(corners[1], corners[2])
	three := side(corners[2], corners[0])
	return (one >= 0 && two >= 0 && three >= 0) ||
		(one <= 0 && two <= 0 && three <= 0)
}

// levels is how lit each fixture is at a moment, by its place in its
// kind's list, nought to one: the common accent the drawings pulse
// with, in step with the panel's lamps.
type levels struct {
	head, strobe, search, flood, ship func(int) float64
}

// firing is each fixture's level at a moment, from the cues as
// lightshow lights them.
func firing(state cues, seconds float64) levels {
	gates := state.gates()
	frame := rig.StrobeFrame(seconds)
	forced := []int{state.white, state.magenta}
	floods := []float64{state.floodLeft, state.floodRight}
	running := rig.Running(searchCount, state.searchMovePhase)
	return levels{
		head: func(k int) float64 {
			if state.rig == 0 {
				return 0
			}
			return gates[[]int{0, 1, 1, 0}[k]]
		},
		strobe: func(k int) float64 {
			if rig.StrobeFires(uint64(k+1), forced[k], frame) {
				return 1
			}
			return 0
		},
		search: func(k int) float64 {
			level := state.search * state.searchGate
			if state.searchMove == "chase" && k != running {
				level *= chaseDim
			}
			return level
		},
		flood: func(k int) float64 { return math.Min(1, floods[k]) },
		ship:  func(int) float64 { return math.Min(1, state.overhead) },
	}
}

// pulse lights a fixture's mark in its lamp's colour as far as it is
// lit: a soft disc round the mark, so a firing reads in every drawing
// at the instant the panel's lamp lights.
func (pane view) pulse(picture *image.RGBA, at rig.Point,
	colour [3]float64, level float64) {
	if level < shown {
		return
	}
	x, y, ahead := pane.onPane(at, picture.Rect.Size())
	if ahead {
		spot(picture, x, y, pulseRadius, colour, level)
	}
}

// pulseRadius is a firing mark's glow, in pane pixels. Picked by the
// untraced film's author, 2026-11-12.
const pulseRadius = 3 * markRadius

// lighten raises a pixel to the colour's channels times a share where it
// is darker, and leaves it where it is lighter: overlapping sheets of
// light do not pile up into a slab.
func lighten(picture *image.RGBA, x, y int, colour [3]float64,
	share float64) {
	was := picture.RGBAAt(x, y)
	up := func(channel uint8, light float64) uint8 {
		return uint8(math.Max(float64(channel), math.Min(math.MaxUint8,
			float64(channel)*(1-share)+math.MaxUint8*light*share)))
	}
	picture.SetRGBA(x, y, color.RGBA{up(was.R, colour[0]),
		up(was.G, colour[1]), up(was.B, colour[2]), math.MaxUint8})
}
