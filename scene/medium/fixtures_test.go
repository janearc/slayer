package medium

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// What the tests count in beyond the package's own nought, unity and double,
// each true by definition.
const (
	floatBits  = 64 // defined: the bits in the float64 a cell is read as
	prefixBase = 0  // defined by strconv: base nought takes the base from the prefix
)

// The files the tests read. Every number a test chooses is in numbers.yaml,
// and a test asks for it by name. Tables of rows are csv files beside it, and
// a test asks for a row by its name column.
const numbersFile = "testdata/numbers.yaml"

// chosenNumbers is numbers.yaml, read once.
var chosenNumbers = sync.OnceValues(readChosenNumbers)

// readChosenNumbers reads numbers.yaml.
func readChosenNumbers() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// picked is a number the tests chose, by its name in numbers.yaml. A name
// the file does not hold stops the test.
func picked(t *testing.T, name string) float64 {
	t.Helper()
	chosen, err := chosenNumbers()
	if err != nil {
		t.Fatal(err)
	}
	number, err := chosen.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// pickedCount is a picked number that counts something, as an int.
func pickedCount(t *testing.T, name string) int {
	t.Helper()
	return int(picked(t, name))
}

// sheet is a csv file read: its rows, each by the names in its first row.
type sheet []map[string]string

// sheets holds the one reading of each csv file asked for, by its name.
var (
	sheetsLock sync.Mutex
	sheets     = map[string]func() (sheet, error){}
)

// sheetLoader is a csv file to be read once.
type sheetLoader struct{ file string }

// load reads the file.
func (loader sheetLoader) load() (sheet, error) { return loadSheet(loader.file) }

// readSheet is the rows of a csv file, read once. A file that cannot be
// read stops the test. Callers must not change the rows.
func readSheet(t *testing.T, file string) sheet {
	t.Helper()
	sheetsLock.Lock()
	load, known := sheets[file]
	if !known {
		load = sync.OnceValues(sheetLoader{file}.load)
		sheets[file] = load
	}
	sheetsLock.Unlock()
	rows, err := load()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// loadSheet reads a csv file from disk.
func loadSheet(file string) (sheet, error) {
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
	header, body := all[nought], all[unity:]
	rows := make(sheet, len(body))
	for place, line := range body {
		rows[place] = map[string]string{}
		for column, name := range header {
			rows[place][name] = line[column]
		}
	}
	return rows, nil
}

// row is the row whose name column is name, or a stop naming the file.
func (rows sheet) row(t *testing.T, file, name string) map[string]string {
	t.Helper()
	for _, each := range rows {
		if each["name"] == name {
			return each
		}
	}
	t.Fatalf("%s: no row called %q", file, name)
	return nil
}

// cell is the number in a column of a row. A cell that is no number stops
// the test.
func cell(t *testing.T, row map[string]string, column string) float64 {
	t.Helper()
	got, err := strconv.ParseFloat(row[column], floatBits)
	if err != nil {
		t.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// cellCount is the whole number in a column of a row, or a stop.
func cellCount(t *testing.T, row map[string]string, column string) int {
	t.Helper()
	got, err := strconv.ParseInt(row[column], prefixBase, floatBits)
	if err != nil {
		t.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return int(got)
}
