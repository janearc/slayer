package main

import (
	"fmt"
	"math"
)

// sixteenths is how many sixteenth notes a beat holds: the beat is a
// quarter note in every meter a cheeptune score writes.
const sixteenths = 4

// nudge is added to a count of sixteenths before it is floored, so a
// moment that falls on a sixteenth, give or take rounding, reads as that
// sixteenth and not the one before. A millionth of a sixteenth.
const nudge = 1e-6

// position is the bar, the beat and the sixteenth that a count of beats
// from the start falls in, each counted from one. A moment before the
// start reads as the first.
func position(beats float64, meter int) (bar, beat, sixteenth int) {
	count := max(0, int(math.Floor(beats*sixteenths+nudge)))
	perBar := meter * sixteenths
	return count/perBar + 1, count%perBar/sixteenths + 1,
		count%sixteenths + 1
}

// readout is the position written as she quotes it: 8.2.3.
func readout(beats float64, meter int) string {
	bar, beat, sixteenth := position(beats, meter)
	return fmt.Sprintf("%d.%d.%d", bar, beat, sixteenth)
}
