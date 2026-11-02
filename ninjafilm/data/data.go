// Package data embeds stock tables and chosen parameters and parses their CSV
// representations. It separates recorded data from calculated values.
package data

import (
	_ "embed"
	"errors"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/numbers"
)

// pickedFile is the picked numbers' file, as a refusal names it.
const pickedFile = "picked-numbers.yml"

var (
	//go:embed stocks.csv
	stocksTable string
	//go:embed records.csv
	recordsTable string
	//go:embed gauges.csv
	gaugesTable string
	//go:embed picked-numbers.yml
	pickedRaw []byte
)

// chosen is the picked numbers as the file holds them. unread is why
// the file could not be read, if it could not. Files that will not read
// leave the numbers empty. Every name asked for is then refused with
// the reason.
var chosen, unread = numbers.Read(pickedFile, pickedRaw)

// StocksTable is the text of the table with a row for each stock.
func StocksTable() string { return stocksTable }

// RecordsTable is the text of the table with a row for each record.
func RecordsTable() string { return recordsTable }

// GaugesTable is the text of the table with a row for each gauge of film.
func GaugesTable() string { return gaugesTable }

// Picked is a picked number by its name. Names the file does not hold
// are mistakes in the code. Its tests find them. So Picked stops the
// program and says which.
func Picked(name string) float64 {
	number, err := chosen.Get(name)
	if err != nil {
		cause := fmt.Errorf("ninjafilm: %w", err)
		panic(errors.Join(unread, cause))
	}
	return number.Value
}

// Whole is a picked number by its name as a whole count. Numbers that
// are not whole are mistakes in the file. They stop the program too.
func Whole(name string) int {
	value := Picked(name)
	if value != math.Trunc(value) {
		panic(fmt.Errorf("ninjafilm: %s is %v, not a whole count", name,
			value))
	}
	return int(value)
}
