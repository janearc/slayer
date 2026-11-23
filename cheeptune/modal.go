package cheeptune

import (
	"math"
	"sort"
	"strings"
)

// The modal engine: a struck object as a bank of resonators. Anything
// stiff that is hit rings at a handful of frequencies of its own, its
// modes, each dying at its own pace; the modes of a bar, a plate or a
// tube are not whole multiples of the lowest, and that is the difference
// between a bell and a string.
//
// Each mode is a two-pole resonator. All of them are driven by one short
// pulse, the blow: a hard blow is a short pulse, which has energy far up
// and rings the high modes; a soft blow is a long pulse and rings only
// the low ones.

// modal is a modal channel's sound. Set names a shipped object (modes.go)
// or modes lists the object's own. Hardness is the blow, nought (a soft
// mallet) to one (metal on metal). Decay is the seconds the lowest mode
// takes to fall sixty decibels; a named set's higher modes die faster.
//
// Release, when it is given, is the seconds the sound takes to die once
// the note is let go, as if a hand were laid on it; without it the object
// rings on freely.
type modal struct {
	Set      string  `json:"set"`
	Modes    []mode  `json:"modes"`
	Hardness float64 `json:"hardness"`
	Decay    float64 `json:"decay"`
	Release  float64 `json:"release"`
}

// mode is one resonance: its frequency as a ratio to the note, the
// seconds it takes to fall sixty decibels, and its level.
type mode struct {
	Ratio float64 `json:"ratio"`
	Decay float64 `json:"decay"`
	Level float64 `json:"level"`
}

// The ranges of a modal channel. Picked.
const (
	modesMost      = 12
	modeRatioLeast = 0.1
	modeRatioMost  = 40
	modeTimeLeast  = 0.01
	// blowLongest and blowShortest are how long the blow lasts at
	// hardness nought and one, in seconds.
	blowLongest  = 0.004
	blowShortest = 0.00005
	// modeCeiling is the highest a mode may ring, as a share of the
	// rate; a mode above it is left out, not folded back.
	modeCeiling = 0.45
)

// init puts the modal engine in the studio's table.
func init() {
	register("modal", engineRow{
		shaped: "each mode has its own decay",
		check:  checkModal, build: buildModal})
}

// setNames lists the shipped mode sets, for an error.
func setNames() string {
	names := make([]string, 0, len(modeSets))
	for name := range modeSets {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// checkModal is a modal channel's own rules.
func checkModal(c *channel, q *quarrel) {
	m := c.Modal
	_, named := modeSets[m.Set]
	switch {
	case m.Set != "" && len(m.Modes) > 0:
		q.fail("modal has a set and its own modes; give one")
	case m.Set != "" && !named:
		q.fail("modal set %q; sets are %s", m.Set, setNames())
	case m.Set == "" && len(m.Modes) == 0:
		q.fail("modal needs a set (%s) or its own modes", setNames())
	case len(m.Modes) > modesMost:
		q.fail("modal has %d modes and the most is %d", len(m.Modes),
			modesMost)
	}
	if named {
		q.within("modal decay", m.Decay, modeTimeLeast, envelopeMost)
	} else if m.Decay != 0 {
		q.fail("modal decay is for a named set; each of your own " +
			"modes carries its decay")
	}
	for i, mode := range m.Modes {
		name := "modal mode " + string(rune('1'+i))
		q.within(name+" ratio", mode.Ratio, modeRatioLeast, modeRatioMost)
		q.within(name+" decay", mode.Decay, modeTimeLeast, envelopeMost)
		q.within(name+" level", mode.Level, 0, 1)
	}
	q.within("modal hardness", m.Hardness, 0, 1)
	q.within("modal release", m.Release, 0, envelopeMost)
}

// modalEngine is a modal channel ready to sound.
type modalEngine struct {
	modal *modal
	modes []mode
}

// buildModal is the engine for a checked modal channel: the channel's own
// modes, or a named set's with the times its loss gives them.
func buildModal(c *channel) engine {
	e := &modalEngine{modal: c.Modal, modes: c.Modal.Modes}
	if set, ok := modeSets[c.Modal.Set]; ok {
		e.modes = setModes(set, c.Modal.Decay)
	}
	return e
}

// setModes is a named set's modes with the times its loss gives them,
// the lowest ringing for the decay.
func setModes(set modeSet, decay float64) []mode {
	var modes []mode
	for i, ratio := range set.ratios {
		rings := decay / math.Pow(ratio/set.ratios[0], set.loss)
		modes = append(modes, mode{Ratio: ratio,
			Decay: math.Max(rings, modeTimeLeast),
			Level: set.levels[i]})
	}
	return modes
}

// longest is the longest any mode rings, in seconds.
func (e *modalEngine) longest() float64 {
	longest := 0.0
	for _, mode := range e.modes {
		longest = math.Max(longest, mode.Decay)
	}
	return longest
}

// ring is the release when a hand damps the note, and otherwise whatever
// of the longest mode's time is left when the note is let go.
func (e *modalEngine) ring(st strike) float64 {
	if e.modal.Release > 0 {
		return e.modal.Release
	}
	return math.Max(0, e.longest()-st.gate)
}

// resonator is one mode running.
type resonator struct {
	turn, level  float64
	rings        float64 // seconds, as its mode says
	feed, a1, a2 float64
	y1, y2       float64
}

// decayTo sets the resonator to fall sixty decibels in so many seconds.
func (r *resonator) decayTo(seconds float64) {
	radius := math.Exp(-sixtyDecibels / (seconds * studioRate))
	r.a1 = 2 * radius * math.Cos(r.turn)
	r.a2 = -radius * radius
}

// step takes one sample of the blow and gives the mode's sound.
func (r *resonator) step(x float64) float64 {
	y := r.feed*x + r.a1*r.y1 + r.a2*r.y2
	r.y2, r.y1 = r.y1, y
	return y * r.level
}

// bank is the modes that fit under the ceiling for a note, their levels
// scaled to sum to one.
func (e *modalEngine) bank(hertz float64) []resonator {
	var out []resonator
	total := 0.0
	for _, mode := range e.modes {
		at := hertz * mode.Ratio
		if at > modeCeiling*studioRate {
			continue
		}
		turn := 2 * math.Pi * at / studioRate
		r := resonator{turn: turn, level: mode.Level,
			rings: mode.Decay, feed: math.Sin(turn)}
		r.decayTo(mode.Decay)
		out = append(out, r)
		total += mode.Level
	}
	for i := range out {
		if total > 0 {
			out[i].level /= total
		}
	}
	return out
}

// blow is the pulse that strikes: a raised cosine as long as the hardness
// says, at least two samples, summing to one.
func (e *modalEngine) blow() []float64 {
	seconds := blowLongest *
		math.Pow(blowShortest/blowLongest, e.modal.Hardness)
	n := max(2, int(math.Round(seconds*studioRate)))
	out := make([]float64, n)
	for i := range out {
		out[i] = (1 - math.Cos(2*math.Pi*(float64(i)+0.5)/float64(n))) /
			float64(n)
	}
	return out
}

// sound renders one note.
func (e *modalEngine) sound(st strike, out []float64) {
	bank := e.bank(st.hz)
	blow := e.blow()
	letGo := -1
	if e.modal.Release > 0 {
		letGo = int(st.gate * studioRate)
	}
	for i := range out {
		x := 0.0
		if i < len(blow) {
			x = blow[i]
		}
		if i == letGo {
			for k := range bank {
				seconds := math.Min(bank[k].rings, e.modal.Release)
				bank[k].decayTo(seconds)
			}
		}
		sum := 0.0
		for k := range bank {
			sum += bank[k].step(x)
		}
		out[i] = sum
	}
}
