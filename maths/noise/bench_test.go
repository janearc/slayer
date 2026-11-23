package noise_test

import (
	"testing"

	"github.com/janearc/slayer/maths/noise"
)

// The benchmarks. A benchmark is a test, and tests here are run by hand:
// these are written to be run through the lane on a quiet machine.
//
// Each walks its points across many cells by the point's number, so the
// hashes, the branch on the edge and the cache see what a caller's loop
// shows them, not one point over and over.

// sink keeps the compiler from dropping a result nobody reads.
var sink float64

// walked is the point a benchmark asks at its iteration.
func walked(bench *testing.B, index int64) place {
	bench.Helper()
	return thrown(bench, index)
}

func BenchmarkHash(bench *testing.B) {
	seed := pickedSeed(bench, "field-seed")
	sum := nothing
	for index := range int64(bench.N) {
		sum += noise.Hash(seed, index)
	}
	sink = sum
}

func BenchmarkHash3(bench *testing.B) {
	seed := pickedSeed(bench, "field-seed")
	sum := nothing
	for index := range int64(bench.N) {
		sum += noise.Hash3(seed, index, -index, index+unity)
	}
	sink = sum
}

func BenchmarkGradient(bench *testing.B) {
	seed := pickedSeed(bench, "field-seed")
	points := make([]place, count(bench, "smooth-points"))
	for index := range points {
		points[index] = walked(bench, int64(index))
	}
	bench.ResetTimer()
	sum := nothing
	for index := range bench.N {
		at := points[index%len(points)]
		sum += noise.Gradient(seed, at.x, at.y, at.z)
	}
	sink = sum
}

// BenchmarkFractal is Fractal of the picked octave count, what most
// callers ask, at points walked across cells.
func BenchmarkFractal(bench *testing.B) {
	seed, octaves := pickedSeed(bench, "fractal-seed"), count(bench, "bench-octaves")
	points := make([]place, count(bench, "smooth-points"))
	for index := range points {
		points[index] = walked(bench, int64(index))
	}
	bench.ResetTimer()
	sum := nothing
	for index := range bench.N {
		at := points[index%len(points)]
		sum += noise.Fractal(seed, at.x, at.y, at.z, octaves)
	}
	sink = sum
}

// BenchmarkFractalsThree and BenchmarkThreeFractals ask the same three
// fields at the same points, once through Fractals and once through
// Fractal three times, which is what Fractals is for.
func BenchmarkFractalsThree(bench *testing.B) {
	first, octaves := pickedSeed(bench, "fractal-seed"), count(bench, "bench-octaves")
	seeds := []uint64{first, first + unity, first + pair}
	into := make([]float64, len(seeds))
	points := make([]place, count(bench, "smooth-points"))
	for index := range points {
		points[index] = walked(bench, int64(index))
	}
	bench.ResetTimer()
	sum := nothing
	for index := range bench.N {
		at := points[index%len(points)]
		noise.Fractals(seeds, at.x, at.y, at.z, octaves, into)
		sum += into[nought] + into[unity] + into[pair]
	}
	sink = sum
}

func BenchmarkThreeFractals(bench *testing.B) {
	first, octaves := pickedSeed(bench, "fractal-seed"), count(bench, "bench-octaves")
	points := make([]place, count(bench, "smooth-points"))
	for index := range points {
		points[index] = walked(bench, int64(index))
	}
	bench.ResetTimer()
	sum := nothing
	for index := range bench.N {
		at := points[index%len(points)]
		sum += noise.Fractal(first, at.x, at.y, at.z, octaves) +
			noise.Fractal(first+unity, at.x, at.y, at.z, octaves) +
			noise.Fractal(first+pair, at.x, at.y, at.z, octaves)
	}
	sink = sum
}
