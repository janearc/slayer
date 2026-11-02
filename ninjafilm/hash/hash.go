// Package hash supplies deterministic randomness for film sites and frames. The
// same seed, site, and frame produce the same draw, so development can be
// reproduced exactly.
//
// It uses SplitMix64’s finalizer, matching Veronica’s audited Lamb noise
// implementation retained in Slayer’s maths/noise at 3b6549b.
package hash

// The finaliser is two steps of shift, xor and multiply. Then it shifts
// once more. Its shifts and factors are next. Splitmix64 defines each.

// firstShift is the first right shift. It is defined by splitmix64.
const firstShift = 30

// firstFactor is the first multiplier. It is defined by splitmix64.
const firstFactor = 0xbf58476d1ce4e5b9

// secondShift is the second right shift. It is defined by splitmix64.
const secondShift = 27

// secondFactor is the second multiplier. It is defined by splitmix64.
const secondFactor = 0x94d049bb133111eb

// lastShift is the last right shift. It is defined by splitmix64.
const lastShift = 31

// mix is splitmix64's finaliser. Every bit of the input reaches every
// bit of the output.
func mix(bits uint64) uint64 {
	bits ^= bits >> firstShift
	bits *= firstFactor
	bits ^= bits >> secondShift
	bits *= secondFactor
	return bits ^ bits>>lastShift
}

// Float64s hold 53 bits of fraction. Hashes are 64 bits wide.

// spareBits is the 11 bits of a 64 bit hash that a 53 bit fraction
// leaves over. It is defined by the widths of the two.
const spareBits = 11

// fraction is 2 to the 53, how many numbers a 53 bit fraction holds. It
// is defined by that width.
const fraction = 1 << 53

// unit is the top 53 bits of a hash as a number from zero up to one.
func unit(bits uint64) float64 { return float64(bits>>spareBits) / fraction }

// golden is the odd constant splitmix64 steps by. Added to an index before it
// is mixed, it keeps index zero from mixing to zero. It is defined by
// splitmix64 as its golden gamma.
const golden = 0x9e3779b97f4a7c15

// Bits is the whole 64 bit hash for a seed and a site of a frame. It
// serves callers that want several numbers out of one site. The lattice
// corner of gradient noise takes its gradient from the low bits.
func Bits(seed uint64, x, y, frame int64) uint64 {
	framed := mix(uint64(frame) + golden)
	rowed := mix(uint64(y) + framed)
	placed := mix(uint64(x) + rowed)
	return mix(seed ^ placed)
}

// Three is a number from zero up to one for a seed and a site of a frame.
func Three(seed uint64, x, y, frame int64) float64 {
	return unit(Bits(seed, x, y, frame))
}
