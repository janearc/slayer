package mesh

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// numbersFiles is where every number the tests choose is, each with what
// it is and why it was chosen: one file for each test file that has
// numbers, named for it, so that none is in two places. A test asks for a
// number by its name, and no name is in two files.
const numbersFiles = "testdata/numbers*.yaml"

// chosenNumbers is every numbers file, read once.
var chosenNumbers = sync.OnceValues(readNumbers)

// readNumbers reads every numbers file there is.
func readNumbers() (numbers.Numbers, error) {
	files, err := filepath.Glob(numbersFiles)
	if err != nil {
		return nil, err
	}
	var all numbers.Numbers
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		read, err := numbers.Read(file, raw)
		if err != nil {
			return nil, err
		}
		all = append(all, read...)
	}
	return all, nil
}

// picked is a number someone chose, by its name in a numbers file. A file
// that cannot be read, or no such name, stops the test.
func picked(test testing.TB, name string) float64 {
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

// pickedCount is a picked number that counts something, as an int.
func pickedCount(test testing.TB, name string) int {
	test.Helper()
	return int(picked(test, name))
}

// rows is a csv file read: each row by the names in its first row.
type rows []map[string]string

// loaders holds the one reading of each csv file asked for.
var (
	loadersLock sync.Mutex
	loaders     = map[string]func() (rows, error){}
)

// read is the rows of a csv file in testdata, read once. A file that
// cannot be read stops the test. Callers must not change the rows.
func read(test testing.TB, file string) rows {
	test.Helper()
	loadersLock.Lock()
	load, ok := loaders[file]
	if !ok {
		load = sync.OnceValues(loading{file}.load)
		loaders[file] = load
	}
	loadersLock.Unlock()
	got, err := load()
	if err != nil {
		test.Fatal(err)
	}
	return got
}

// loading is a csv file to be read once, by name.
type loading struct{ file string }

// load reads the file.
func (named loading) load() (rows, error) {
	opened, err := os.Open(named.file)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	all, err := csv.NewReader(opened).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", named.file, err)
	}
	if len(all) == nought {
		return nil, fmt.Errorf("%s: has no header", named.file)
	}
	header, body := all[nought], all[unity:]
	made := make(rows, len(body))
	for place, row := range body {
		made[place] = map[string]string{}
		for column, name := range header {
			made[place][name] = row[column]
		}
	}
	return made, nil
}

// floatBits is the width a number in a column is read at.
const floatBits = 64 // defined: the bits of a float64

// float is the number in a column of a row. A column that is no number
// stops the test.
func float(test testing.TB, row map[string]string, column string) float64 {
	test.Helper()
	got, err := strconv.ParseFloat(row[column], floatBits)
	if err != nil {
		test.Fatalf("%s: %v", column, err)
	}
	return got
}

// whole is the whole number in a column of a row, as an int.
func whole(test testing.TB, row map[string]string, column string) int {
	test.Helper()
	return int(float(test, row, column))
}
