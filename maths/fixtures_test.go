package maths_test

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
)

// What the tests count in, each defined.
const (
	nought     = 0  // defined: no amount
	unity      = 1  // defined: one
	floatWidth = 64 // defined: the bits in the float64 a column is read as
	quarter    = 90 // defined: degrees in a quarter turn
	infinity   = 1  // defined: the sign math.IsInf and math.Inf take for plus infinity
)

// The files the tests read. Every number a test chooses is in one of
// them, and the tests ask for it by name.
const (
	numbersFile      = "testdata/numbers.yaml"
	vectorsFile      = "testdata/vectors.csv"
	orientationsFile = "testdata/orientations.csv"
	spacesFile       = "testdata/subspaces.csv"
	hitsFile         = "testdata/hits.csv"
	turnsFile        = "testdata/turns.csv"
	looksFile        = "testdata/looks.csv"
	scalesFile       = "testdata/scales.csv"
	smoothFile       = "testdata/smooth.csv"
	shortsFile       = "testdata/shorts.csv"
)

// table is one csv file read: its rows, each by the names in its first row.
type table []map[string]string

// loaders holds the one reading of each csv file asked for.
// Property tests ask on every draw. Each file is read once.
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
		load = sync.OnceValues(func() (table, error) { return load1(file) })
		loaders[file] = load
	}
	loadersLock.Unlock()
	rows, err := load()
	if err != nil {
		test.Fatal(err)
	}
	return rows
}

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

// chosenNumbers is numbers.yaml, read once.
var chosenNumbers = sync.OnceValues(func() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
})

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

// vector is a named vector of vectors.csv.
func vector(test reporter, name string) maths.Vec {
	test.Helper()
	row := read(test, vectorsFile).by(test, vectorsFile, name)
	return maths.V(float(test, row, "x"), float(test, row, "y"), float(test, row, "z"))
}

// orientation is a named orientation of orientations.csv. Its x, y and z
// each go where three numbers say.
func orientation(test reporter, name string) maths.Orientation {
	test.Helper()
	row := read(test, orientationsFile).by(test, orientationsFile, name)
	return maths.Orientation{X: columns(test, row, "x"), Y: columns(test, row, "y"),
		Z: columns(test, row, "z")}
}

// columns is the vector a row holds under three columns. The columns are
// named by a letter and then x, y and z.
func columns(test reporter, row map[string]string, letter string) maths.Vec {
	test.Helper()
	return maths.V(float(test, row, letter+"x"), float(test, row, letter+"y"),
		float(test, row, letter+"z"))
}

// turned is the rotation about a named axis by a number of degrees.
func turned(test reporter, axis string, degrees float64) maths.Orientation {
	test.Helper()
	return maths.Rotation(vector(test, axis), degrees*maths.Degree)
}

// space is a named subspace of subspaces.csv.
func space(test reporter, name string) maths.Subspace {
	test.Helper()
	row := read(test, spacesFile).by(test, spacesFile, name)
	made := maths.Subspace{Origin: vector(test, row["origin"]),
		Scale: float(test, row, "scale")}
	if row["axis"] != "" {
		made.Orientation = turned(test, row["axis"], float(test, row, "degrees"))
	}
	return made
}

// within is the tolerance a rounded answer is held to.
func within(test reporter) float64 { return picked(test, "within") }

// near stops the test unless two vectors are within the tolerance.
func near(test reporter, what string, got, want maths.Vec) {
	test.Helper()
	if !got.Near(want, within(test)) {
		test.Errorf("%s: %v, want %v", what, got, want)
	}
}

// exactly stops the test unless two vectors are the same bit for bit.
func exactly(test reporter, what string, got, want maths.Vec) {
	test.Helper()
	if got != want {
		test.Errorf("%s: %v, want %v", what, got, want)
	}
}

// nearTurn stops the test unless two orientations are within the
// tolerance, axis by axis.
func nearTurn(test reporter, what string, got, want maths.Orientation) {
	test.Helper()
	near(test, what+", x", got.X, want.X)
	near(test, what+", y", got.Y, want.Y)
	near(test, what+", z", got.Z, want.Z)
}

// plusInf is plus infinity.
func plusInf() float64 { return math.Inf(infinity) }

// abs is the size of a number without its sign.
func abs(value float64) float64 { return math.Abs(value) }

// light is a named vector of vectors.csv read as an amount of light. Its
// three parts are the red, green and blue.
func light(test reporter, name string) maths.Light {
	test.Helper()
	from := vector(test, name)
	return maths.RGB(from.X, from.Y, from.Z)
}

// reporter is what a test and a property test both do: say where a
// helper is, and stop with a message. testing.T and rapid.T both are.
type reporter interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
}
