package noise_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths/noise"
)

// The reference is maths/noise written out again here, outside the package,
// in its plainest form: each of a cell's eight corners hashed whole, and
// graded by a switch on the hash. The package is held to it bit for bit.
// Its numbers are splitmix64's and Perlin's, each defined by them.
const (
	firstShift   = 30                 // defined by splitmix64: the first right shift
	firstFactor  = 0xbf58476d1ce4e5b9 // defined by splitmix64: the first multiplier
	secondShift  = 27                 // defined by splitmix64: the second right shift
	secondFactor = 0x94d049bb133111eb // defined by splitmix64: the second multiplier
	lastShift    = 31                 // defined by splitmix64: the last right shift
	golden       = 0x9e3779b97f4a7c15 // defined by splitmix64: its golden gamma, the odd number it steps by
	quinticLead  = 6                  // defined by Perlin's quintic: the number on t to the fifth
	quinticDrop  = 15                 // defined by Perlin's quintic: the number taken off, on t to the fourth
	quinticTail  = 10                 // defined by Perlin's quintic: the number on t cubed
	cubeEdges    = 12                 // defined by a cube: its edges, the directions a corner may lean
	octaveStep   = 0x632be59bd9b4e019 // defined by the recorded numbers: the step between octaves' seeds
	octaveFall   = 0.5                // defined by the recorded numbers: each octave's share of the one before
	octaveRise   = 2.03               // defined by the recorded numbers: how many times finer each octave is
	nothing      = 0.0                // defined: no amount, what a sum starts at
	one          = 1.0                // defined: an amplitude of one, the first octave's
	spareBits    = 11                 // defined: the bits of a 64 bit hash past a 53 bit fraction
	fraction     = 1 << 53            // defined: how many numbers a 53 bit fraction holds
)

// the twelve edges, in Perlin's order, as the switch names them
const (
	xPlusY = iota
	minusXPlusY
	xMinusY
	minusXMinusY
	xPlusZ
	minusXPlusZ
	xMinusZ
	minusXMinusZ
	yPlusZ
	minusYPlusZ
	yMinusZ
	minusYMinusZ
)

// mix is splitmix64's finaliser, as the reference had it.
func mix(bits uint64) uint64 {
	bits ^= bits >> firstShift
	bits *= firstFactor
	bits ^= bits >> secondShift
	bits *= secondFactor
	return bits ^ bits>>lastShift
}

// bits3 is the whole hash of a seed and three indices, as the reference
// had it.
func bits3(seed uint64, first, second, third int64) uint64 {
	under := mix(uint64(third) + golden)
	columned := mix(uint64(second) + under)
	placed := mix(uint64(first) + columned)
	return mix(seed ^ placed)
}

// hashStepping is Hash with a step in place of the golden one: the index
// plus the step is mixed, the seed is stirred in, and the top 53 bits are
// the number. The reference Hash is hashStepping at golden.
func hashStepping(step, seed uint64, index int64) float64 {
	placed := mix(uint64(index) + step)
	return float64(mix(seed^placed)>>spareBits) / fraction
}

// fade is Perlin's quintic, as the reference had it.
func fade(along float64) float64 {
	return along * along * along * (along*(along*quinticLead-quinticDrop) +
		quinticTail)
}

// lerp is a straight line from one number to another, as the reference
// had it.
func lerp(along, from, to float64) float64 { return from + along*(to-from) }

// referenceGrad is grad written as a switch on the hash.
func referenceGrad(hashed uint64, x, y, z float64) float64 {
	switch hashed % cubeEdges {
	case xPlusY:
		return x + y
	case minusXPlusY:
		return -x + y
	case xMinusY:
		return x - y
	case minusXMinusY:
		return -x - y
	case xPlusZ:
		return x + z
	case minusXPlusZ:
		return -x + z
	case xMinusZ:
		return x - z
	case minusXMinusZ:
		return -x - z
	case yPlusZ:
		return y + z
	case minusYPlusZ:
		return -y + z
	case yMinusZ:
		return y - z
	}
	return -y - z
}

// turnedGrad is referenceGrad with the last edge's first sign turned
// round: y - z in place of -y - z. It is the wrong noise the sweep is
// shown, to see that it is caught.
func turnedGrad(hashed uint64, x, y, z float64) float64 {
	if hashed%cubeEdges == minusYMinusZ {
		return y - z
	}
	return referenceGrad(hashed, x, y, z)
}

// grader is a grad: the dot of an offset with the edge a hash picks.
type grader func(hashed uint64, x, y, z float64) float64

// gradientOver is Gradient over a grader: each of the eight corners hashed
// whole by bits3, and graded.
func gradientOver(grad grader, seed uint64, x, y, z float64) float64 {
	floorX, floorY, floorZ := math.Floor(x), math.Floor(y), math.Floor(z)
	ix, iy, iz := int64(floorX), int64(floorY), int64(floorZ)
	x, y, z = x-floorX, y-floorY, z-floorZ
	easedX, easedY, easedZ := fade(x), fade(y), fade(z)
	in := referenceCell{grad, seed, ix, iy, iz, x, y, z}
	nearLow := lerp(easedX, in.corner(nought, nought, nought), in.corner(unity, nought, nought))
	nearHigh := lerp(easedX, in.corner(nought, unity, nought), in.corner(unity, unity, nought))
	farLow := lerp(easedX, in.corner(nought, nought, unity), in.corner(unity, nought, unity))
	farHigh := lerp(easedX, in.corner(nought, unity, unity), in.corner(unity, unity, unity))
	return lerp(easedZ, lerp(easedY, nearLow, nearHigh), lerp(easedY, farLow, farHigh))
}

// referenceCell is a point in its cell as the reference graded it: the
// grader, the seed, the cell's lattice indices and the point's place in it.
type referenceCell struct {
	grad       grader
	seed       uint64
	ix, iy, iz int64
	x, y, z    float64
}

// corner is one corner's share: hashed whole by bits3, and graded.
func (in referenceCell) corner(dx, dy, dz int64) float64 {
	hashed := bits3(in.seed, in.ix+dx, in.iy+dy, in.iz+dz)
	return in.grad(hashed, in.x-float64(dx), in.y-float64(dy), in.z-float64(dz))
}

// referenceGradient is the reference's Gradient.
func referenceGradient(seed uint64, x, y, z float64) float64 {
	return gradientOver(referenceGrad, seed, x, y, z)
}

// turnedGradient is the reference's Gradient with one edge turned round.
func turnedGradient(seed uint64, x, y, z float64) float64 {
	return gradientOver(turnedGrad, seed, x, y, z)
}

// finite is whether a number is one: not NaN, and not infinite either
// way, as the reference had it.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, nought)
}

// referenceFractal is Fractal over referenceGradient.
func referenceFractal(seed uint64, x, y, z float64, octaves int) float64 {
	return fractalRising(octaveRise, seed, x, y, z, octaves)
}

// fractalRising is referenceFractal with each octave finer than the one
// before by rise in place of octaveRise: a fractal with a wrong constant,
// to show the recorded rows are moved by one.
func fractalRising(rise float64, seed uint64, x, y, z float64,
	octaves int) float64 {
	sum, amp, norm := nothing, one, nothing
	for octave := range octaves {
		if !finite(x) || !finite(y) || !finite(z) {
			break
		}
		field := seed + uint64(octave)*octaveStep
		sum += amp * referenceGradient(field, x, y, z)
		norm += amp
		amp *= octaveFall
		x, y, z = x*rise, y*rise, z*rise
	}
	if norm == nothing {
		return nothing
	}
	return sum / norm
}

// the kinds of point a sweep makes, by the point's number: one of its
// coordinates rounded to the lattice, or moved to a half, or made a
// signed zero; the other kinds are left as thrown
const (
	rounded = iota
	halved
	signedZero
)

// sweeping is a sweep set up from sweeps.csv and reaches.csv: its random
// numbers, how many points it asks, and the scales it throws them at.
type sweeping struct {
	random *rand.Rand
	points int
	scales []float64
	kinds  int
}

// sweepOf is the sweep a row of sweeps.csv names.
func sweepOf(test reporter, name string) sweeping {
	test.Helper()
	row := read(test, sweepsFile).by(test, sweepsFile, name)
	var scales []float64
	for _, reach := range read(test, reachesFile) {
		scales = append(scales, float(test, reach, "scale"))
	}
	first, second := seed(test, row, "first"), seed(test, row, "second")
	source := rand.NewPCG(first, second)
	return sweeping{
		random: rand.New(source),
		points: int(whole(test, row, "points")),
		scales: scales,
		kinds:  count(test, "point-kinds")}
}

// point is a seed and a place for the sweep's next point: places near the
// middle, far out, on the lattice, on halves, and a coordinate a signed
// zero.
func (sweep sweeping) point(index int) (uint64, place) {
	seed := sweep.random.Uint64()
	scale := sweep.scales[index%len(sweep.scales)]
	at := [axes]float64{sweep.random.NormFloat64() * scale,
		sweep.random.NormFloat64() * scale, sweep.random.NormFloat64() * scale}
	axis := index % len(at)
	switch index % sweep.kinds {
	case rounded:
		at[axis] = math.Round(at[axis])
	case halved:
		at[axis] = math.Round(at[axis]) + half
	case signedZero:
		at[axis] = math.Copysign(nought, -one)
	}
	return seed, place{at[nought], at[unity], at[pair]}
}

// apart is how many of the sweep's points a gradient differs from the
// reference at, to the bit.
func (sweep sweeping) apart(gradient field) int {
	found := nought
	for index := range sweep.points {
		seed, at := sweep.point(index)
		if !sameBits(gradient(seed, at.x, at.y, at.z),
			referenceGradient(seed, at.x, at.y, at.z)) {
			found++
		}
	}
	return found
}

// TestGradientIsTheReferenceToTheBit asks Gradient and the reference at
// two million random points: not one bit differs.
func TestGradientIsTheReferenceToTheBit(test *testing.T) {
	sweep := sweepOf(test, "gradient")
	if found := sweep.apart(noise.Gradient); found != nought {
		test.Fatalf("%d of %d points differ from the reference", found,
			sweep.points)
	}
}

// nudgedUp is Gradient moved up by one unit in the last place.
func nudgedUp(seed uint64, x, y, z float64) float64 {
	return math.Nextafter(noise.Gradient(seed, x, y, z), plusInf())
}

// TestTheSweepCatchesOneWrongEdgeAndOneUlp is the anti-test: with one
// edge's sign turned round, or a result moved by one unit in the last
// place, the sweep finds points that differ.
func TestTheSweepCatchesOneWrongEdgeAndOneUlp(test *testing.T) {
	if sweepOf(test, "gradient").apart(turnedGradient) == nought {
		test.Error("one edge turned round went unseen")
	}
	sweep := sweepOf(test, "gradient")
	if sweep.apart(nudgedUp) != sweep.points {
		test.Error("a result one unit in the last place up went unseen")
	}
	if !sameBits(nought, nought) || sameBits(nought, math.Copysign(nought, -one)) {
		test.Error("the bits of the zeros are not told apart")
	}
}

// TestFractalIsTheReferenceToTheBit asks Fractal of one to eight octaves
// at random points, and the reference: not one bit differs.
func TestFractalIsTheReferenceToTheBit(test *testing.T) {
	sweep, most := sweepOf(test, "fractal"), count(test, "most-octaves")
	for index := range sweep.points {
		seed, at := sweep.point(index)
		octaves := unity + index%most
		want := referenceFractal(seed, at.x, at.y, at.z, octaves)
		got := noise.Fractal(seed, at.x, at.y, at.z, octaves)
		if !sameBits(got, want) {
			test.Fatalf("seed %d at %v, %d octaves: %v, want %v", seed, at,
				octaves, got, want)
		}
	}
}

// threeSeeds are the seeds Fractals is asked at once: a run from the
// first, or on the odd points the first, a random one and the first
// with its last bit turned.
func threeSeeds(sweep sweeping, first uint64, index int) []uint64 {
	if index%pair == unity {
		return []uint64{first, sweep.random.Uint64(), first ^ unity}
	}
	return []uint64{first, first + unity, first + pair}
}

// octaveCounts are the octave counts of counts.csv.
func octaveCounts(test reporter) []int {
	test.Helper()
	var counts []int
	for _, row := range read(test, countsFile) {
		octaves := int(whole(test, row, "octaves"))
		counts = append(counts, octaves)
	}
	return counts
}

// TestFractalsAreFractalsToTheBit asks Fractals for three seeds at two
// hundred thousand random points, signed zeros and lattice points among
// them, from no octaves to past what a float holds: each result is the
// reference Fractal of its own seed, to the bit.
func TestFractalsAreFractalsToTheBit(test *testing.T) {
	sweep, counts := sweepOf(test, "fractals"), octaveCounts(test)
	into := make([]float64, count(test, "fractal-seeds"))
	for index := range sweep.points {
		first, at := sweep.point(index)
		seeds := threeSeeds(sweep, first, index)
		octaves := counts[index%len(counts)]
		noise.Fractals(seeds, at.x, at.y, at.z, octaves, into)
		for which, each := range seeds {
			want := referenceFractal(each, at.x, at.y, at.z, octaves)
			if !sameBits(into[which], want) {
				test.Fatalf("seed %d at %v, %d octaves: %v, want %v", each, at,
					octaves, into[which], want)
			}
		}
	}
}

// TestFractalsPastWhatAFloatHoldsAreFractals asks octaves past where a
// coordinate overflows: the octaves past it are left out, as Fractal
// leaves them, to the bit.
func TestFractalsPastWhatAFloatHoldsAreFractals(test *testing.T) {
	sweep, octaves := sweepOf(test, "overflow"), count(test, "past-a-float")
	into := make([]float64, count(test, "fractal-seeds"))
	for index := range sweep.points {
		first, at := sweep.point(index)
		seeds := []uint64{first, first + unity, first + pair}
		noise.Fractals(seeds, at.x, at.y, at.z, octaves, into)
		for which, each := range seeds {
			want := referenceFractal(each, at.x, at.y, at.z, octaves)
			if !sameBits(into[which], want) {
				test.Fatalf("seed %d at %v: %v, want %v", each, at,
					into[which], want)
			}
		}
	}
}

// TestFractalsTellSeedsApartAndLeaveTheRestAlone is the anti-test: the
// same seeds asked in the other order give their results in that order,
// so a result put in another seed's place is caught; and the places of
// into past the seeds are left as they were.
func TestFractalsTellSeedsApartAndLeaveTheRestAlone(test *testing.T) {
	first, kept := pickedSeed(test, "fractal-seed"), picked(test, "kept")
	row := read(test, recordedFile).by(test, recordedFile,
		"Fractal of 7 over five octaves")
	at := place{float(test, row, "a"), float(test, row, "b"),
		float(test, row, "c")}
	octaves := count(test, "fractal-octaves")
	seeds := []uint64{first, first + unity, first + pair}
	into := []float64{nought, nought, nought, kept}
	noise.Fractals(seeds, at.x, at.y, at.z, octaves, into)
	turned := make([]float64, len(seeds))
	noise.Fractals([]uint64{first + pair, first + unity, first}, at.x, at.y,
		at.z, octaves, turned)
	if !sameBits(into[nought], turned[pair]) || sameBits(into[nought], turned[nought]) {
		test.Error("the seeds' results are not in their seeds' places")
	}
	if into[len(seeds)] != kept {
		test.Errorf("a place past the seeds became %v", into[len(seeds)])
	}
	if !sameBits(into[nought], noise.Fractal(first, at.x, at.y, at.z, octaves)) {
		test.Error("Fractals is not Fractal for the first seed")
	}
	none := []float64{kept}
	noise.Fractals(nil, at.x, at.y, at.z, octaves, none)
	if none[nought] != kept {
		test.Error("no seeds wrote a result")
	}
}
