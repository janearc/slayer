package cheeptune

import "math"

// An engine is what makes a studio channel's sound: it renders one strike
// (a note or a hit, whole) into samples.
type engine interface {
	// ring is how many seconds a strike sounds after it is let go.
	ring(st strike) float64
	// sound writes the strike into out, out[0] being its first sample,
	// and nothing outside minus one to one by more than its design.
	sound(st strike, out []float64)
}

// engineRow is one engine in the table: its rules and how it is made.
type engineRow struct {
	percussive bool
	envelope   bool   // whether the channel's envelope shapes it
	vibrato    bool   // whether its pitch can move: vibrato and slides
	shaped     string // what to say when an envelope is refused
	check      func(c *channel, q *quarrel)
	build      func(c *channel) engine
}

// engines is every engine the studio has, by channel kind. A new engine is
// a file and a row here.
var engines = map[string]engineRow{}

// register adds an engine's row; each engine file calls it from init.
func register(kind string, row engineRow) { engines[kind] = row }

// sineSteps is the size of the sine table. Picked: with a straight line
// between entries, 4096 steps is wrong by under one part in a million.
const sineSteps = 4096

// sineTable is one cycle of a sine and its first entry again.
var sineTable = func() (t [sineSteps + 1]float64) {
	for i := range t {
		t[i] = math.Sin(2 * math.Pi * float64(i) / sineSteps)
	}
	return t
}()

// sine is the sine of a phase counted in cycles, any sign.
func sine(cycles float64) float64 {
	cycles -= math.Floor(cycles)
	at := cycles * sineSteps
	i := int(at)
	f := at - float64(i)
	return sineTable[i] + (sineTable[i+1]-sineTable[i])*f
}

// noise is a small repeatable source of white noise, minus one to one.
// A render is the same every time because every strike seeds its own.
type noise struct{ state uint64 }

// newNoise is a source for a seed; nought is moved off nought.
func newNoise(seed uint64) *noise {
	return &noise{state: seed*0x9e3779b97f4a7c15 + 0x2545f4914f6cdd1d}
}

// next is the next sample: a xorshift step, scaled.
func (n *noise) next() float64 {
	x := n.state
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	n.state = x
	return float64(int64(x>>11))/float64(1<<52) - 1
}
