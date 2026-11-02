package data

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// none is no count at all; it is defined as the count of nothing.
const none = 0

// errNoHeading is a table with nothing in it, not even a heading.
var errNoHeading = errors.New("ninjafilm: a table starts with a heading")

// Row is one row of a table. Its fields are asked for by the name its
// column has in the heading, never by place.
type Row struct{ fields map[string]string }

// Parse is the rows of a table of csv text after its heading. Every row
// has as many fields as the heading, or the table is refused.
func Parse(text string) ([]Row, error) {
	all, err := csv.NewReader(strings.NewReader(text)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ninjafilm: %w", err)
	}
	if len(all) == none {
		return nil, errNoHeading
	}
	var heading []string
	var rows []Row
	for _, fields := range all {
		if heading == nil {
			heading = fields
			continue
		}
		rows = append(rows, named(heading, fields))
	}
	return rows, nil
}

// named is a row from its fields and the heading that names them.
func named(heading, fields []string) Row {
	row := make(map[string]string, len(heading))
	for place, column := range heading {
		row[column] = fields[place]
	}
	return Row{row}
}

// Text is the field of a column, or why the row has none.
func (row Row) Text(column string) (text string, err error) {
	text, there := row.fields[column]
	if !there {
		return text, fmt.Errorf("ninjafilm: a row has no column "+
			"called %q", column)
	}
	return text, nil
}

// Float is the field of a column as a number. It is read as json reads
// a number. Json takes only what a number looks like. It refuses "Inf"
// and "NaN", which strconv would read. It refuses a number with
// anything after it.
func (row Row) Float(column string) (number float64, err error) {
	text, err := row.Text(column)
	if err != nil {
		return number, err
	}
	if json.Unmarshal([]byte(text), &number) != nil {
		return number, fmt.Errorf("ninjafilm: %q is not a number", text)
	}
	return number, nil
}

// Whole is the field of a column as a whole number.
func (row Row) Whole(column string) (whole int, err error) {
	text, err := row.Text(column)
	if err != nil {
		return whole, err
	}
	whole, err = strconv.Atoi(text)
	if err != nil {
		return whole, fmt.Errorf("ninjafilm: %q is not a whole number",
			text)
	}
	return whole, nil
}
