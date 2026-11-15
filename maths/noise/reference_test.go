package noise

import (
	"math"
	"math/rand/v2"
	"testing"
)

// referenceGrad is grad as it was written before 12 November 2026, a
// switch on the hash, kept so the table that replaced it is held to it.
func referenceGrad(h uint64, x, y, z float64) float64 {
	switch h % 12 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x + z
	case 5:
		return -x + z
	case 6:
		return x - z
	case 7:
		return -x - z
	case 8:
		return y + z
	case 9:
		return -y + z
	case 10:
		return y - z
	}
	return -y - z
}

// referenceGradient is Gradient as it was before 12 November 2026: each
// of the eight corners hashed whole by bits3, and graded by the switch.
func referenceGradient(seed uint64, x, y, z float64) float64 {
	fx, fy, fz := math.Floor(x), math.Floor(y), math.Floor(z)
	ix, iy, iz := int64(fx), int64(fy), int64(fz)
	x, y, z = x-fx, y-fy, z-fz
	u, v, w := fade(x), fade(y), fade(z)
	g := func(dx, dy, dz int64) float64 {
		h := bits3(seed, ix+dx, iy+dy, iz+dz)
		return referenceGrad(h, x-float64(dx), y-float64(dy),
			z-float64(dz))
	}
	return lerp(w,
		lerp(v, lerp(u, g(0, 0, 0), g(1, 0, 0)),
			lerp(u, g(0, 1, 0), g(1, 1, 0))),
		lerp(v, lerp(u, g(0, 0, 1), g(1, 0, 1)),
			lerp(u, g(0, 1, 1), g(1, 1, 1))))
}

// referenceFractal is Fractal over referenceGradient.
func referenceFractal(seed uint64, x, y, z float64, octaves int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for o := range octaves {
		if !finite(x) || !finite(y) || !finite(z) {
			break
		}
		s := seed + uint64(o)*0x632be59bd9b4e019
		sum += amp * referenceGradient(s, x, y, z)
		norm += amp
		amp *= 0.5
		x, y, z = x*2.03, y*2.03, z*2.03
	}
	if norm == 0 {
		return 0
	}
	return sum / norm
}

// sameBits is whether two numbers are one to the bit: a negative zero is
// not a zero.
func sameBits(one, other float64) bool {
	return math.Float64bits(one) == math.Float64bits(other)
}

// offsets are the numbers grad is checked at: both zeros, the lattice's
// steps, halves, and numbers that are not round.
var offsets = []float64{0, math.Copysign(0, -1), 1, -1, 0.5, -0.5, 0.25,
	-0.75, 1e-300, -1e-300, 0.1234567890123, -0.98765432109}

// TestGradIsTheSwitchItReplaced checks every edge, twice round, at every
// three offsets from the list: the table's number is the switch's, to
// the bit, signed zeros among them.
func TestGradIsTheSwitchItReplaced(t *testing.T) {
	for h := range uint64(2 * cubeEdges) {
		for _, x := range offsets {
			for _, y := range offsets {
				for _, z := range offsets {
					want := referenceGrad(h, x, y, z)
					if got := grad(h, x, y, z); !sameBits(got, want) {
						t.Fatalf("edge %d at %v, %v, %v: %v, want %v", h,
							x, y, z, got, want)
					}
				}
			}
		}
	}
}

// sweepPoints and sweepFractals are how many random points the sweeps
// ask, picked 2026-11-12 to take about a second together.
const (
	sweepPoints   = 2000000
	sweepFractals = 200000
)

// point is a seed and a place for a sweep: places near the middle, far
// out, on the lattice, on halves, and a coordinate a signed zero.
func point(random *rand.Rand, index int) (uint64, [3]float64) {
	seed := random.Uint64()
	scale := []float64{1, 40, 1e6, 1e12}[index%4]
	at := [3]float64{random.NormFloat64() * scale,
		random.NormFloat64() * scale, random.NormFloat64() * scale}
	switch index % 7 {
	case 0:
		at[index%3] = math.Round(at[index%3])
	case 1:
		at[index%3] = math.Round(at[index%3]) + 0.5
	case 2:
		at[index%3] = math.Copysign(0, -1)
	}
	return seed, at
}

// sweep is how many of the random points a gradient differs from the
// reference at, to the bit.
func sweep(gradient func(uint64, float64, float64, float64) float64) int {
	random := rand.New(rand.NewPCG(1, 2))
	apart := 0
	for index := range sweepPoints {
		seed, at := point(random, index)
		if !sameBits(gradient(seed, at[0], at[1], at[2]),
			referenceGradient(seed, at[0], at[1], at[2])) {
			apart++
		}
	}
	return apart
}

// TestGradientIsTheReferenceToTheBit asks Gradient and the reference at
// two million random points: not one bit differs.
func TestGradientIsTheReferenceToTheBit(t *testing.T) {
	if apart := sweep(Gradient); apart != 0 {
		t.Fatalf("%d of %d points differ from the reference", apart,
			sweepPoints)
	}
}

// TestTheSweepCatchesOneWrongEdgeAndOneUlp is the anti-test: with one
// edge's sign turned round, or a result moved by one unit in the last
// place, the sweep finds points that differ.
func TestTheSweepCatchesOneWrongEdgeAndOneUlp(t *testing.T) {
	kept := edges[cubeEdges-1]
	edges[cubeEdges-1].firstSign = plus
	turned := sweep(Gradient)
	edges[cubeEdges-1] = kept
	if turned == 0 {
		t.Error("one edge turned round went unseen")
	}
	nudged := func(seed uint64, x, y, z float64) float64 {
		return math.Nextafter(Gradient(seed, x, y, z), math.Inf(1))
	}
	if sweep(nudged) != sweepPoints {
		t.Error("a result one unit in the last place up went unseen")
	}
	if !sameBits(0, 0) || sameBits(0, math.Copysign(0, -1)) {
		t.Error("the bits of the zeros are not told apart")
	}
}

// TestFractalIsTheReferenceToTheBit asks Fractal of one to eight octaves
// at random points, and the reference: not one bit differs.
func TestFractalIsTheReferenceToTheBit(t *testing.T) {
	random := rand.New(rand.NewPCG(3, 4))
	for index := range sweepFractals {
		seed, at := point(random, index)
		octaves := 1 + index%8
		want := referenceFractal(seed, at[0], at[1], at[2], octaves)
		if got := Fractal(seed, at[0], at[1], at[2], octaves); !sameBits(
			got, want) {
			t.Fatalf("seed %d at %v, %d octaves: %v, want %v", seed, at,
				octaves, got, want)
		}
	}
}

// TestTheCornersAreBits3s holds the shared hash to bits3 for the eight
// corners of cells near the middle and at the ends of the int64s.
func TestTheCornersAreBits3s(t *testing.T) {
	for _, cell := range [][3]int64{{0, 0, 0}, {-1, 5, -7},
		{math.MaxInt64, math.MinInt64, -1}, {3, math.MaxInt64, 12}} {
		inner := corners(cell[1], cell[2])
		for dx := range int64(2) {
			for dy := range int64(2) {
				for dz := range int64(2) {
					want := bits3(9, cell[0]+dx, cell[1]+dy, cell[2]+dz)
					got := mix(9 ^ mix(uint64(cell[0]+dx)+inner[dy][dz]))
					if got != want {
						t.Errorf("cell %v corner %d%d%d: %#x, want %#x",
							cell, dx, dy, dz, got, want)
					}
				}
			}
		}
	}
}

// fractalSeeds is how many seeds the shared sweeps ask at once: three,
// as a smoke column's eddies ask them, picked 2026-11-12.
const fractalSeeds = 3

// TestFractalsAreFractalsToTheBit asks Fractals for three seeds at two
// hundred thousand random points, signed zeros and lattice points among
// them, from no octaves to past what a float holds: each result is the
// reference Fractal of its own seed, to the bit.
func TestFractalsAreFractalsToTheBit(t *testing.T) {
	random := rand.New(rand.NewPCG(5, 6))
	octaveCounts := []int{-1, 0, 1, 2, 4, 6, 8}
	into := make([]float64, fractalSeeds)
	for index := range sweepFractals {
		seed, at := point(random, index)
		seeds := []uint64{seed, seed + 1, seed + 2}
		if index%2 == 1 {
			seeds = []uint64{seed, random.Uint64(), seed ^ 1}
		}
		octaves := octaveCounts[index%len(octaveCounts)]
		Fractals(seeds, at[0], at[1], at[2], octaves, into)
		for which, one := range seeds {
			want := referenceFractal(one, at[0], at[1], at[2], octaves)
			if !sameBits(into[which], want) {
				t.Fatalf("seed %d at %v, %d octaves: %v, want %v", one, at,
					octaves, into[which], want)
			}
		}
	}
}

// pastAFloat is octaves enough that a point's octaves overflow a float,
// and overflowPoints how many points ask that many, picked 2026-11-12:
// each asks a thousand octaves, so a few hundred say it.
const (
	pastAFloat     = 1100
	overflowPoints = 300
)

// TestFractalsPastWhatAFloatHoldsAreFractals asks octaves past where a
// coordinate overflows: the octaves past it are left out, as Fractal
// leaves them, to the bit.
func TestFractalsPastWhatAFloatHoldsAreFractals(t *testing.T) {
	random := rand.New(rand.NewPCG(7, 8))
	into := make([]float64, fractalSeeds)
	for index := range overflowPoints {
		seed, at := point(random, index)
		seeds := []uint64{seed, seed + 1, seed + 2}
		Fractals(seeds, at[0], at[1], at[2], pastAFloat, into)
		for which, one := range seeds {
			want := referenceFractal(one, at[0], at[1], at[2], pastAFloat)
			if !sameBits(into[which], want) {
				t.Fatalf("seed %d at %v: %v, want %v", one, at,
					into[which], want)
			}
		}
	}
}

// TestFractalsTellSeedsApartAndLeaveTheRestAlone is the anti-test: the
// same seeds asked in the other order give their results in that order,
// so a result put in another seed's place is caught; and the places of
// into past the seeds are left as they were.
func TestFractalsTellSeedsApartAndLeaveTheRestAlone(t *testing.T) {
	const kept = 42.0
	into := []float64{0, 0, 0, kept}
	Fractals([]uint64{7, 8, 9}, 0.3, 1.7, -2.2, 5, into)
	turned := make([]float64, fractalSeeds)
	Fractals([]uint64{9, 8, 7}, 0.3, 1.7, -2.2, 5, turned)
	if !sameBits(into[0], turned[2]) || sameBits(into[0], turned[0]) {
		t.Error("the seeds' results are not in their seeds' places")
	}
	if into[fractalSeeds] != kept {
		t.Errorf("a place past the seeds became %v", into[fractalSeeds])
	}
	if !sameBits(into[0], Fractal(7, 0.3, 1.7, -2.2, 5)) {
		t.Error("Fractals is not Fractal for the first seed")
	}
	none := []float64{kept}
	Fractals(nil, 0.3, 1.7, -2.2, 5, none)
	if none[0] != kept {
		t.Error("no seeds wrote a result")
	}
}
