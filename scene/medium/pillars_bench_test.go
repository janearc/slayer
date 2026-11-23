package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// sinkF takes each benchmark's result so the compiler keeps the work.
var sinkF float64

// pillarsBenchPicked is a number the benchmarks chose, by its name in
// numbers.yaml. A benchmark holds a *testing.B where picked asks for a
// *testing.T.
func pillarsBenchPicked(bench *testing.B, name string) float64 {
	bench.Helper()
	catalogue, err := chosenNumbers()
	if err != nil {
		bench.Fatal(err)
	}
	number, err := catalogue.Get(name)
	if err != nil {
		bench.Fatal(err)
	}
	return number.Value
}

// pillarsBenchColumns is the three columns every benchmark here measures.
func pillarsBenchColumns(bench *testing.B) Pillars {
	return Pillars{
		Count:     int(pillarsBenchPicked(bench, "pillars-bench-count")),
		Height:    pillarsBenchPicked(bench, "pillars-bench-height"),
		Width:     pillarsBenchPicked(bench, "pillars-bench-width"),
		Thick:     pillarsBenchPicked(bench, "pillars-bench-thick"),
		Seed:      uint64(pillarsBenchPicked(bench, "pillars-bench-seed")),
		Filaments: pillarsBenchPicked(bench, "pillars-bench-filaments"),
	}
}

// pillarsBenchWalk steps a point through the columns: how many steps before
// the walk repeats, and the divisor and offset each axis is taken by.
type pillarsBenchWalk struct {
	cycle                                int
	acrossDivisor, acrossOffset          float64
	upDivisor, alongDivisor, alongOffset float64
}

// pillarsBenchWalkRead is the walk every benchmark here takes.
func pillarsBenchWalkRead(bench *testing.B) pillarsBenchWalk {
	return pillarsBenchWalk{
		cycle:         int(pillarsBenchPicked(bench, "pillars-bench-cycle")),
		acrossDivisor: pillarsBenchPicked(bench, "pillars-bench-across-divisor"),
		acrossOffset:  pillarsBenchPicked(bench, "pillars-bench-across-offset"),
		upDivisor:     pillarsBenchPicked(bench, "pillars-bench-up-divisor"),
		alongDivisor:  pillarsBenchPicked(bench, "pillars-bench-along-divisor"),
		alongOffset:   pillarsBenchPicked(bench, "pillars-bench-along-offset"),
	}
}

// at is the point the walk reaches at a step.
func (walk pillarsBenchWalk) at(step int) maths.Vec {
	fraction := float64(step % walk.cycle)
	x := fraction/walk.acrossDivisor - walk.acrossOffset
	y := fraction / walk.upDivisor
	z := fraction/walk.alongDivisor - walk.alongOffset
	return maths.V(x, y, z)
}

// BenchmarkPillarsDensity is the density of columns placed once.
func BenchmarkPillarsDensity(bench *testing.B) {
	pillars := pillarsBenchColumns(bench)
	walk := pillarsBenchWalkRead(bench)
	bench.ReportAllocs()
	for step := nought; step < bench.N; step++ {
		sinkF += pillars.Density(walk.at(step))
	}
}

// BenchmarkPillarsDensityLonghand is the longhand Density, which places the
// columns at each call, for the same minute as BenchmarkPillarsDensity.
func BenchmarkPillarsDensityLonghand(bench *testing.B) {
	pillars := pillarsBenchColumns(bench)
	walk := pillarsBenchWalkRead(bench)
	bench.ReportAllocs()
	for step := nought; step < bench.N; step++ {
		sinkF += longhandDensity(pillars, walk.at(step))
	}
}

// BenchmarkPillarsDensityPrepared is the density of columns prepared before
// the first query.
func BenchmarkPillarsDensityPrepared(bench *testing.B) {
	pillars := pillarsBenchColumns(bench).Prepared()
	walk := pillarsBenchWalkRead(bench)
	bench.ReportAllocs()
	for step := nought; step < bench.N; step++ {
		sinkF += pillars.Density(walk.at(step))
	}
}
