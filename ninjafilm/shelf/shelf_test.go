package shelf_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/lay"
	"github.com/janearc/slayer/ninjafilm/record"
	"github.com/janearc/slayer/ninjafilm/roll"
	"github.com/janearc/slayer/ninjafilm/shelf"
	"github.com/janearc/slayer/ninjafilm/span"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0  // defined: no amount at all
	shortest = -1 // defined: strconv's code for the fewest digits that read back exactly
	wordBits = 64 // defined: a float64 is 64 bits wide
	once     = 1  // defined: a replacement made once
	whole    = 1  // defined: the whole of a share, all the light
)

// the tables the tests start from, and the table of what is refused
const (
	stocksFile    = "testdata/stocks.csv"
	recordsFile   = "testdata/records.csv"
	gaugesFile    = "testdata/gauges.csv"
	refusalsFile  = "testdata/refusals.csv"
	stockColumns  = "name gauge lay per_square_inch passes"
	recordColumns = "stock sees_from sees_to image_from image_to " +
		"crystal_um slope least most"
)

// what these tests choose, each picked in picked-numbers.yml
var (
	stocksLeast   = data.Whole("shelf-stocks-least")
	stocksMost    = data.Whole("shelf-stocks-most")
	recordsLeast  = data.Whole("shelf-records-least")
	recordsMost   = data.Whole("shelf-records-most")
	widthLeast    = data.Picked("shelf-width-least")
	widthMost     = data.Picked("shelf-width-most")
	floorMost     = data.Picked("shelf-floor-most")
	roomLeast     = data.Picked("shelf-room-least")
	roomMost      = data.Picked("shelf-room-most")
	slopeLeast    = data.Picked("shelf-slope-least")
	slopeMost     = data.Picked("shelf-slope-most")
	elementsLeast = data.Picked("shelf-elements-least")
	elementsMost  = data.Picked("shelf-elements-most")
	passesLeast   = data.Picked("shelf-passes-least")
)

// gaugeNames are the gauges the table of gauges holds.
func gaugeNames() []string {
	return []string{"super 8", "16 mm", "academy", "super 35", "full frame",
		"65 mm", "imax"}
}

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// read is the text of a file in testdata, or the test stops.
func read(t stopper, name string) string {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// formatted is a number as the shortest text that reads back as it.
func formatted(number float64) string {
	return strconv.FormatFloat(number, 'g', shortest, wordBits)
}

func TestTheShelfNamesItsStocksInOrder(t *testing.T) {
	tables := shelf.Of(read(t, stocksFile), read(t, recordsFile),
		read(t, gaugesFile))
	names, err := tables.Names()
	if want := []string{"one", "two"}; err != nil ||
		!slices.Equal(names, want) {
		t.Errorf("the shelf holds %v, %v, want %v", names, err, want)
	}
}

// names not on the shelf are refused, and the refusal lists the names
// that are
func TestAStockNotOnTheShelfIsRefusedWithTheNamesThatAre(t *testing.T) {
	tables := shelf.Of(read(t, stocksFile), read(t, recordsFile),
		read(t, gaugesFile))
	var unfound roll.Stock
	got, err := tables.Find("celluloid")
	if err == nil || !reflect.DeepEqual(got, unfound) {
		t.Fatalf("a stock not on the shelf was found: %v, %v", got, err)
	}
	for _, word := range []string{"celluloid", `"one"`, `"two"`} {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("the refusal %q does not say %s", err, word)
		}
	}
}

// ends are the two ends of a span that holds a lamp. They lie between
// the least the observer sees and the most. The first is at or before
// the second. Spans drawn with no lamp in them are widened to the whole
// of what is seen.
func ends(t *rapid.T) (from, to int) {
	least, most := swatch.Observed()
	from = rapid.IntRange(least, most).Draw(t, "from")
	to = rapid.IntRange(from, most).Draw(t, "to")
	var dark maths.Light
	drawn, err := span.New(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if drawn.Holds() == dark {
		return least, most
	}
	return from, to
}

// drawnRecord is a record a film could have: its row of the records
// table, and the record that row means.
func drawnRecord(t *rapid.T, name string) (row string, made record.Record) {
	from, to := ends(t)
	imageFrom, imageTo := ends(t)
	across := rapid.Float64Range(widthLeast, widthMost).Draw(t, "across")
	least := rapid.Float64Range(none, floorMost).Draw(t, "least")
	room := rapid.Float64Range(roomLeast, roomMost).Draw(t, "room")
	slope := rapid.Float64Range(slopeLeast, slopeMost).Draw(t, "slope")
	window, err := span.New(from, to)
	if err != nil {
		t.Fatal(err)
	}
	image, err := span.New(imageFrom, imageTo)
	if err != nil {
		t.Fatal(err)
	}
	grain, err := crystal.New(across)
	if err != nil {
		t.Fatal(err)
	}
	tone, err := curve.New(least, least+room, slope)
	if err != nil {
		t.Fatal(err)
	}
	made, err = record.New(window.Holds(), image.Holds(), grain, tone)
	if err != nil {
		t.Fatal(err)
	}
	row = strings.Join([]string{name, strconv.Itoa(from),
		strconv.Itoa(to), strconv.Itoa(imageFrom), strconv.Itoa(imageTo),
		formatted(across), formatted(slope), formatted(least),
		formatted(least + room), "x"}, ",")
	return row, made
}

// drawnLay is a lay a film could have: its three fields of the stocks
// table, and the lay they mean.
func drawnLay(t *rapid.T) (fields string, laid lay.Lay) {
	if rapid.Bool().Draw(t, "mosaic") {
		perInch := rapid.Float64Range(elementsLeast,
			elementsMost).Draw(t, "perInch")
		passes := rapid.Float64Range(passesLeast, whole).Draw(t, "passes")
		laid = mustLay(t, "mosaic", perInch, passes)
		return "mosaic," + formatted(perInch) + "," + formatted(passes),
			laid
	}
	return "stacked,0,0", mustLay(t, "stacked", none, none)
}

// mustLay is a lay of a name, or the test stops.
func mustLay(t *rapid.T, name string, perInch, passes float64) lay.Lay {
	made, err := lay.New(name, perInch, passes)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// drawnShelf is the text of the two tables of a shelf a film maker
// could stock, and the stocks they mean by name.
func drawnShelf(t *rapid.T) (stocks, records string,
	want map[string]roll.Stock) {
	want = map[string]roll.Stock{}
	stocks = "name,gauge,lay,per_square_inch,passes,source\n"
	records = "stock,sees_from,sees_to,image_from,image_to,crystal_um," +
		"slope,least,most,source\n"
	count := rapid.IntRange(stocksLeast, stocksMost).Draw(t, "stocks")
	for place := range count {
		name := "s" + strconv.Itoa(place)
		called := rapid.SampledFrom(gaugeNames()).Draw(t, "gauge")
		gauge, err := film.From(read(t, gaugesFile)).Named(called)
		if err != nil {
			t.Fatal(err)
		}
		fields, laid := drawnLay(t)
		stocks += strings.Join([]string{name, called, fields, "x"},
			",") + "\n"
		gauge = gauge.Grainless()
		var made []record.Record
		many := rapid.IntRange(recordsLeast, recordsMost).Draw(t, "many")
		for range many {
			row, each := drawnRecord(t, name)
			records += row + "\n"
			made = append(made, each)
		}
		want[name] = roll.NewStock(gauge, laid, made)
	}
	return stocks, records, want
}

// The shelf is every stock the tables name. Each is cut to the gauge the
// table of gauges has, without the grain of the gauge. It is laid as the
// row says and has every record of the stock in the order the table has
// them.
func TestTheShelfHoldsWhatItsTablesSay(t *testing.T) {
	rapid.Check(t, shelfHoldsWhatItIsTold)
}

func shelfHoldsWhatItIsTold(t *rapid.T) {
	stocks, records, want := drawnShelf(t)
	tables := shelf.Of(stocks, records, read(t, gaugesFile))
	names, err := tables.Names()
	if err != nil || len(names) != len(want) {
		t.Fatalf("the shelf holds %v, %v, want %d", names, err, len(want))
	}
	for name, stock := range want {
		got, err := tables.Find(name)
		if err != nil || !reflect.DeepEqual(got, stock) {
			t.Fatalf("%s is %v, %v, want %v", name, got, err, stock)
		}
	}
}

// replaced is a table's text with the first of one thing in it put
// right, or the test stops.
func replaced(t testing.TB, text, find, with string) string {
	t.Helper()
	if !strings.Contains(text, find) {
		t.Fatalf("%q is not in the table", find)
	}
	return strings.Replace(text, find, with, once)
}

// anti-tests: every row of the table of refusals is the shipped sample
// spoiled in one place. The shelf refuses it, and says why.
func TestAShelfThatIsSpoiledIsRefusedForTheReason(t *testing.T) {
	rows, err := data.Parse(read(t, refusalsFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		spoiled, ok := spoil(t, row)
		if !ok {
			continue
		}
		names, err := spoiled.Names()
		label, _ := row.Text("label")
		says, _ := row.Text("says")
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: %v, %v, was not refused for %q", label, names,
				err, says)
		}
	}
}

// spoil is the sample shelf with a row of the refusals put into it.
func spoil(t *testing.T, row data.Row) (shelf.Tables, bool) {
	t.Helper()
	stocks, records := read(t, stocksFile), read(t, recordsFile)
	table, _ := row.Text("table")
	find, _ := row.Text("find")
	with, _ := row.Text("replace")
	if table == "stocks" {
		stocks = replaced(t, stocks, find, with)
	} else {
		records = replaced(t, records, find, with)
	}
	return shelf.Of(stocks, records, read(t, gaugesFile)), true
}

// anti-tests: no text at all for either table is not a shelf.
func TestAShelfWithATableMissingIsRefused(t *testing.T) {
	stocks, records := read(t, stocksFile), read(t, recordsFile)
	gauges := read(t, gaugesFile)
	for name, tables := range map[string]shelf.Tables{
		"no stocks table":  shelf.Of("", records, gauges),
		"no records table": shelf.Of(stocks, "", gauges),
		"no gauges table":  shelf.Of(stocks, records, ""),
	} {
		names, err := tables.Names()
		if err == nil || !strings.Contains(err.Error(), "heading") {
			t.Errorf("%s: the shelf held %v, %v", name, names, err)
		}
		got, err := tables.Find("one")
		if err == nil || !strings.Contains(err.Error(), "heading") {
			t.Errorf("%s: found %v, %v", name, got, err)
		}
	}
}

// without is a table's text with a column taken out of it.
func without(t testing.TB, text, column string) string {
	t.Helper()
	all, err := csv.NewReader(strings.NewReader(text)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writer := csv.NewWriter(&out)
	var gone int
	for place, row := range all {
		if place == none {
			gone = slices.Index(row, column)
		}
		kept := slices.Delete(slices.Clone(row), gone, gone+once)
		if err := writer.Write(kept); err != nil {
			t.Fatal(err)
		}
	}
	writer.Flush()
	return out.String()
}

// anti-tests: a table whose heading does not name a column the shelf
// reads is refused, and the refusal names the column.
func TestAShelfWithAColumnMissingIsRefusedByName(t *testing.T) {
	stocks, records := read(t, stocksFile), read(t, recordsFile)
	gauges := read(t, gaugesFile)
	for _, column := range strings.Fields(stockColumns) {
		tables := shelf.Of(without(t, stocks, column), records, gauges)
		checkNamed(t, tables, column)
	}
	for _, column := range strings.Fields(recordColumns) {
		tables := shelf.Of(stocks, without(t, records, column), gauges)
		checkNamed(t, tables, column)
	}
}

// checkNamed stops the test unless a shelf is refused for a column.
func checkNamed(t *testing.T, tables shelf.Tables, column string) {
	t.Helper()
	_, err := tables.Names()
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%q", column)) {
		t.Errorf("without %s the shelf was refused with %v", column, err)
	}
}
