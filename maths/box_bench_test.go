package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// benchBox is the box the benchmarks ask, by the names of vectors.csv.
func benchBox(bench *testing.B) maths.Box {
	bench.Helper()
	return maths.Box{Min: vector(bench, "bench box min"),
		Max: vector(bench, "bench box max")}
}

// benchRay is the ray the benchmarks ask, which crosses benchBox.
func benchRay(bench *testing.B) maths.Ray {
	bench.Helper()
	return maths.Ray{At: vector(bench, "bench ray start"),
		Dir: vector(bench, "bench ray way").Unit()}
}

// BenchmarkBoxHit is one box asked by a ray that crosses it, which is
// the question a tree walk asks of every branch.
func BenchmarkBoxHit(bench *testing.B) {
	box, ray, far := benchBox(bench), benchRay(bench), picked(bench, "far-reach")
	bench.ReportAllocs()
	bench.ResetTimer()
	for range bench.N {
		box.Hit(ray, nought, far)
	}
}

// BenchmarkBoxMeets is BenchmarkBoxHit's box and ray asked by Meets, with the
// reciprocal taken once outside the loop, as a tree walk takes it.
func BenchmarkBoxMeets(bench *testing.B) {
	box, ray, far := benchBox(bench), benchRay(bench), picked(bench, "far-reach")
	inverse := ray.Dir.Inverse()
	bench.ReportAllocs()
	bench.ResetTimer()
	for range bench.N {
		box.Meets(ray, inverse, nought, far)
	}
}
