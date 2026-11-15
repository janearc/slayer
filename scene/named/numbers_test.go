package named_test

import "time"

// what the tests count in, each defined
const (
	none = 0 // defined: no count, no time
	once = 1 // defined: one of a thing
)

// the values the tests give, each picked by the author of this package,
// 2026-11-12: a few of a thing, a fraction, and a spell of time
const (
	few      = 3
	fraction = 2.5
	aSpell   = 20 * time.Second
	spelled  = "20s"
)
