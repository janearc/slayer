package show

import "sort"

// Beat is the beat a moment is in: the last beat of the grid at or
// before it, and the score's meter, which says its bar. Before the
// first beat a moment is in none.
type Beat struct {
	index    int
	meter    int
	struck   float64
	since    float64
	measured bool
}

// BeatAt is the beat a moment is in: the grid's latest beat at or before
// it, so a moment between two beats is in the earlier. Before the grid
// it is in none; past the last beat it stays in the last.
func (of *Score) BeatAt(seconds float64) Beat {
	index := sort.SearchFloat64s(of.beats, seconds)
	if index < len(of.beats) && of.beats[index] == seconds {
		index++
	}
	index--
	if index < 0 {
		return Beat{index: -1, meter: of.meter}
	}
	return Beat{index: index, meter: of.meter, struck: of.beats[index],
		since: seconds - of.beats[index], measured: of.measured[index]}
}

// Index is the beat's place in the grid, from nought; -1 before it.
func (of Beat) Index() int { return of.index }

// On is whether the moment is on the grid at all: false before the
// first beat.
func (of Beat) On() bool { return of.index >= 0 }

// Bar is the bar the beat is in, from one; nought before the grid.
func (of Beat) Bar() int {
	if !of.On() {
		return 0
	}
	return of.index/of.meter + 1
}

// InBar is the beat's place in its bar, from one: one is the downbeat.
// Nought before the grid.
func (of Beat) InBar() int {
	if !of.On() {
		return 0
	}
	return of.index%of.meter + 1
}

// Measured is whether the beat was measured from the drums; false for
// one carried through a break at the median spacing.
func (of Beat) Measured() bool { return of.measured }

// Since is how long ago the beat struck, in seconds.
func (of Beat) Since() float64 { return of.since }
