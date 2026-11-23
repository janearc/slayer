package thing

import (
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv tables the benchmark gas is read from.
const (
	excitedBenchLineTable       = "testdata/excited_bench-lines.csv"
	excitedBenchExcitationTable = "testdata/excited_bench-excitations.csv"
)

// benchExcited is a star-lit gas of four lines, as the pillars glow. Its
// numbers are in testdata. It has no test to stop, so a file that cannot be
// read panics.
func benchExcited() Excited {
	gas, err := excitedBenchRead()
	if err != nil {
		panic(err)
	}
	return *gas
}

// excitedBenchRead is the benchmark gas, read from testdata.
func excitedBenchRead() (*Excited, error) {
	lines, err := excitedBenchTable(excitedBenchLineTable, "strength")
	if err != nil {
		return nil, err
	}
	excitations, err := excitedBenchTable(excitedBenchExcitationTable,
		"excitation")
	if err != nil {
		return nil, err
	}
	width, err := excitedBenchPicked("excited_bench-width")
	if err != nil {
		return nil, err
	}
	passes, err := excitedBenchPicked("excited_bench-passes")
	if err != nil {
		return nil, err
	}
	return &Excited{Lines: lines, Excitations: excitations,
		Width: width, Passes: passes}, nil
}

// excitedBenchPicked is a number from numbers.yaml, by its name.
func excitedBenchPicked(name string) (float64, error) {
	chosen, err := chosenNumbers()
	if err != nil {
		return nought, err
	}
	number, err := chosen.Get(name)
	if err != nil {
		return nought, err
	}
	return number.Value, nil
}

// excitedBenchTable is a csv file's column of values, each by the
// wavelength in its nm column.
func excitedBenchTable(file, column string) (map[float64]float64, error) {
	rows, err := loadSheet(file)
	if err != nil {
		return nil, err
	}
	table := map[float64]float64{}
	for _, row := range rows {
		nanometres, err := strconv.ParseFloat(row["nm"], floatBits)
		if err != nil {
			return nil, err
		}
		value, err := strconv.ParseFloat(row[column], floatBits)
		if err != nil {
			return nil, err
		}
		table[nanometres] = value
	}
	return table, nil
}

// excitedBenchSweep is the bands the benchmarks ask for: a run of whole
// nanometres from a first one, which begins again after a cycle.
type excitedBenchSweep struct {
	base  float64
	cycle int
}

// excitedBenchSweepFor is the sweep the benchmarks run, or a stop.
func excitedBenchSweepFor(bench *testing.B) excitedBenchSweep {
	bench.Helper()
	base, err := excitedBenchPicked("excited_bench-sweep-base")
	if err != nil {
		bench.Fatal(err)
	}
	cycle, err := excitedBenchPicked("excited_bench-sweep-cycle")
	if err != nil {
		bench.Fatal(err)
	}
	return excitedBenchSweep{base: base, cycle: int(cycle)}
}

// band is the band a benchmark asks for at a step.
func (sweep excitedBenchSweep) band(step int) maths.Band {
	return maths.NewBand(sweep.base + float64(step%sweep.cycle))
}

// sinkF takes what the benchmarks compute, so that it is used.
var sinkF float64

// BenchmarkExcitedEmits is Emits on a gas built by a literal.
func BenchmarkExcitedEmits(bench *testing.B) {
	gas := benchExcited()
	sweep := excitedBenchSweepFor(bench)
	bench.ReportAllocs()
	for i := nought; i < bench.N; i++ {
		sinkF += gas.Emits(sweep.band(i))
	}
}

// BenchmarkExcitedExcitation is Excitation on a gas built by a literal.
func BenchmarkExcitedExcitation(bench *testing.B) {
	gas := benchExcited()
	sweep := excitedBenchSweepFor(bench)
	bench.ReportAllocs()
	for i := nought; i < bench.N; i++ {
		sinkF += gas.Excitation(sweep.band(i))
	}
}

// BenchmarkExcitedEmitsPrepared is Emits on a prepared gas.
func BenchmarkExcitedEmitsPrepared(bench *testing.B) {
	gas := benchExcited().Prepared()
	sweep := excitedBenchSweepFor(bench)
	bench.ReportAllocs()
	for i := nought; i < bench.N; i++ {
		sinkF += gas.Emits(sweep.band(i))
	}
}

// BenchmarkExcitedExcitationPrepared is Excitation on a prepared gas.
func BenchmarkExcitedExcitationPrepared(bench *testing.B) {
	gas := benchExcited().Prepared()
	sweep := excitedBenchSweepFor(bench)
	bench.ReportAllocs()
	for i := nought; i < bench.N; i++ {
		sinkF += gas.Excitation(sweep.band(i))
	}
}

// BenchmarkExcitedEmitsOld is the reference Emits, to set beside the others.
func BenchmarkExcitedEmitsOld(bench *testing.B) {
	gas := benchExcited()
	sweep := excitedBenchSweepFor(bench)
	bench.ReportAllocs()
	for i := nought; i < bench.N; i++ {
		sinkF += oldEmits(gas, sweep.band(i))
	}
}
