package manufacture

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
)

// A book is one model's dimensions, counts and placements, read from files
// beside its code and checked once. The model's code asks for each number by
// name through an asking, and the book says in words which names it lacks
// and which it holds that nothing asked for.
//
// Single numbers are a numbers file, so each says what it is, its unit and
// where it came from. Numbers that are placed or sized together are a points
// file of three numbers a row; rows that share a name are a table.

// floatBits is the size a number is read at. It is defined: a float64 is
// 64 bits.
const floatBits = 64

// bookUnitsRaw is book-units.csv: every unit a book's numbers may have, and
// the span each may lie in.
//
//go:embed book-units.csv
var bookUnitsRaw []byte

// bookUnits is book-units.csv read, before any book is.
var bookUnits, bookUnitsError = readUnits(bookUnitsRaw)

// A bookUnit is the span a number of one unit may lie in, and whether it
// must be whole.
type bookUnit struct {
	low, high float64
	whole     bool
}

// bookUnitsByName is the units of the book by their names.
type bookUnitsByName map[string]bookUnit

// judge is why a value is not one of the unit, or nothing.
func (units bookUnitsByName) judge(name string, value float64) error {
	unit, known := units[name]
	switch {
	case !known:
		return fmt.Errorf("has unit %q, which the book does not know", name)
	case math.IsNaN(value) || math.Abs(value) > math.MaxFloat64:
		return errors.New("is not a finite number")
	case value < unit.low || value > unit.high:
		return fmt.Errorf("is %g, outside %g to %g for a %s", value,
			unit.low, unit.high, name)
	case unit.whole && value != math.Trunc(value):
		return fmt.Errorf("is %g, which is not whole, for a %s", value,
			name)
	}
	return nil
}

// readRows is the rows of a csv after its header, or why they are not
// readable: a header other than the one asked for, or a row of another
// width. Each row is returned as its cells by the name in the header.
func readRows(file string, raw []byte, header []string,
) ([]map[string]string, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = len(header)
	first, err := reader.Read()
	if err == nil && !slices.Equal(first, header) {
		err = fmt.Errorf("the first row must be %s",
			strings.Join(header, ","))
	}
	rows, more := reader.ReadAll()
	if err = errors.Join(err, more); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	named := make([]map[string]string, len(rows))
	for place, row := range rows {
		named[place] = map[string]string{}
		for column, name := range header {
			named[place][name] = strings.TrimSpace(row[column])
		}
	}
	return named, nil
}

// readFloat is the number a cell writes, or why it is not one.
func readFloat(text string) (float64, error) {
	value, err := strconv.ParseFloat(text, floatBits)
	if err != nil {
		return value, fmt.Errorf("%q is not a number", text)
	}
	return value, nil
}

// readUnits is the units a units file holds, or every reason it is broken.
func readUnits(raw []byte) (bookUnitsByName, error) {
	rows, err := readRows("book-units.csv", raw,
		[]string{"unit", "low", "high", "whole", "says"})
	units := bookUnitsByName{}
	for _, row := range rows {
		low, lowErr := readFloat(row["low"])
		high, highErr := readFloat(row["high"])
		whole := row["whole"] == "yes"
		if row["whole"] != "yes" && row["whole"] != "no" {
			err = errors.Join(err, fmt.Errorf("%s: whole is yes or no",
				row["unit"]))
		}
		err = errors.Join(err, lowErr, highErr)
		units[row["unit"]] = bookUnit{low: low, high: high, whole: whole}
	}
	return units, err
}

// readScalars is the single numbers of a numbers file by name, each judged
// against its unit. The numbers library has already refused a number with no
// name, no sentence, or no word on where it came from.
func readScalars(file string, raw []byte, units bookUnitsByName,
) (map[string]numbers.Number, error) {
	read, err := numbers.Read(file, raw)
	named := map[string]numbers.Number{}
	for _, number := range read {
		if refusal := units.judge(number.Unit, number.Value); refusal != nil {
			err = errors.Join(err, fmt.Errorf("%s: %s %w", file,
				number.Name, refusal))
		}
		named[number.Name] = number
	}
	return named, err
}

// readPoint is the three numbers of a row of a points file, each judged
// against the row's unit, or every reason the row is refused.
func readPoint(file string, row map[string]string, units bookUnitsByName,
) (maths.Vec, error) {
	var err error
	part := map[string]float64{}
	for _, axis := range []string{"x", "y", "z"} {
		value, readErr := readFloat(row[axis])
		err = errors.Join(err, readErr)
		if judged := units.judge(row["unit"], value); readErr == nil &&
			judged != nil {
			err = errors.Join(err, fmt.Errorf("%s: %s %s %w", file,
				row["name"], axis, judged))
		}
		part[axis] = value
	}
	if row["name"] == "" || row["from"] == "" || row["says"] == "" {
		err = errors.Join(err, fmt.Errorf("%s: a row needs a name, where "+
			"it came from, and what it says", file))
	}
	return maths.V(part["x"], part["y"], part["z"]), err
}

// bookPoints are the rows of a points file: a name that is written once is
// one point, and a name that is written again is a table of its rows in the
// order they are written.
type bookPoints struct {
	single map[string]maths.Vec
	tables map[string][]maths.Vec
}

// add holds the row of a name, as a point or as the next of a table.
func (held *bookPoints) add(name string, at maths.Vec) {
	rows, tabled := held.tables[name]
	first, single := held.single[name]
	switch {
	case tabled:
		held.tables[name] = append(rows, at)
	case single:
		delete(held.single, name)
		held.tables[name] = []maths.Vec{first, at}
	default:
		held.single[name] = at
	}
}

// readPoints is the points of a points file, or every reason it is broken.
func readPoints(file string, raw []byte, units bookUnitsByName,
) (bookPoints, error) {
	rows, err := readRows(file, raw, []string{"name", "x", "y", "z", "unit",
		"from", "says"})
	held := bookPoints{single: map[string]maths.Vec{},
		tables: map[string][]maths.Vec{}}
	for _, row := range rows {
		at, refusal := readPoint(file, row, units)
		err = errors.Join(err, refusal)
		held.add(row["name"], at)
	}
	return held, err
}

// A book is the numbers and points of one model, checked and held.
type book struct {
	scalars map[string]numbers.Number
	points  bookPoints
}

// readBook is the book of the numbers and points files of one model, or every
// reason it is broken, all together and with the file named.
func readBook(numbersFile string, numbersRaw []byte, pointsFile string,
	pointsRaw []byte,
) (*book, error) {
	if bookUnitsError != nil {
		return nil, bookUnitsError
	}
	scalars, numbersErr := readScalars(numbersFile, numbersRaw, bookUnits)
	points, pointsErr := readPoints(pointsFile, pointsRaw, bookUnits)
	read := &book{scalars: scalars, points: points}
	if err := errors.Join(numbersErr, pointsErr,
		read.twice(numbersFile, pointsFile)); err != nil {
		return nil, err
	}
	return read, nil
}

// twice is why a name is both a number and a point or a table, or nothing.
func (read *book) twice(numbersFile, pointsFile string) error {
	var err error
	for name := range read.scalars {
		_, point := read.points.single[name]
		_, table := read.points.tables[name]
		if point || table {
			err = errors.Join(err, fmt.Errorf("%s is written in both %s "+
				"and %s", name, numbersFile, pointsFile))
		}
	}
	return err
}

// asking is one making of a model: the names its code has asked for, and
// those the book did not hold. It is made once for each model made, from the
// book, which it does not change.
type asking struct {
	read    *book
	asked   map[string]bool
	missing []string
}

// asking is a fresh asking of the book.
func (read *book) asking() *asking {
	return &asking{read: read, asked: map[string]bool{}}
}

// number is the single number of a name. A name the book does not hold is
// kept to be told, and is nothing in the meantime.
func (ask *asking) number(name string) float64 {
	ask.asked[name] = true
	number, held := ask.read.scalars[name]
	if !held {
		ask.missing = append(ask.missing, name)
	}
	return number.Value
}

// count is a number that is whole, which the book has checked it is.
func (ask *asking) count(name string) int {
	return int(ask.number(name))
}

// point is the point of a name, or nothing and the name kept to be told.
func (ask *asking) point(name string) maths.Vec {
	ask.asked[name] = true
	at, held := ask.read.points.single[name]
	if !held {
		ask.missing = append(ask.missing, name)
	}
	return at
}

// table is the rows of a name written more than once, in order.
func (ask *asking) table(name string) []maths.Vec {
	ask.asked[name] = true
	rows, held := ask.read.points.tables[name]
	if !held {
		ask.missing = append(ask.missing, name)
	}
	return rows
}

// failed is whether a name has been asked for that the book does not hold.
func (ask *asking) failed() bool {
	return len(ask.missing) > spaceZero
}

// unasked is every name in the book that nothing has asked for, sorted.
func (ask *asking) unasked() []string {
	var names []string
	for name := range ask.read.scalars {
		names = append(names, name)
	}
	for name := range ask.read.points.single {
		names = append(names, name)
	}
	for name := range ask.read.points.tables {
		names = append(names, name)
	}
	names = slices.DeleteFunc(names, ask.wasAsked)
	sort.Strings(names)
	return names
}

// wasAsked is whether a name has been asked for.
func (ask *asking) wasAsked(name string) bool {
	return ask.asked[name]
}

// refusal is why the making cannot be trusted, or nothing: names the code
// asked for that the book lacks, and names the book holds that were not
// asked for.
func (ask *asking) refusal() error {
	lacked := listed("the code asks for names the data lacks", ask.missing)
	unasked := ask.unasked()
	spare := listed("the data holds names nothing asks for", unasked)
	return errors.Join(lacked, spare)
}

// listed says what a list of names is, or nothing if the list is empty.
func listed(what string, names []string) error {
	if len(names) == spaceZero {
		return nil
	}
	return fmt.Errorf("%s: %s", what, strings.Join(names, ", "))
}
