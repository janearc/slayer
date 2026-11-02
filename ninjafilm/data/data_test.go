package data_test

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/ninjafilm/data"
	"pgregory.net/rapid"
)

// how strconv is asked to write a number
const (
	shortest = -1 // defined: strconv's code for the fewest digits that read back exactly
	decimal  = 10 // defined: the base ten numbers are written in
	wordBits = 64 // defined: an int and a float64 are 64 bits wide here
	either   = 0  // defined: math.IsInf's code for an infinity of either sign
)

// recovered keeps what a panic was given, and stops the panic.
func recovered(into *any) { *into = recover() }

// panics is what a call panics with, or nothing if it does not.
func panics(call func()) (message any) {
	defer recovered(&message)
	call()
	return message
}

// absent is a name the picked numbers do not hold.
const absent = "there-is-no-such-number"

func askForAbsent() { data.Picked(absent) }

// asWhole asks for a number that is picked but not whole.
func askForAFraction() { data.Whole("hash-avalanche-least") }

func TestAPickedNumberIsAskedForByName(t *testing.T) {
	whole := data.Whole("hash-bins")
	if got := data.Picked("hash-bins"); got != float64(whole) {
		t.Errorf("hash-bins is %v and then %d", got, whole)
	}
}

// anti-tests: a name nobody picked stops the program and says which.
// So does a number that is not whole when a whole count is asked for.
func TestAPickedNumberThatIsNotThereOrNotWholeIsRefused(t *testing.T) {
	message := panics(askForAbsent)
	if text, ok := message.(error); !ok ||
		!strings.Contains(text.Error(), absent) {
		t.Errorf("a name that is not there panicked with %v", message)
	}
	message = panics(askForAFraction)
	if text, ok := message.(error); !ok ||
		!strings.Contains(text.Error(), "hash-avalanche-least") {
		t.Errorf("a number that is not whole panicked with %v", message)
	}
}

// The tables the stocks come from are text that reads as a table. There
// is a row for each stock, for each record and for each gauge.
func TestTheTablesOfTheStocksRead(t *testing.T) {
	for column, table := range map[string]string{"name": data.StocksTable(),
		"stock": data.RecordsTable(), "width_mm": data.GaugesTable()} {
		rows, err := data.Parse(table)
		if err != nil || len(rows) == nothing {
			t.Fatalf("the table with %q read as %v, %v", column, rows,
				err)
		}
		if _, err := rows[nothing].Text(column); err != nil {
			t.Errorf("the table has no column %q: %v", column, err)
		}
	}
}

// nothing is no count at all, defined as the count of nothing.
const nothing = 0

func TestAHeadingAloneIsATableOfNoRows(t *testing.T) {
	rows, err := data.Parse("a,b\n")
	if err != nil || len(rows) != nothing {
		t.Errorf("a heading alone read as %v, %v", rows, err)
	}
}

// anti-tests: no text at all is not a table. Nor is a row with more or
// fewer fields than the heading.
func TestAnEmptyOrRaggedTableIsRefused(t *testing.T) {
	for _, text := range []string{"", "a,b\n1\n", "a,b\n1,2,3\n",
		"a,b\n\"1,2\n"} {
		if rows, err := data.Parse(text); err == nil {
			t.Errorf("%q read as %v", text, rows)
		} else if strings.Contains(err.Error(), "heading") != (text == "") {
			t.Errorf("%q was refused with the wrong reason: %v", text, err)
		}
	}
}

// formatted is a number as the shortest text that reads back as it.
func formatted(number float64) string {
	return strconv.FormatFloat(number, 'g', shortest, wordBits)
}

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Fatalf(format string, args ...any)
}

// single is the count of one row, defined as one.
const single = 1

// rowOf is the one row of the table that a heading and a field make.
func rowOf(t stopper, field string) data.Row {
	rows, err := data.Parse("name,other\n" + field + ",x\n")
	if err != nil || len(rows) != single {
		t.Fatalf("a table of one row read as %v, %v", rows, err)
	}
	return rows[nothing]
}

func TestAFieldIsReadBackAsTheNumberItWas(t *testing.T) {
	rapid.Check(t, fieldsReadBack)
}

func fieldsReadBack(t *rapid.T) {
	number := rapid.Float64().Filter(isFinite).Draw(t, "number")
	whole := rapid.Int().Draw(t, "whole")
	row := rowOf(t, formatted(number))
	if got, err := row.Float("name"); err != nil || got != number {
		t.Fatalf("%v was read back as %v, %v", number, got, err)
	}
	text := strconv.FormatInt(int64(whole), decimal)
	row = rowOf(t, text)
	if got, err := row.Whole("name"); err != nil || got != whole {
		t.Fatalf("%d was read back as %d, %v", whole, got, err)
	}
	if got, err := row.Text("other"); err != nil || got != "x" {
		t.Fatalf("the other column read as %q, %v", got, err)
	}
}

// isFinite is whether a number is neither endless nor lost.
func isFinite(number float64) bool {
	return !math.IsInf(number, either) && !math.IsNaN(number)
}

// anti-tests: text that is not a number is not read as one, and text
// that is not whole is not read as a whole number.
func TestAFieldThatIsNotANumberIsRefused(t *testing.T) {
	for _, text := range []string{"x", "", "1.5x", "NaN", "Inf", "-Inf"} {
		row := rowOf(t, text)
		if got, err := row.Float("name"); err == nil {
			t.Errorf("%q was read as the number %v", text, got)
		}
	}
	for _, text := range []string{"x", "", "3.5", "1e3"} {
		row := rowOf(t, text)
		if got, err := row.Whole("name"); err == nil {
			t.Errorf("%q was read as the whole number %v", text, got)
		}
	}
}

// anti-tests: a column the heading does not name is refused by every
// way of asking for it.
func TestAColumnThatIsNotThereIsRefused(t *testing.T) {
	row := rowOf(t, "1")
	if text, err := row.Text(absent); err == nil {
		t.Errorf("a column that is not there read as %q", text)
	}
	if number, err := row.Float(absent); err == nil || !strings.Contains(err.Error(), "column") {
		t.Errorf("a column that is not there read as %v, %v", number, err)
	}
	if whole, err := row.Whole(absent); err == nil || !strings.Contains(err.Error(), "column") {
		t.Errorf("a column that is not there read as %d, %v", whole, err)
	}
}
