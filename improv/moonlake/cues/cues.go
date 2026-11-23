package cues

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// colours is each palette name as linear srgb, near enough. green is
// lightshow's own 532 nm green.
func colours() map[string][3]float64 {
	return map[string][3]float64{
		"green": {0.08, 1, 0.12}, "red": {1, 0.04, 0.04},
		"blue": {0.08, 0.2, 1}, "cyan": {0.05, 0.85, 1},
		"amber": {1, 0.55, 0.08}, "magenta": {1, 0.25, 0.85},
		"white": {1, 1, 1},
	}
}

// note is a transcribed note: where it starts, in the file's seconds,
// and how hard it is struck.
type note struct {
	Sec   float64 `yaml:"sec"`
	AtSec float64 `yaml:"at_sec"`
	Vel   int     `yaml:"vel"`
}

// start is the note's start however the score wrote it.
func (of note) start() float64 { return math.Max(of.Sec, of.AtSec) }

// score is what cueshow reads of score.yaml: the grid and the lead.
type score struct {
	Score struct {
		Duration float64           `yaml:"duration"`
		Tracks   map[string][]note `yaml:"tracks"`
		Beats    struct {
			Times []float64 `yaml:"times"`
		} `yaml:"beats"`
	} `yaml:"score"`
}

// cue is one line of the sheet: when, and only the fields it names.
type cue struct {
	At        string   `yaml:"at"`
	AtSec     *float64 `yaml:"at_sec"`
	Rig       *float64 `yaml:"rig"`
	Beams     *int     `yaml:"beams"`
	Palette   []string `yaml:"palette"`
	SweepBars *float64 `yaml:"sweep_bars"`
	Strobes   *string  `yaml:"strobes"`
	Duty      *float64 `yaml:"duty"`
	Hot       *int     `yaml:"hot"`
	Hotter    *int     `yaml:"hotter"`
	Mist      *float64 `yaml:"mist"`
	// the lasers' pulse: sixteen steps over a bar, x a hit and . a rest;
	// decay is how many sixteenths a hit takes to fall to a third; split
	// lights the fan's even beams on one hit and its odd on the next
	Pulse *string  `yaml:"pulse"`
	Decay *float64 `yaml:"decay"`
	Split *bool    `yaml:"split"`
	// the flank floods: off, lead (the acid line's loud notes), grid16,
	// grid8 or ones; each hit fires one side, left and right in turn
	Floods *string `yaml:"floods"`
	// the flank spots: their level, 0 to 1, and how many bars a sweep
	// across and back takes
	Spots    *float64 `yaml:"spots"`
	SpotBars *float64 `yaml:"spot_bars"`
	// the searchlights behind the rim: their level, 0 to 1, and how many
	// bars a sweep out and back takes
	Search     *float64 `yaml:"search"`
	SearchBars *float64 `yaml:"search_bars"`
	// master dims the moonlit scene, 0 to 1; overhead is the ship's light
	// from above, 0 to 1 of daylight; exposure is the camera's, in stops
	// from the moon's
	Master   *float64 `yaml:"master"`
	Overhead *float64 `yaml:"overhead"`
	Exposure *float64 `yaml:"exposure"`
	// fire is the shore fires' level, 0 to 1: above nought, the fires
	// burst on every bar's first beat, each burst its own fire from
	// ignition, flaring and fading under the next
	Fire *float64 `yaml:"fire"`
	// the moon: its disc's level, 0 to 1; its altitude in degrees, the
	// rim standing at 1.65 and the disc 1.55 in radius; and how deep it
	// pulses on the beat, 0 steady to 1 dark between beats
	Moon      *float64 `yaml:"moon"`
	MoonAlt   *float64 `yaml:"moon_alt"`
	MoonPulse *float64 `yaml:"moon_pulse"`
	// the lasers as an instrument: figures is the looks they wear in
	// turn, figure_bars bars each, from this cue (beam, pair, fan,
	// fan64, scan, cone, wave); banks is how the left heads and the right
	// play the pulse in turn, banks_bars bars each: unison, or offset,
	// the right a sixteenth behind, so the hits go back and forth
	Figures    []string `yaml:"figures"`
	FigureBars *float64 `yaml:"figure_bars"`
	Banks      []string `yaml:"banks"`
	BanksBars  *float64 `yaml:"banks_bars"`
	// the searchlights' moves in turn, search_move_bars bars each from
	// this cue (sweep, chase, fan, viewer), and search_beat, how deep each
	// beat strikes them, 0 steady to 1 dark between beats
	SearchMoves    []string `yaml:"search_moves"`
	SearchMoveBars *float64 `yaml:"search_move_bars"`
	SearchBeat     *float64 `yaml:"search_beat"`
	// fade is how many seconds rig, mist, master, overhead, exposure,
	// fire, moon and moon_alt take to glide from where they are to what
	// this cue says; none is a cut
	Fade *float64 `yaml:"fade"`
}

type sheet struct {
	Cues []cue `yaml:"cues"`
}

// state is the lights' state once every cue up to the moment has
// spoken: the defaults are a dark rig over the resting mist.
type state struct {
	rig, sweepBars, duty, mist, decay   float64
	spots, spotBars, search, searchBars float64
	beams, hot, hotter                  int
	palette                             []string
	strobes, pulse, floods              string
	split                               bool
	// the fading levels
	rigFade, mistFade, master, overhead, exposure, fire ramp
	moon, moonAlt                                       ramp
	// fireFrom is where on the grid the fires' bursts are counted from,
	// in beats, or below nought when no cue has lit them
	fireFrom float64
	// moonPulse is how deep the moon pulses on the beat
	moonPulse float64
	// the lasers' figures and banks, each a list taken in turn so many
	// bars apiece from where on the grid its cue stands
	figures, banks         []string
	figureBars, banksBars  float64
	figuresFrom, banksFrom float64
	// the searchlights' moves, taken in turn the same way, and how deep
	// the beat strikes them
	searchMoves                     []string
	searchMoveBars, searchMovesFrom float64
	searchBeat                      float64
}

// figureNames, bankNames and moveNames are the figures, banks and
// searchlight moves lightshow plays.
func figureNames() []string { return []string{"beam", "pair", "fan", "fan64", "scan", "cone", "wave"} }

// bankNames are the banks of laser heads lightshow plays.
func bankNames() []string { return []string{"unison", "offset"} }

// moveNames are the searchlight moves lightshow plays.
func moveNames() []string { return []string{"sweep", "chase", "fan", "viewer"} }

// known is a cue's list of names, each one lightshow plays, or the cue
// fails saying which it is not.
func known(at float64, what string, list, names []string) ([]string, error) {
	for _, name := range list {
		if !slices.Contains(names, name) {
			return nil, fmt.Errorf("cue at %v: no %s %q; choose %v", at, what, name,
				names)
		}
	}
	return list, nil
}

// inTurn is which of a list holds at a place on the grid, taken in turn
// so many bars apiece from a place, and how far through its bars it is;
// the first before the grid or the place, and nothing of an empty list.
func inTurn(list []string, bars, from, position float64) (string, float64) {
	if len(list) == 0 {
		return "", 0
	}
	if bars <= 0 || position < 0 || position < from {
		return list[0], 0
	}
	turns := (position - from) / 4 / bars
	k := int(math.Floor(turns))
	return list[k%len(list)], turns - float64(k)
}

const (
	// setMoonAlt is the moon's altitude in sets/moonlake.yaml, where it
	// stands until a cue moves it
	setMoonAlt = 2.3
	// moonBeat is how much of a beat the moon's pulse takes to fall to a
	// third
	moonBeat = 0.3
)

// gridAt is a moment's place on the grid: beats from its start,
// fractional, or less than nought before the first drum.
func gridAt(seconds float64, beats []float64) float64 {
	if len(beats) < 2 || seconds < beats[0] {
		return -1
	}
	index := len(beats) - 2
	for i := range len(beats) - 1 {
		if beats[i] > seconds {
			index = i - 1
			break
		}
	}
	span := beats[index+1] - beats[index]
	return float64(index) + (seconds-beats[index])/span
}

// burstSpan is how long after a downbeat its fire burst still shows, in
// seconds: the fire is rendered from ignition this far, 82 frames at 24
// a second, two bars at the song's tempo.
const burstSpan = 82.0 / 24

// bursts is how long ago each burst still showing began, newest first:
// a burst on the beat from, on the grid, and every four beats after, up
// to the moment at position. None before from, or without one.
func bursts(seconds, position, from float64, beats []float64) []float64 {
	if from < 0 || position < from {
		return nil
	}
	var since []float64
	for n := math.Floor((position - from) / 4); n >= 0; n-- {
		ago := seconds - beatTime(from+4*n, beats)
		if ago >= burstSpan {
			break
		}
		if ago >= 0 {
			since = append(since, ago)
		}
	}
	return since
}

// lit is how long ago the fires were lit, in seconds: since the fire cue's
// beat, from, on the grid; -1 before it, or without one. lightshow tells
// the ignition's burst from the downbeats' by it, and keeps the fire
// burning between them.
func lit(seconds, position, from float64, beats []float64) float64 {
	if from < 0 || position < from {
		return -1
	}
	return seconds - beatTime(from, beats)
}

// ramp is a level gliding from one value to another, from a moment, over
// some seconds, eased in and out.
type ramp struct{ from, to, start, seconds float64 }

// held is a ramp resting at a value.
func held(value float64) ramp { return ramp{value, value, 0, 0} }

// at is the ramp's value at a moment.
func (r ramp) at(t float64) float64 {
	if r.seconds <= 0 || t >= r.start+r.seconds {
		if t < r.start {
			return r.from
		}
		return r.to
	}
	if t <= r.start {
		return r.from
	}
	f := (t - r.start) / r.seconds
	f = f * f * (3 - 2*f)
	return r.from + (r.to-r.from)*f
}

// glide sets a ramp off from where it is at a moment toward a value.
func (r *ramp) glide(to, start, seconds float64) {
	*r = ramp{from: r.at(start), to: to, start: start, seconds: seconds}
}

// floodFlash is how long a flood's flash takes to fall to a third, in
// seconds: a strobe's afterglow as a camera sees it
const floodFlash = 0.08

// flooded is the two floods' levels at a moment: each hit fires one
// side, the sides in turn, and a hit falls by floodFlash.
func flooded(mode string, seconds, position float64, beats []float64,
	lead []note, hot int) (left, right float64) {
	// a hit is when the floods fire and its count, which picks its side
	type hit struct {
		at    float64
		count int
	}
	var hits []hit
	switch mode {
	case "grid16", "grid8", "ones":
		if position < 0 {
			return 0, 0
		}
		per := map[string]float64{"grid16": 4, "grid8": 2, "ones": 0.25}[mode]
		step := math.Floor(position * per)
		for back := 0.0; back < 3; back++ {
			if at := beatTime((step-back)/per, beats); at >= 0 && at <= seconds {
				hits = append(hits, hit{at, int(step - back)})
			}
		}
	case "lead":
		for i, one := range lead {
			at := one.start()
			if at <= seconds && at > seconds-0.5 && one.Vel >= hot {
				hits = append(hits, hit{at, i})
			}
		}
	default:
		return 0, 0
	}
	for _, h := range hits {
		level := math.Exp(-(seconds - h.at) / floodFlash)
		if h.count%2 == 0 {
			left = math.Max(left, level)
		} else {
			right = math.Max(right, level)
		}
	}
	return left, right
}

// beatTime is a place on the grid, in beats, as seconds: the measured
// beat it is in, and the way to the next.
func beatTime(position float64, beats []float64) float64 {
	if position < 0 || len(beats) < 2 {
		return -1
	}
	i := int(math.Floor(position))
	if i >= len(beats)-1 {
		span := beats[len(beats)-1] - beats[len(beats)-2]
		return beats[len(beats)-1] + (position-float64(len(beats)-1))*span
	}
	return beats[i] + (position-float64(i))*(beats[i+1]-beats[i])
}

// gated is the lasers' level under a pulse at a place on the grid, in
// beats, and which hit it is, counted from the grid's start: the last
// hit at or before the moment, fallen by the decay since. A pulse with
// no hits holds the fan dark.
func gated(pulse string, decay, position float64) (float64, int) {
	steps := len(pulse)
	if steps == 0 || position < 0 {
		return 1, 0
	}
	hits := strings.Count(pulse, "x")
	sixteenth := position * 4
	now := int(math.Floor(sixteenth))
	for back := range steps {
		step := now - back
		if step < 0 || pulse[step%steps] != 'x' {
			continue
		}
		since := sixteenth - float64(step)
		bar, in := step/steps, step%steps
		return math.Exp(-since / decay), bar*hits + strings.Count(pulse[:in], "x")
	}
	return 0, 0
}

// when is a cue's moment in seconds: at_sec as it is, or bar.beat
// looked up in the grid.
func when(of cue, beats []float64) (float64, error) {
	if of.AtSec != nil {
		return *of.AtSec, nil
	}
	fields := strings.SplitN(of.At, ".", 2)
	bar, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, fmt.Errorf("cue at %q: %w", of.At, err)
	}
	beat := 1
	if len(fields) == 2 {
		if beat, err = strconv.Atoi(fields[1]); err != nil {
			return 0, fmt.Errorf("cue at %q: %w", of.At, err)
		}
	}
	index := (bar-1)*4 + beat - 1
	if index < 0 || index >= len(beats) {
		return 0, fmt.Errorf("cue at %q: bar off the grid", of.At)
	}
	return beats[index], nil
}

type Show struct {
	written score
	sheet   sheet
	times   []float64
	errors  []error
}

// Parse reads a score and a cue sheet, both YAML, and works out each cue's
// time. A cue whose time cannot be resolved is not an error here; At
// reports it.
func Parse(scoreYAML, cuesYAML []byte) (*Show, error) {
	var show Show
	if err := yaml.Unmarshal(scoreYAML, &show.written); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(cuesYAML, &show.sheet); err != nil {
		return nil, err
	}
	show.compile()
	return &show, nil
}

// compile works out the time of each cue in the sheet and keeps it, or the
// error from working it out, in step with the cue. At reports those errors.
func (show *Show) compile() {
	for _, line := range show.sheet.Cues {
		at, err := when(line, show.written.Score.Beats.Times)
		show.times = append(show.times, at)
		show.errors = append(show.errors, err)
	}
}

// At is the frame for the moment seconds, where each frame lasts 1/fps. It
// applies every cue at or before seconds in the order the sheet gives them.
// The fps decides which notes of the lead fall inside this frame, for the
// strobes. It refuses a nil show, a seconds or fps that is not finite, and
// any cue whose time did not resolve, even one that is later than seconds.
func (show *Show) At(seconds, fps float64) (Frame, error) {
	if show == nil {
		return Frame{}, fmt.Errorf("cues: show is nil")
	}
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || math.IsNaN(fps) || math.IsInf(fps, 0) {
		return Frame{}, fmt.Errorf("cues: seconds and fps must be finite")
	}
	written, sheet := show.written, show.sheet
	beats := written.Score.Beats.Times
	// hot and hotter rest at the lead's own loudness: its 90th and
	// 95th centiles, so a tenth of its notes strike a strobe.
	now := state{rig: 0, beams: 10, sweepBars: 8, duty: 0.25,
		mist: 0.12, palette: []string{"green"}, strobes: "off",
		hot: 78, hotter: 88, pulse: "x...x...x...x...", decay: 1.5,
		floods: "off", spots: 0, spotBars: 2, search: 0, searchBars: 4,
		rigFade: held(0), mistFade: held(0.12), master: held(1),
		overhead: held(0), exposure: held(0), fire: held(0), fireFrom: -1,
		moon: held(1), moonAlt: held(setMoonAlt)}
	for i, line := range sheet.Cues {
		if show.errors[i] != nil {
			return Frame{}, show.errors[i]
		}
		at := show.times[i]
		if at > seconds {
			continue
		}
		fade := 0.0
		if line.Fade != nil {
			fade = *line.Fade
		}
		for _, each := range []struct {
			said *float64
			r    *ramp
		}{
			{line.Rig, &now.rigFade}, {line.Mist, &now.mistFade},
			{line.Master, &now.master}, {line.Overhead, &now.overhead},
			{line.Exposure, &now.exposure}, {line.Fire, &now.fire},
			{line.Moon, &now.moon}, {line.MoonAlt, &now.moonAlt},
		} {
			if each.said != nil {
				each.r.glide(*each.said, at, fade)
			}
		}
		if line.Beams != nil {
			now.beams = *line.Beams
		}
		if len(line.Palette) > 0 {
			now.palette = line.Palette
		}
		if line.SweepBars != nil {
			now.sweepBars = *line.SweepBars
		}
		if line.Strobes != nil {
			now.strobes = *line.Strobes
		}
		if line.Duty != nil {
			now.duty = *line.Duty
		}
		if line.Hot != nil {
			now.hot = *line.Hot
		}
		if line.Hotter != nil {
			now.hotter = *line.Hotter
		}
		if line.Pulse != nil {
			now.pulse = *line.Pulse
		}
		if line.Decay != nil {
			now.decay = *line.Decay
		}
		if line.Split != nil {
			now.split = *line.Split
		}
		if line.Floods != nil {
			now.floods = *line.Floods
		}
		if line.Spots != nil {
			now.spots = *line.Spots
		}
		if line.SpotBars != nil {
			now.spotBars = *line.SpotBars
		}
		if line.Search != nil {
			now.search = *line.Search
		}
		if line.SearchBars != nil {
			now.searchBars = *line.SearchBars
		}
		// the fires burst on the beat of the cue that lights them and
		// every four beats after: where the sheet puts it is the phase
		if line.Fire != nil {
			now.fireFrom = gridAt(at, beats)
		}
		if line.MoonPulse != nil {
			now.moonPulse = *line.MoonPulse
		}
		if len(line.Figures) > 0 {
			var err error
			now.figures, err = known(at, "figure", line.Figures, figureNames())
			if err != nil {
				return Frame{}, err
			}
			now.figuresFrom = gridAt(at, beats)
		}
		if line.FigureBars != nil {
			now.figureBars = *line.FigureBars
		}
		if len(line.Banks) > 0 {
			var err error
			now.banks, err = known(at, "banks", line.Banks, bankNames())
			if err != nil {
				return Frame{}, err
			}
			now.banksFrom = gridAt(at, beats)
		}
		if line.BanksBars != nil {
			now.banksBars = *line.BanksBars
		}
		if len(line.SearchMoves) > 0 {
			var err error
			now.searchMoves, err = known(at, "move", line.SearchMoves, moveNames())
			if err != nil {
				return Frame{}, err
			}
			now.searchMovesFrom = gridAt(at, beats)
		}
		if line.SearchMoveBars != nil {
			now.searchMoveBars = *line.SearchMoveBars
		}
		if line.SearchBeat != nil {
			now.searchBeat = *line.SearchBeat
		}
	}
	now.rig, now.mist = now.rigFade.at(seconds), now.mistFade.at(seconds)
	position := gridAt(seconds, beats)

	colour := colours()[now.palette[0]]
	sweep := -1.0
	if position >= 0 {
		beat := int(position) % 4
		colour = colours()[now.palette[beat%len(now.palette)]]
		sweep = position / 4 / now.sweepBars
		sweep -= math.Floor(sweep)
	}

	// the strobes: forced off, left to lightshow's lattice, burning
	// the first share of each grid subdivision, or riding the lead's
	// loud notes inside this frame. the magenta gel takes the off
	// subdivisions, and the lead's hardest notes.
	white, magenta := 0, 0
	switch now.strobes {
	case "lattice":
		white, magenta = -1, -1
	case "grid16", "grid8":
		if position >= 0 {
			times := 4.0
			if now.strobes == "grid8" {
				times = 2.0
			}
			sub := position * times
			if sub-math.Floor(sub) < now.duty {
				if int(sub)%2 == 0 {
					white = 1
				} else {
					magenta = 1
				}
			}
		}
	case "lead":
		for _, one := range written.Score.Tracks["lead"] {
			at := one.start()
			if at >= seconds && at < seconds+1/fps {
				if one.Vel >= now.hotter {
					magenta = 1
				} else if one.Vel >= now.hot {
					white = 1
				}
			}
		}
	}

	frame := Frame{Rig: now.rig}
	frame.Beams = now.beams
	frame.Colour = colour
	frame.Sweep = sweep
	frame.White = white
	frame.Magenta = magenta
	frame.Mist = now.mist
	gate, hit := gated(now.pulse, now.decay, position)
	frame.Gate = gate
	parity := -1
	if now.split {
		parity = hit % 2
	}
	frame.Parity = parity
	// the lasers' figure now and how far through its bars, and the banks:
	// offset, the right heads play the pulse a sixteenth behind the left
	figure, phase := inTurn(now.figures, now.figureBars, now.figuresFrom, position)
	if figure == "" {
		figure = "fan"
	}
	banks, _ := inTurn(now.banks, now.banksBars, now.banksFrom, position)
	if banks == "" {
		banks = "unison"
	}
	right := gate
	if banks == "offset" && position >= 0 {
		right, _ = gated(now.pulse, now.decay, position-0.25)
	}
	frame.Figure, frame.FigurePhase, frame.Banks = figure, phase, banks
	frame.GateLeft, frame.GateRight = gate, right
	left, right := flooded(now.floods, seconds, position, beats,
		written.Score.Tracks["lead"], now.hot)
	frame.FloodLeft, frame.FloodRight = left, right
	// the spots sweep across and back once in spot_bars bars: aim runs 0
	// to 2 and back, the left spots' way; the right ones' is its mirror
	aim := 0.0
	if position >= 0 && now.spotBars > 0 {
		phase := position / 4 / now.spotBars
		phase -= math.Floor(phase)
		aim = 2 * (1 - math.Abs(2*phase-1))
	}
	frame.Spots, frame.SpotAim = now.spots, aim
	// the searchlights' sweep: a phase through search_bars bars, 0 to 1
	sway := 0.0
	if position >= 0 && now.searchBars > 0 {
		sway = position / 4 / now.searchBars
		sway -= math.Floor(sway)
	}
	frame.Search, frame.SearchPhase = now.search, sway
	// the searchlights' move now, how far through it, and their gate,
	// struck on each beat as deep as search_beat says
	move, through := inTurn(now.searchMoves, now.searchMoveBars,
		now.searchMovesFrom, position)
	if move == "" {
		move = "sweep"
	}
	struck := 1.0
	if position >= 0 && now.searchBeat > 0 {
		beat := math.Exp(-(position - math.Floor(position)) / 0.35)
		struck = 1 - now.searchBeat + now.searchBeat*beat
	}
	frame.SearchMove, frame.SearchMovePhase, frame.SearchGate = move, through, struck
	frame.Master, frame.Overhead, frame.Exposure = now.master.at(seconds), now.overhead.at(seconds), now.exposure.at(seconds)
	// the fires: their level, and how long ago each burst still showing
	// began, newest first
	frame.Fire = now.fire.at(seconds)
	frame.FireSince = bursts(seconds, position, now.fireFrom, beats)
	frame.FireLit = lit(seconds, position, now.fireFrom, beats)
	// the moon: its level, struck on each beat and falling by moonBeat
	// as deep as its pulse, and where it stands
	beat := 0.0
	if position >= 0 {
		beat = math.Exp(-(position - math.Floor(position)) / moonBeat)
	}
	moon := now.moon.at(seconds) * (1 - now.moonPulse + now.moonPulse*beat)
	frame.Moon, frame.MoonAlt = moon, now.moonAlt.at(seconds)
	if position >= 0 {
		frame.Bar = fmt.Sprintf("%d.%d", int(position)/4+1, int(position)%4+1)
	} else {
		frame.Bar = "intro"
	}
	return frame, nil
}

type Frame struct {
	Rig                                                     float64
	Beams                                                   int
	Colour                                                  [3]float64
	Sweep                                                   float64
	White, Magenta                                          int
	Mist, Gate                                              float64
	Parity                                                  int
	Figure                                                  string
	FigurePhase                                             float64
	Banks                                                   string
	GateLeft, GateRight, FloodLeft, FloodRight              float64
	Spots, SpotAim, Search, SearchPhase                     float64
	SearchMove                                              string
	SearchMovePhase, SearchGate, Master, Overhead, Exposure float64
	Fire                                                    float64
	FireSince                                               []float64
	FireLit, Moon, MoonAlt                                  float64
	Bar                                                     string
}

// WriteTo writes the frame as one name=value line per field, with the
// number of digits fixed for each field and fire_since joined by commas. It
// returns the bytes written.
func (f Frame) WriteTo(w io.Writer) (int64, error) {
	var since []string
	for _, ago := range f.FireSince {
		since = append(since, strconv.FormatFloat(ago, 'f', 4, 64))
	}
	n, err := fmt.Fprintf(w, "rig=%.3f\nbeams=%d\ncolour=%.3f,%.3f,%.3f\nsweep=%.4f\nwhite=%d\nmagenta=%d\nmist=%.3f\ngate=%.4f\nparity=%d\nfigure=%s\nfigure_phase=%.4f\nbanks=%s\ngate_left=%.4f\ngate_right=%.4f\nflood_left=%.4f\nflood_right=%.4f\nspots=%.3f\nspot_aim=%.4f\nsearch=%.3f\nsearch_phase=%.4f\nsearch_move=%s\nsearch_move_phase=%.4f\nsearch_gate=%.4f\nmaster=%.4f\noverhead=%.4f\nexposure=%.3f\nfire=%.4f\nfire_since=%s\nfire_lit=%.4f\nmoon=%.4f\nmoon_alt=%.4f\nbar=%s\n",
		f.Rig, f.Beams, f.Colour[0], f.Colour[1], f.Colour[2], f.Sweep, f.White, f.Magenta, f.Mist, f.Gate, f.Parity, f.Figure, f.FigurePhase, f.Banks, f.GateLeft, f.GateRight, f.FloodLeft, f.FloodRight, f.Spots, f.SpotAim, f.Search, f.SearchPhase, f.SearchMove, f.SearchMovePhase, f.SearchGate, f.Master, f.Overhead, f.Exposure, f.Fire, strings.Join(since, ","), f.FireLit, f.Moon, f.MoonAlt, f.Bar)
	return int64(n), err
}

// Quantized is the frame with each number rounded to the digits WriteTo
// prints for it, so a frame compares equal to one read back from text.
func (f Frame) Quantized() Frame {
	for _, group := range []struct {
		digits int
		values []*float64
	}{
		{3, []*float64{&f.Rig, &f.Colour[0], &f.Colour[1], &f.Colour[2], &f.Mist, &f.Spots, &f.Search, &f.Exposure}},
		{4, []*float64{&f.Sweep, &f.Gate, &f.FigurePhase, &f.GateLeft, &f.GateRight, &f.FloodLeft, &f.FloodRight, &f.SpotAim, &f.SearchPhase, &f.SearchMovePhase, &f.SearchGate, &f.Master, &f.Overhead, &f.Fire, &f.FireLit, &f.Moon, &f.MoonAlt}},
	} {
		for _, value := range group.values {
			*value, _ = strconv.ParseFloat(strconv.FormatFloat(*value, 'f', group.digits, 64), 64)
		}
	}
	if f.FireSince != nil {
		since := make([]float64, len(f.FireSince))
		for i, value := range f.FireSince {
			since[i], _ = strconv.ParseFloat(strconv.FormatFloat(value, 'f', 4, 64), 64)
		}
		f.FireSince = since
	}
	return f
}
