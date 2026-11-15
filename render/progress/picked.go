package progress

import "time"

// The counts the package is made of, each defined.
const (
	none   = 0 // defined: no rows, no frames, no time
	fewest = 1 // defined: the least a side, a count or a job is
)

// The numbers the package picked. Each is picked by the author of this
// slice, 2026-11-12, open to cutting.
const (
	// MinElapsed is how long a job must have run before the time left is
	// offered. Under a second the pace is the start-up's, and a guess
	// then would swing wildly.
	MinElapsed = time.Second

	// MinRows is how many rows must be finished before the time left is
	// offered. One band is too few to say a pace.
	MinRows = 8

	// Interval is the least time between two lines written to a pipe,
	// the line that ends a frame aside. Often enough to see a job live,
	// rarely enough to read a log.
	Interval = 2 * time.Second
)

// The words of a duration, each defined.
const (
	secondsInMinute = 60  // defined
	percentOf       = 100 // defined: a share of one, as a percentage
)
