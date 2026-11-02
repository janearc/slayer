// Package shelf assembles the film stocks of the hills from stock and record
// tables. Each stock has one row and one or more associated record rows. See
// data/SOURCES.md for the provenance and verification status of values; not
// every parameter is a direct measurement.
package shelf

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/lay"
	"github.com/janearc/slayer/ninjafilm/record"
	"github.com/janearc/slayer/ninjafilm/roll"
	"github.com/janearc/slayer/ninjafilm/span"
)

// none is no count at all, defined as zero.
const none = 0

// what a table is refused for
var (
	errTwice = errors.New("ninjafilm: a stock is on the shelf twice")
	errBare  = errors.New("ninjafilm: a stock has no records")
)

// Tables is a shelf read from the text of its two tables each time it
// is asked. So a table which will not read is said and not hidden.
type Tables struct{ stocks, records, gauges string }

// Of is the shelf the text of a stocks table, a records table and a table
// of gauges make.
func Of(stocks, records, gauges string) Tables {
	return Tables{stocks, records, gauges}
}

// Names is the name of every stock on the shelf, in order. Find is
// asked for a name.
func (tables Tables) Names() ([]string, error) {
	found, err := read(tables.stocks, tables.records, tables.gauges)
	if err != nil {
		return nil, err
	}
	return slices.Sorted(maps.Keys(found)), nil
}

// Find is the stock of a name. Names not on the shelf are refused, and
// the refusal lists the names that are.
func (tables Tables) Find(name string) (stock roll.Stock, err error) {
	found, err := read(tables.stocks, tables.records, tables.gauges)
	if err != nil {
		return stock, err
	}
	stock, known := found[name]
	if !known {
		names := slices.Sorted(maps.Keys(found))
		return stock, fmt.Errorf("ninjafilm: no stock is called "+
			"%q; the shelf holds %q", name, names)
	}
	return stock, nil
}

// drafted is a stock with its records still being gathered.
type drafted struct {
	gauge   film.Gauge
	laid    lay.Lay
	records []record.Record
}

// stockOf is a stock with no records yet, from a row of the stocks
// table. The row names its gauge and its lay.
func stockOf(row data.Row, gauges film.Table) (drafted, error) {
	name, err := row.Text("gauge")
	if err != nil {
		return drafted{}, err
	}
	gauge, err := gauges.Named(name)
	if err != nil {
		return drafted{}, err
	}
	laid, err := layOf(row)
	if err != nil {
		return drafted{}, err
	}
	return drafted{gauge: gauge.Grainless(), laid: laid}, nil
}

// layOf is the lay a row of the stocks table names. It is made from the
// two numbers of the row.
func layOf(row data.Row) (lay.Lay, error) {
	kind, err := row.Text("lay")
	if err != nil {
		return nil, err
	}
	perInch, err := row.Float("per_square_inch")
	if err != nil {
		return nil, err
	}
	passes, err := row.Float("passes")
	if err != nil {
		return nil, err
	}
	return lay.New(kind, perInch, passes)
}

// spanOf is the span a row names by the columns that hold its ends.
func spanOf(row data.Row, from, to string) (made span.Span, err error) {
	start, err := row.Whole(from)
	if err != nil {
		return made, err
	}
	end, err := row.Whole(to)
	if err != nil {
		return made, err
	}
	return span.New(start, end)
}

// toneOf is the curve a row names by its slope, least and most.
func toneOf(row data.Row) (made curve.Curve, err error) {
	slope, err := row.Float("slope")
	if err != nil {
		return made, err
	}
	least, err := row.Float("least")
	if err != nil {
		return made, err
	}
	most, err := row.Float("most")
	if err != nil {
		return made, err
	}
	return curve.New(least, most, slope)
}

// recordOf is a record from a row of the records table. The row names
// the span the record sees and the span its image blocks. It also gives
// the width of its crystal. The slope, least and most of its curve come
// last.
func recordOf(row data.Row) (made record.Record, err error) {
	sees, err := spanOf(row, "sees_from", "sees_to")
	if err != nil {
		return made, err
	}
	blocks, err := spanOf(row, "image_from", "image_to")
	if err != nil {
		return made, err
	}
	across, err := row.Float("crystal_um")
	if err != nil {
		return made, err
	}
	grain, err := crystal.New(across)
	if err != nil {
		return made, err
	}
	tone, err := toneOf(row)
	if err != nil {
		return made, err
	}
	return record.New(sees.Holds(), blocks.Holds(), grain, tone)
}

// read is every stock by its name from the two tables. It says what is
// wrong with them if anything is.
func read(stocks, records, gauges string) (map[string]roll.Stock,
	error) {
	cut := film.From(gauges)
	found := map[string]drafted{}
	listed, err := data.Parse(stocks)
	if err != nil {
		return nil, err
	}
	for _, row := range listed {
		made, err := stockOf(row, cut)
		if err != nil {
			return nil, err
		}
		name, err := row.Text("name")
		if err != nil {
			return nil, err
		}
		if _, twice := found[name]; twice {
			return nil, errTwice
		}
		found[name] = made
	}
	return stocked(found, records)
}

// stocked is the shelf with every record put in its stock. It says what
// is wrong with the records table if anything is.
func stocked(found map[string]drafted, records string) (
	map[string]roll.Stock, error) {
	listed, err := data.Parse(records)
	if err != nil {
		return nil, err
	}
	for _, row := range listed {
		made, err := recordOf(row)
		if err != nil {
			return nil, err
		}
		name, err := row.Text("stock")
		if err != nil {
			return nil, err
		}
		holds, known := found[name]
		if !known {
			return nil, fmt.Errorf("ninjafilm: a record is for "+
				"%q, which is not on the shelf", name)
		}
		holds.records = append(holds.records, made)
		found[name] = holds
	}
	return shelved(found)
}

// shelved is every drafted stock as a stock. It refuses the first that
// has no records.
func shelved(found map[string]drafted) (map[string]roll.Stock, error) {
	shelf := map[string]roll.Stock{}
	for name, holds := range found {
		if len(holds.records) == none {
			return nil, errBare
		}
		shelf[name] = roll.NewStock(holds.gauge, holds.laid,
			holds.records)
	}
	return shelf, nil
}

// Tables are what a roll is loaded from.
var _ roll.Shelf = Tables{}
