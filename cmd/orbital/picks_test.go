package main_test

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// what these tests count in, each defined
const (
	nothing = 0 // defined: no count at all
	once    = 1 // defined: a call or a frame that happens one time
)

// the files the orbital tests read
const (
	numbersFile  = "testdata/numbers.yaml"
	refusedFile  = "testdata/cells-refused.csv"
	farGolden    = "testdata/far-cells.golden"
	farRendering = "-far"
)

// chosen is the picked numbers as the file holds them, and why the file
// could not be read if it could not. When the file will not read the
// numbers stay empty, and every name asked for is refused with the reason.
var chosen, unread = readChosen()

// readChosen reads the numbers file.
func readChosen() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// picked is the number of that name as a whole count. Asking for an
// unknown name or a fraction is a mistake in the tests. It stops them
// and says which.
func picked(name string) int {
	number, err := chosen.Get(name)
	if err != nil {
		refusal := fmt.Errorf("orbital: %w", err)
		panic(errors.Join(unread, refusal))
	}
	value := number.Value
	if value != math.Trunc(value) {
		panic(fmt.Errorf("orbital: %s is %v, not a whole count", name,
			value))
	}
	return int(value)
}

// tableRow is one row of a table of cases: its fields, by the names the
// table's heading gives them.
type tableRow map[string]string

// casesOf is the rows of a table of cases in testdata, read as csv under
// its heading, or the test stops.
func casesOf(t *testing.T, name string) []tableRow {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	all, err := csv.NewReader(bytes.NewReader(text)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var heading []string
	var rows []tableRow
	for _, fields := range all {
		if heading == nil {
			heading = fields
			continue
		}
		row := tableRow{}
		for place, column := range heading {
			row[column] = fields[place]
		}
		rows = append(rows, row)
	}
	if heading == nil {
		t.Fatalf("%s has no heading", name)
	}
	return rows
}

// textIn is the text in a column of a row, or the test stops.
func textIn(t *testing.T, row tableRow, column string) string {
	t.Helper()
	text, there := row[column]
	if !there {
		t.Fatalf("a row has no column called %q", column)
	}
	return text
}
