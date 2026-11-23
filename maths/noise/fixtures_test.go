package noise_test

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"

	"github.com/janearc/puzzlemouth/numbers"
)

// What the tests count in, each defined.
const (
	nought       = 0   // defined: no amount
	unity        = 1   // defined: one, and the next index after any
	pair         = 2   // defined: two, the index after the next
	axes         = 3   // defined: x, y and z, the numbers a point has
	half         = 0.5 // defined: a half, the middle of a cell along an axis
	floatWidth   = 64  // defined: the bits in the float64 a column is read as
	fromPrefix   = 0   // defined by strconv.ParseUint: base nought takes the base from the prefix, 0x for the recorded bits
	plusInfinity = 1   // defined: the sign math.Inf takes for plus infinity
)

// The files the tests read. Every number a test chooses is in one of
// them, and the tests ask for it by name. recorded.csv holds the numbers
// the package gives, bit for bit, as recorded.
const (
	numbersFile  = "testdata/numbers.yaml"
	recordedFile = "testdata/recorded.csv"
	latticeFile  = "testdata/lattice.csv"
	octavesFile  = "testdata/octaves.csv"
	sweepsFile   = "testdata/sweeps.csv"
	reachesFile  = "testdata/reaches.csv"
	countsFile   = "testdata/counts.csv"
)

// table is one csv file read: its rows, each by the names in its first row.
type table []map[string]string

// loaders holds the one reading of each csv file asked for. Property
// tests ask on every draw. Each file is read once.
var (
	loadersLock sync.Mutex
	loaders     = map[string]func() (table, error){}
)

// read is the rows of a csv file, read once. Files that cannot be read
// stop the test. Callers must not change the rows.
func read(test reporter, file string) table {
	test.Helper()
	loadersLock.Lock()
	load, ok := loaders[file]
	if !ok {
		load = sync.OnceValues(loading{file}.load)
		loaders[file] = load
	}
	loadersLock.Unlock()
	rows, err := load()
	if err != nil {
		test.Fatal(err)
	}
	return rows
}

// loading is a csv file to be read once, by name.
type loading struct{ file string }

// load reads the file.
func (named loading) load() (table, error) { return load1(named.file) }

// load1 reads a csv file from disk.
func load1(file string) (table, error) {
	opened, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	all, err := csv.NewReader(opened).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", file, err)
	}
	if len(all) == nought {
		return nil, fmt.Errorf("%s: has no header", file)
	}
	header, rows := all[nought], all[unity:]
	out := make(table, len(rows))
	for place, row := range rows {
		out[place] = map[string]string{}
		for column, name := range header {
			out[place][name] = row[column]
		}
	}
	return out, nil
}

// by is the row of a name, or a stop naming the file.
func (rows table) by(test reporter, file, name string) map[string]string {
	test.Helper()
	for _, row := range rows {
		if row["name"] == name {
			return row
		}
	}
	test.Fatalf("%s: no row called %q", file, name)
	return nil
}

// float is the number in a column of a row. Columns that are no number
// stop the test.
func float(test reporter, row map[string]string, column string) float64 {
	test.Helper()
	got, err := strconv.ParseFloat(row[column], floatWidth)
	if err != nil {
		test.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// whole is the whole number in a column of a row, or a stop.
func whole(test reporter, row map[string]string, column string) int64 {
	test.Helper()
	got, err := strconv.ParseInt(row[column], fromPrefix, floatWidth)
	if err != nil {
		test.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// seed is the seed in a column of a row, or a stop.
func seed(test reporter, row map[string]string, column string) uint64 {
	test.Helper()
	got, err := strconv.ParseUint(row[column], fromPrefix, floatWidth)
	if err != nil {
		test.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// bits are the float64 bits written in a column of a row, as 0x and
// sixteen hex digits, or a stop.
func bits(test reporter, row map[string]string, column string) uint64 {
	test.Helper()
	return seed(test, row, column)
}

// chosenNumbers is numbers.yaml, read once.
var chosenNumbers = sync.OnceValues(readNumbers)

// readNumbers reads numbers.yaml.
func readNumbers() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// picked is a number someone chose, by its name in numbers.yaml.
func picked(test reporter, name string) float64 {
	test.Helper()
	chosen, err := chosenNumbers()
	if err != nil {
		test.Fatal(err)
	}
	number, err := chosen.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return number.Value
}

// count is a picked number that counts something, as an int.
func count(test reporter, name string) int {
	test.Helper()
	return int(picked(test, name))
}

// pickedSeed is a picked number that is a seed.
func pickedSeed(test reporter, name string) uint64 {
	test.Helper()
	return uint64(picked(test, name))
}

// place is a point the tests ask the noise at.
type place struct{ x, y, z float64 }

// point is the place at the x, y and z columns of a row.
func point(test reporter, row map[string]string) place {
	test.Helper()
	return place{float(test, row, "x"), float(test, row, "y"),
		float(test, row, "z")}
}

// thrown is a point in the box the tests throw points in, from the
// point's number: each coordinate is a hash of the number under its own
// seed, spread across the span and moved back by the reach, so the box
// is centred on the origin.
func thrown(test reporter, index int64) place {
	test.Helper()
	span, reach := picked(test, "span"), picked(test, "reach")
	return place{
		hashed(test, "coordinate-seed-x", index)*span - reach,
		hashed(test, "coordinate-seed-y", index)*span - reach,
		hashed(test, "coordinate-seed-z", index)*span - reach}
}

// sameBits is whether two numbers are one to the bit: a negative zero is
// not a zero.
func sameBits(one, other float64) bool {
	return math.Float64bits(one) == math.Float64bits(other)
}

// plusInf is plus infinity.
func plusInf() float64 { return math.Inf(plusInfinity) }

// reporter is what a test and a property test both do: say where a
// helper is, and stop with a message. testing.T, testing.B and rapid.T
// all are.
type reporter interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
}
