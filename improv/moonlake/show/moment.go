package show

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Moment is what the lights are doing at one moment, as lightshow obeys
// it: every cue up to the moment folded, and the grid's pulses, sweeps
// and strobes worked out for the frame that starts there. A bar is the
// score's meter in beats.
type Moment struct {
	seconds, position      float64
	meter                  float64
	now                    state
	colour                 [3]float64
	sweep                  float64
	white, magenta         int
	gate, gateRight        float64
	parity                 int
	figure, banks, move    string
	figurePhase, movePhase float64
	floodLeft, floodRight  float64
	spotAim, searchPhase   float64
	searchGate, moon       float64
	bursts                 []float64
	fireLit                float64
}

// At is the lights at a moment, in seconds, for a frame of 1/fps
// seconds: a lead note starting inside the frame fires a strobe. A
// frame rate of nought or less is refused by name.
func (of *Sheet) At(seconds, fps float64) (Moment, error) {
	if !(fps > 0) || math.IsInf(fps, 0) {
		return Moment{}, fmt.Errorf("%w: a frame rate of %v; give one "+
			"above nought", ErrShow, fps)
	}
	beats, meter := of.score.beats, float64(of.score.meter)
	made := Moment{seconds: seconds, now: of.fold(seconds),
		position: gridAt(seconds, beats), meter: meter}
	made.palette()
	made.white, made.magenta = made.strobed(of.score.tracks["lead"], fps)
	made.lasers()
	made.floodLeft, made.floodRight = flooded(made.now.floods, seconds,
		made.position, meter, beats, of.score.tracks["lead"], made.now.hot)
	made.searchlights()
	made.bursts = bursts(seconds, made.position, made.now.fireFrom, meter,
		beats)
	made.fireLit = lit(seconds, made.position, made.now.fireFrom, beats)
	made.moon = made.moonLevel()
	return made, nil
}

// palette is the beam colour for the beat, and the mirrors' sweep.
func (made *Moment) palette() {
	made.colour = paletteColours[made.now.palette[0]]
	made.sweep = noGrid
	if made.position < 0 {
		return
	}
	beat := int(made.position) % int(made.meter)
	made.colour = paletteColours[made.now.palette[beat%len(made.now.palette)]]
	made.sweep = made.position / made.meter / made.now.sweepBars
	made.sweep -= math.Floor(made.sweep)
}

// strobed is the strobes for the frame: forced off, left to lightshow's
// lattice, burning the first share of each grid subdivision, or riding
// the lead's loud notes inside the frame. The magenta gel takes the off
// subdivisions, and the lead's hardest notes.
func (made *Moment) strobed(lead []Note, fps float64) (white, magenta int) {
	switch made.now.strobes {
	case strobesLattice:
		return latticeFires, latticeFires
	case strobesGrid16, strobesGrid8:
		if made.position < 0 {
			return 0, 0
		}
		sub := made.position * map[string]float64{
			strobesGrid16: stepsPerBeat, strobesGrid8: eighthsPerBeat,
		}[made.now.strobes]
		if sub-math.Floor(sub) >= made.now.duty {
			return 0, 0
		}
		if int(sub)%2 == 0 {
			return 1, 0
		}
		return 0, 1
	case strobesLead:
		return made.leadStrobes(lead, fps)
	}
	return 0, 0
}

// leadStrobes is the strobes a lead note starting in the frame fires.
func (made *Moment) leadStrobes(lead []Note, fps float64) (white,
	magenta int) {
	for _, one := range lead {
		if one.start < made.seconds || one.start >= made.seconds+1/fps {
			continue
		}
		if one.vel >= made.now.hotter {
			magenta = 1
		} else if one.vel >= made.now.hot {
			white = 1
		}
	}
	return white, magenta
}

// lasers is the pulse's gate on each bank, which hit it is, and the
// figure the fans wear: offset, the right heads play the pulse a
// sixteenth behind the left.
func (made *Moment) lasers() {
	var hit int
	made.gate, hit = gated(made.now.pulse, made.now.decay, made.position)
	made.parity = -1
	if made.now.split {
		made.parity = hit % 2
	}
	made.figure, made.figurePhase = inTurn(made.now.figures,
		made.now.figureBars, made.meter, made.now.figuresFrom, made.position)
	if made.figure == "" {
		made.figure = restFigure
	}
	made.banks, _ = inTurn(made.now.banks, made.now.banksBars, made.meter,
		made.now.banksFrom, made.position)
	if made.banks == "" {
		made.banks = restBanks
	}
	made.gateRight = made.gate
	if made.banks == banksOffset && made.position >= 0 {
		made.gateRight, _ = gated(made.now.pulse, made.now.decay,
			made.position-1.0/stepsPerBeat)
	}
}

// searchlights is the spots' aim, the searchlights' sweep, their move
// and how hard the beat strikes them.
func (made *Moment) searchlights() {
	now, position := made.now, made.position
	if position >= 0 && now.spotBars > 0 {
		phase := position / made.meter / now.spotBars
		phase -= math.Floor(phase)
		made.spotAim = sweepTurn * (1 - math.Abs(2*phase-1))
	}
	if position >= 0 && now.searchBars > 0 {
		made.searchPhase = position / made.meter / now.searchBars
		made.searchPhase -= math.Floor(made.searchPhase)
	}
	made.move, made.movePhase = inTurn(now.searchMoves, now.searchMoveBars,
		made.meter, now.searchMovesFrom, position)
	if made.move == "" {
		made.move = restMove
	}
	made.searchGate = 1
	if position >= 0 && now.searchBeat > 0 {
		beat := math.Exp(-(position - math.Floor(position)) / searchStrike)
		made.searchGate = 1 - now.searchBeat + now.searchBeat*beat
	}
}

// moonLevel is the moon's disc, struck on each beat and falling by
// moonBeat as deep as its pulse.
func (made *Moment) moonLevel() float64 {
	beat := 0.0
	if made.position >= 0 {
		beat = math.Exp(-(made.position - math.Floor(made.position)) /
			moonBeat)
	}
	pulse := made.now.moonPulse
	return made.now.moon.at(made.seconds) * (1 - pulse + pulse*beat)
}

// Lines is the moment as cueshow says it: key=value lines for
// lightshow.
func (made Moment) Lines() string {
	var out strings.Builder
	now, at := made.now, made.seconds
	fmt.Fprintf(&out, "rig=%.3f\nbeams=%d\n", made.Rig(), now.beams)
	fmt.Fprintf(&out, "colour=%.3f,%.3f,%.3f\n", made.colour[0],
		made.colour[1], made.colour[2])
	fmt.Fprintf(&out, "sweep=%.4f\nwhite=%d\nmagenta=%d\nmist=%.3f\n",
		made.sweep, made.white, made.magenta, now.mist.at(at))
	fmt.Fprintf(&out, "gate=%.4f\nparity=%d\n", made.gate, made.parity)
	fmt.Fprintf(&out, "figure=%s\nfigure_phase=%.4f\nbanks=%s\n",
		made.figure, made.figurePhase, made.banks)
	fmt.Fprintf(&out, "gate_left=%.4f\ngate_right=%.4f\n", made.gate,
		made.gateRight)
	fmt.Fprintf(&out, "flood_left=%.4f\nflood_right=%.4f\n",
		made.floodLeft, made.floodRight)
	fmt.Fprintf(&out, "spots=%.3f\nspot_aim=%.4f\n", now.spots, made.spotAim)
	fmt.Fprintf(&out, "search=%.3f\nsearch_phase=%.4f\n", now.search,
		made.searchPhase)
	fmt.Fprintf(&out, "search_move=%s\nsearch_move_phase=%.4f\n"+
		"search_gate=%.4f\n", made.move, made.movePhase, made.searchGate)
	fmt.Fprintf(&out, "master=%.4f\noverhead=%.4f\nexposure=%.3f\n",
		made.Master(), made.Overhead(), now.exposure.at(at))
	fmt.Fprintf(&out, "fire=%.4f\n", now.fire.at(at))
	var since []string
	for _, ago := range made.bursts {
		since = append(since, strconv.FormatFloat(ago, 'f', 4, 64))
	}
	fmt.Fprintf(&out, "fire_since=%s\nfire_lit=%.4f\n",
		strings.Join(since, ","), made.fireLit)
	fmt.Fprintf(&out, "moon=%.4f\nmoon_alt=%.4f\n", made.moon,
		now.moonAlt.at(at))
	fmt.Fprintf(&out, "bar=%s\n", made.bar())
	return out.String()
}

// bar is the bar and beat for the eye, or intro before the grid.
func (made Moment) bar() string {
	if made.position < 0 {
		return "intro"
	}
	beat, meter := int(made.position), int(made.meter)
	return fmt.Sprintf("%d.%d", beat/meter+1, beat%meter+1)
}
