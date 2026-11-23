package cities

import "math/rand/v2"

// The constants of mixing a seed and a member number into a stream's two
// halves: the golden ratio's 64 bits, and the two multipliers and three
// shifts of the splitmix64 generator of Steele, Lea and Flood.
const (
	// mixStep is 2^64 over the golden ratio.
	mixStep = 0x9e3779b97f4a7c15 // defined
	// mixFirst is splitmix64's first multiplier.
	mixFirst = 0xbf58476d1ce4e5b9 // defined
	// mixSecond is splitmix64's second multiplier.
	mixSecond = 0x94d049bb133111eb // defined
	// mixFirstShift, mixSecondShift and mixLastShift are its three shifts.
	mixFirstShift  = 30 // defined
	mixSecondShift = 27 // defined
	mixLastShift   = 31 // defined
)

// finished is a number scattered over all 64 bits, the same for the same
// number: the finishing step of splitmix64.
func finished(number uint64) uint64 {
	number = (number ^ number>>mixFirstShift) * mixFirst
	number = (number ^ number>>mixSecondShift) * mixSecond
	return number ^ number>>mixLastShift
}

// MemberStream is the stream of random draws of one member of a stand, from
// the stand's seed and the member's number alone, counting from nought. The
// same seed and number give the same draws in the same order, and nothing
// placed before the member changes them.
func MemberStream(seed uint64, member int) *rand.Rand {
	first := finished(seed + uint64(member+1)*mixStep)
	return rand.New(rand.NewPCG(first, finished(first+mixStep)))
}
