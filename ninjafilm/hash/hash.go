// Package hash supplies deterministic randomness for film sites and frames. The
// same seed, site, and frame produce the same draw, so development can be
// reproduced exactly.
//
// The hash itself is slayer's one hash, in maths/noise: a site of a frame
// is three indices, and this package only says so in the film's words. Its
// tests hold that hash to the film's recorded numbers.
package hash

import "github.com/janearc/slayer/maths/noise"

// Bits is the whole 64 bit hash for a seed and a site of a frame. It
// serves callers that want several numbers out of one site. The lattice
// corner of gradient noise takes its gradient from the low bits.
func Bits(seed uint64, x, y, frame int64) uint64 {
	return noise.Bits3(seed, x, y, frame)
}

// Three is a number from zero up to one for a seed and a site of a frame.
func Three(seed uint64, x, y, frame int64) float64 {
	return noise.Hash3(seed, x, y, frame)
}
