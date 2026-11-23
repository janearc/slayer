// Package noise supplies deterministic hashes and smooth seeded noise. The
// values look irregular but are reproducible without mutable random state.
//
// Hash, Hash2, Hash3 and Bits3 are whole-number work and are the same on
// every machine. Gradient, Fractal and Fractals are not exact to the last
// bit across machines: Go may fuse a multiply and an add into one rounding,
// as it does on arm64 and does not on amd64 at the default GOAMD64, and a
// fused result can differ from an unfused one in its last bit. The tests
// that hold recorded bits hold arm64's.
package noise

import "math"

// splitmix64's finaliser is two rounds of shift, xor and multiply, and a
// last shift. Steele, Lea and Flood defined its shifts and factors, and
// each is kept here as they defined it.
const (
	firstShift   = 30                 // defined by splitmix64: the first right shift
	firstFactor  = 0xbf58476d1ce4e5b9 // defined by splitmix64: the first multiplier
	secondShift  = 27                 // defined by splitmix64: the second right shift
	secondFactor = 0x94d049bb133111eb // defined by splitmix64: the second multiplier
	lastShift    = 31                 // defined by splitmix64: the last right shift
)

// Mix is splitmix64's finaliser: every bit of the input reaches every bit
// of the output. A package that wants a word stirred, to seed a stream or
// to place a thing, asks here and does not write the finaliser out again.
func Mix(bits uint64) uint64 {
	bits ^= bits >> firstShift
	bits *= firstFactor
	bits ^= bits >> secondShift
	bits *= secondFactor
	return bits ^ bits>>lastShift
}

// a float64 holds 53 bits of fraction and a hash is 64 bits wide, so the
// hash's top 53 bits are its number from zero up to one
const (
	spareBits = 11      // defined: the bits of a 64 bit hash past a 53 bit fraction, 64 less 53
	fraction  = 1 << 53 // defined: 2 to the 53, how many numbers a 53 bit fraction holds
)

// Unit is the top 53 bits of a hash as a number from zero up to one: how
// a stirred word becomes a share, here and for any package that stirs
// its own with Mix.
func Unit(bits uint64) float64 { return float64(bits>>spareBits) / fraction }

// Stirred is a number from zero up to one for a word: the word stirred by
// Mix, and the top 53 bits of that as Unit has them. A package that wants a
// share from a word, to place a thing or to draw a sample, asks here.
func Stirred(bits uint64) float64 { return Unit(Mix(bits)) }

// Golden is the odd constant splitmix64 steps by, defined by splitmix64 as
// its golden gamma. Added to an index before it is mixed, it keeps index
// zero from mixing to zero, which Mix does with zero. A package that steps
// or spreads its own word by this number asks for it here.
const Golden = 0x9e3779b97f4a7c15

// Hash is a number from zero up to one for a seed and an index. The same pair
// gives the same number forever.
func Hash(seed uint64, index int64) float64 {
	placed := Mix(uint64(index) + Golden)
	return Stirred(seed ^ placed)
}

// Hash2 is a number from zero up to one for a seed and a pair of indices,
// as Hash is for one. It is its own field: Hash2 of a pair is not Hash
// of either index.
func Hash2(seed uint64, first, second int64) float64 {
	columned := Mix(uint64(second) + Golden)
	placed := Mix(uint64(first) + columned)
	return Stirred(seed ^ placed)
}

// Hash3 is a number from zero up to one for a seed and three indices.
func Hash3(seed uint64, first, second, third int64) float64 {
	return Unit(Bits3(seed, first, second, third))
}

// Bits3 is the whole 64 bit hash of a seed and three indices, for a
// caller that wants several numbers out of one: a lattice corner's
// gradient takes its low bits, and Hash3 is its top 53 as a fraction.
func Bits3(seed uint64, first, second, third int64) uint64 {
	under := Mix(uint64(third) + Golden)
	columned := Mix(uint64(second) + under)
	placed := Mix(uint64(first) + columned)
	return Mix(seed ^ placed)
}

// Gradient is three-dimensional gradient noise at a point: zero on every
// integer lattice point, smooth between them, and seldom past one either way:
// it can be, a little, as 1.033 at seed 41885775 and (0.35, 0.5, 0.475). over a
// million points it was never past 0.98. The most it can be is two, by the
// arithmetic; how near that any point comes is not proved, so a caller that
// needs a hard bound clamps.
//
// a coordinate is a number within 2^52 of zero. past that a float has no
// fraction, so the noise is the same everywhere; past 2^63 the lattice index is
// the largest or the smallest whole number an int64 holds. Go leaves that
// conversion to the machine, and arm64 holds it at the nearer end. NaN or an
// infinity gives NaN.
//
// Perlin's improved noise, with the gradient at each corner taken from the hash
// rather than a shuffled table. A table is 256 entries of one fixed shuffle;
// the hash gives every seed its own field, which is how two planets differ.
//
// The last bit of the value is the machine's, as the package doc says: the
// quintic and the blends are multiply-adds that arm64 fuses.
func Gradient(seed uint64, x, y, z float64) float64 {
	at := cellAt(x, y, z)
	return at.gradient(seed)
}

// cell is a point in its lattice cell, worked out once for every seed
// asked there: its place in the cell, the fades of it, and each
// corner's hash before the seed is mixed in, by its step along x, y, z.
type cell struct {
	x, y, z                float64
	easedX, easedY, easedZ float64
	placed                 [cellEnds][cellEnds][cellEnds]uint64
}

// cellAt is the cell a point is in, and where in it.
func cellAt(x, y, z float64) cell {
	floorX, floorY, floorZ := math.Floor(x), math.Floor(y), math.Floor(z)
	ix, iy, iz := int64(floorX), int64(floorY), int64(floorZ)
	at := cell{x: x - floorX, y: y - floorY, z: z - floorZ}
	at.easedX, at.easedY, at.easedZ = fade(at.x), fade(at.y), fade(at.z)
	inner := corners(iy, iz)
	for dx := range int64(cellEnds) {
		for dy := range cellEnds {
			for dz := range cellEnds {
				at.placed[dx][dy][dz] = Mix(uint64(ix+dx) + inner[dy][dz])
			}
		}
	}
	return at
}

// the two ends of a cell along an axis, as a corner's step from the
// cell's low corner
const (
	low  = 0 // defined: the low corner's step along an axis, none
	high = 1 // defined: the high corner's step along an axis, one lattice spacing
)

// gradient is Gradient's noise for a seed at the cell's point: each
// corner's hash finished with the seed, as bits3 finishes it, graded by
// the offset from that corner to the point, and the eight shares blended
// across the cell, along x, then y, then z. A corner's name is its steps
// along x, y and z.
//
// The corners are graded here and not in a method of their own: a
// method with grad and mix inside it is past what the compiler inlines,
// and eight calls a point would cost most of what working the cell once
// saves.
func (at *cell) gradient(seed uint64) float64 {
	nearX, farX := at.x-low, at.x-high
	nearY, farY := at.y-low, at.y-high
	nearZ, farZ := at.z-low, at.z-high
	corner000 := grad(Mix(seed^at.placed[low][low][low]), nearX, nearY, nearZ)
	corner100 := grad(Mix(seed^at.placed[high][low][low]), farX, nearY, nearZ)
	corner010 := grad(Mix(seed^at.placed[low][high][low]), nearX, farY, nearZ)
	corner110 := grad(Mix(seed^at.placed[high][high][low]), farX, farY, nearZ)
	corner001 := grad(Mix(seed^at.placed[low][low][high]), nearX, nearY, farZ)
	corner101 := grad(Mix(seed^at.placed[high][low][high]), farX, nearY, farZ)
	corner011 := grad(Mix(seed^at.placed[low][high][high]), nearX, farY, farZ)
	corner111 := grad(Mix(seed^at.placed[high][high][high]), farX, farY, farZ)
	nearLow := lerp(at.easedX, corner000, corner100)
	nearHigh := lerp(at.easedX, corner010, corner110)
	farLow := lerp(at.easedX, corner001, corner101)
	farHigh := lerp(at.easedX, corner011, corner111)
	near := lerp(at.easedY, nearLow, nearHigh)
	far := lerp(at.easedY, farLow, farHigh)
	return lerp(at.easedZ, near, far)
}

// corners is the part of bits3 that the four columns of a lattice cell
// share, by the corner's step along y and along z: the hash of the second
// and third indices, which every corner's first is then added to. Worked
// once a cell, it is the same number bits3 works for each of the eight
// corners.
func corners(second, third int64) [cellEnds][cellEnds]uint64 {
	var inner [cellEnds][cellEnds]uint64
	for dz := range int64(cellEnds) {
		under := Mix(uint64(third+dz) + Golden)
		for dy := range int64(cellEnds) {
			inner[dy][dz] = Mix(uint64(second+dy) + under)
		}
	}
	return inner
}

// cellEnds is the two ends of a lattice cell along an axis, its low
// corner and its high, defined by what a cell is.
const cellEnds = 2

// eitherWay is the sign math.IsInf takes to ask for an infinity of either
// sign, defined by math.IsInf as nought.
const eitherWay = 0

// finite is whether a number is one: not NaN, and not infinite either way.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, eitherWay)
}

// Perlin's quintic is 6t^5 - 15t^4 + 10t^3, written as t cubed times
// (t times (t times 6 minus 15) plus 10). Perlin defined its three numbers.
const (
	quinticLead = 6  // defined by the quintic: the number on t to the fifth
	quinticDrop = 15 // defined by the quintic: the number taken off, on t to the fourth
	quinticTail = 10 // defined by the quintic: the number on t cubed
)

// fade is Perlin's quintic, whose first two derivatives are zero at both ends,
// so the lattice does not show as creases.
func fade(along float64) float64 {
	return along * along * along * (along*(along*quinticLead-quinticDrop) +
		quinticTail)
}

// lerp is a straight line from one number to another, along its length.
func lerp(along, from, to float64) float64 { return from + along*(to-from) }

// axes is how many numbers an offset has, defined: x, y and z.
const axes = 3

// grad is the dot product of an offset with one of the twelve edge
// directions of a cube, chosen by the low bits of a hash.
//
// The edge is looked up, not switched on: the hash makes the choice
// unpredictable, and a mispredicted branch costs most of the noise's time.
// Each term is the offset's number times one or minus one, which is
// exact, so the sum is the one the switch made, to the bit.
func grad(hashed uint64, x, y, z float64) float64 {
	along := edges[hashed%cubeEdges]
	offset := [axes]float64{x, y, z}
	return along.firstSign*offset[along.first] +
		along.secondSign*offset[along.second]
}

// cubeEdges is how many edges a cube has, defined by what a cube is.
const cubeEdges = 12

// edge is one edge direction of a cube as grad reads it: which two of an
// offset's numbers it adds, x, y or z by place, and the sign of each.
type edge struct {
	first, second         int
	firstSign, secondSign float64
}

// the places of an offset's numbers, and the signs an edge gives them
const (
	alongX = 0    // defined: x is the first of an offset's three numbers
	alongY = 1    // defined: y is the second
	alongZ = 2    // defined: z is the third
	plus   = 1.0  // defined: the sign that keeps a number as it is
	minus  = -1.0 // defined: the sign that turns a number round
)

// edges are the twelve, in the order of Perlin's improved noise: x+y,
// -x+y, x-y, -x-y, then x and z, then y and z, the same way round.
var edges = [cubeEdges]edge{
	{alongX, alongY, plus, plus}, {alongX, alongY, minus, plus},
	{alongX, alongY, plus, minus}, {alongX, alongY, minus, minus},
	{alongX, alongZ, plus, plus}, {alongX, alongZ, minus, plus},
	{alongX, alongZ, plus, minus}, {alongX, alongZ, minus, minus},
	{alongY, alongZ, plus, plus}, {alongY, alongZ, minus, plus},
	{alongY, alongZ, plus, minus}, {alongY, alongZ, minus, minus},
}

// what a fractal's sum starts from
const (
	nothing = 0.0 // defined: no amount, what a sum starts at
	whole   = 1.0 // defined: an amplitude of one, the first octave's
)

// Fractal is gradient noise summed over octaves, each about twice the
// frequency and half the amplitude of the one before. It is what makes a
// coastline ragged at every scale rather than round at one.
//
// the sum is divided by its octaves' amplitudes added up, so it is never
// wider than one octave, which is Gradient and can pass one a little.
//
// the octaves seldom peak together, so it is narrower the more there are:
// over a million points, 0.98 at most for one octave, 0.83 for two, and
// 0.65 for five or eight.
//
// each octave's field is the seed plus a step times the octave, so two
// seeds a step apart share fields an octave apart. an octave whose
// position is past what a float holds, which a thousand or so do, is not
// added, nor any after it; nor is an octave whose share has fallen to
// nothing, which at the origin, where no position grows, is what ends
// the sum. no octaves, or fewer, is nought.
//
// a point with a coordinate that is NaN or infinite is no place, and the
// fractal there is NaN, as Gradient is, whatever the octaves: a caller
// that asks with no number is told so, and not handed a nought that looks
// like a lattice point. a finite coordinate that a late octave scales
// past what a float holds is another thing: that octave and those after
// it are not added, and the sum so far stands.
func Fractal(seed uint64, x, y, z float64, octaves int) float64 {
	if !finite(x) || !finite(y) || !finite(z) {
		return math.NaN()
	}
	sum, amp, norm := nothing, whole, nothing
	for octave := range octaves {
		if amp == nothing || !finite(x) || !finite(y) || !finite(z) {
			break
		}
		field := seed + uint64(octave)*octaveStep
		sum += amp * Gradient(field, x, y, z)
		norm += amp
		amp *= octaveFall
		x, y, z = x*octaveRise, y*octaveRise, z*octaveRise
	}
	if norm == nothing {
		return nothing
	}
	return sum / norm
}

// the octaves of a fractal. All three were picked, not derived,
// and the reference numbers in testdata/recorded.csv hold them to the
// bit: every seeded world slayer draws is made of them, so they are
// defined by those numbers, and a change is a new world.
const (
	// octaveStep is the step between one octave's seed and the next's,
	// an odd 64 bit number. picked, and defined by the recorded numbers.
	octaveStep = 0x632be59bd9b4e019
	// octaveFall is each octave's amplitude as a share of the one before.
	// picked as a half, and defined by the recorded numbers.
	octaveFall = 0.5
	// octaveRise is how many times finer each octave is than the one
	// before: 2.03 and not 2, so the octaves' lattices do not line up.
	// picked, and defined by the recorded numbers.
	octaveRise = 2.03
)

// Fractals is Fractal at one point for several seeds: into[i] is
// Fractal(seeds[i], x, y, z, octaves), to the bit. Each octave's cell,
// its fades and the seed-free part of its corners' hashes are worked
// once for all the seeds, which is most of a fractal's work beside the
// grading.
//
// into must be at least as long as seeds, or Fractals panics; what is in
// it past the seeds is left as it was. at a point that is no place, with
// a coordinate that is NaN or infinite, every seed's is NaN, as Fractal's
// is.
func Fractals(seeds []uint64, x, y, z float64, octaves int,
	into []float64) {
	into = into[:len(seeds):len(into)]
	if !finite(x) || !finite(y) || !finite(z) {
		for index := range into {
			into[index] = math.NaN()
		}
		return
	}
	clear(into)
	amp, norm := whole, nothing
	for octave := range octaves {
		if amp == nothing || !finite(x) || !finite(y) || !finite(z) {
			break
		}
		at := cellAt(x, y, z)
		field := uint64(octave) * octaveStep
		for index, seed := range seeds {
			into[index] += amp * at.gradient(seed+field)
		}
		norm += amp
		amp *= octaveFall
		x, y, z = x*octaveRise, y*octaveRise, z*octaveRise
	}
	for index := range into {
		if norm == nothing {
			into[index] = nothing
			continue
		}
		into[index] /= norm
	}
}
