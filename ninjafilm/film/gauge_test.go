package film_test

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0  // defined: no amount at all
	whole    = 1  // defined: one
	upward   = 1  // defined: math.Inf's code for infinity above all numbers
	downward = -1 // defined: math.Inf's code for infinity below all numbers
	either   = 0  // defined: math.IsInf's code for infinity of either sign
	shortest = -1 // defined: strconv's code for the fewest digits that read back exactly
	wordBits = 64 // defined: a float64 is 64 bits wide
)

// the tables the tests are run over
const (
	knownFile   = "testdata/known.csv"
	refusedFile = "testdata/refused.csv"
)

// what these tests choose, each picked in picked-numbers.yml
var (
	sizeLeast  = data.Picked("film-size-least")
	sizeMost   = data.Picked("film-size-most")
	grainMost  = data.Picked("film-grain-most")
	oddLeast   = data.Picked("film-odd-least")
	oddMost    = data.Picked("film-odd-most")
	gaugesMost = data.Whole("film-gauges-most")
)

// table is the rows of a table of cases in testdata, or the test stops.
func table(t *testing.T, name string) []data.Row {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// number is the number in a column of a row, or the test stops.
func number(t *testing.T, row data.Row, column string) float64 {
	t.Helper()
	amount, err := row.Float(column)
	if err != nil {
		t.Fatal(err)
	}
	return amount
}

// formatted is a number as the shortest text that reads back as it.
func formatted(amount float64) string {
	return strconv.FormatFloat(amount, 'g', shortest, wordBits)
}

// odd draws an ordinary number or an odd one.
func odd() *rapid.Generator[float64] {
	strange := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), none})
	return rapid.OneOf(rapid.Float64Range(oddLeast, oddMost), strange)
}

// sized is whether a number is greater than zero and finite.
func sized(amount float64) bool {
	return amount > none && !math.IsInf(amount, either)
}

// Whether a gauge is made follows from its numbers. It has a name. Its width
// and height are greater than zero and finite. Its grain is zero or more and
// finite.
func TestAGaugeIsMadeExactlyWhenItCanBe(t *testing.T) {
	rapid.Check(t, gaugeIsMadeWhenItCan)
}

func gaugeIsMadeWhenItCan(t *rapid.T) {
	name := rapid.SampledFrom([]string{"", "a chip"}).Draw(t, "name")
	width, height := odd().Draw(t, "width"), odd().Draw(t, "height")
	grain := odd().Draw(t, "grain")
	want := name != "" && sized(width) && sized(height) &&
		grain >= none && !math.IsInf(grain, either)
	made, err := film.NewGauge(name, width, height, grain)
	if (err == nil) != want {
		t.Fatalf("a gauge %q %v by %v with grain %v: made is %v, want %v",
			name, width, height, grain, err == nil, want)
	}
	if err == nil && (made.Name() != name || made.Width() != width ||
		made.Height() != height || made.Grain() != grain) {
		t.Fatalf("a gauge came back as %v", made)
	}
}

// anti-tests: a gauge with no size is not made. Nor is one with a grain less
// than zero. What comes back is the zero gauge.
func TestAGaugeRefusesWhatItCannotBe(t *testing.T) {
	var zero film.Gauge
	for _, row := range table(t, refusedFile) {
		name, _ := row.Text("name")
		width, height := number(t, row, "width"), number(t, row, "height")
		grain := number(t, row, "grain")
		got, err := film.NewGauge(name, width, height, grain)
		if err == nil || got != zero {
			label, _ := row.Text("label")
			t.Errorf("%s was made: %v, %v", label, got, err)
		}
	}
	if got, err := film.NewGauge("", whole, whole, none); err == nil ||
		got != zero {
		t.Errorf("a gauge with no name was made: %v, %v", got, err)
	}
}

// Frames are as wide against as tall as their sides say. Zero gauges
// with no height to speak of are square.
func TestAGaugeIsAsWideAsItsSidesSay(t *testing.T) {
	rapid.Check(t, aspectIsWidthOverHeight)
	var zero film.Gauge
	if got := zero.Aspect(); got != whole {
		t.Errorf("a gauge of no height has the aspect %v", got)
	}
}

func aspectIsWidthOverHeight(t *rapid.T) {
	width := rapid.Float64Range(sizeLeast, sizeMost).Draw(t, "width")
	height := rapid.Float64Range(sizeLeast, sizeMost).Draw(t, "height")
	grain := rapid.Float64Range(none, grainMost).Draw(t, "grain")
	made, err := film.NewGauge("drawn", width, height, grain)
	if err != nil {
		t.Fatal(err)
	}
	if got := made.Aspect(); got != width/height {
		t.Fatalf("%v by %v has the aspect %v", width, height, got)
	}
	bare := made.Grainless()
	if bare.Grain() != none || bare.Name() != made.Name() ||
		bare.Width() != width || bare.Height() != height {
		t.Fatalf("a gauge without its grain is %v, from %v", bare, made)
	}
	if made.Grain() != grain {
		t.Fatal("taking the grain off a gauge changed the gauge")
	}
}

// drawnTable is the text of a table of gauges, and the gauges in it.
func drawnTable(t *rapid.T) (text string, names []string,
	made map[string]film.Gauge) {
	made = map[string]film.Gauge{}
	text = "name,width_mm,height_mm,grain,source\n"
	many := rapid.IntRange(whole, gaugesMost).Draw(t, "gauges")
	for place := range many {
		name := "g" + strconv.Itoa(place)
		width := rapid.Float64Range(sizeLeast, sizeMost).Draw(t, "width")
		height := rapid.Float64Range(sizeLeast, sizeMost).Draw(t, "height")
		grain := rapid.Float64Range(none, grainMost).Draw(t, "grain")
		gauge, err := film.NewGauge(name, width, height, grain)
		if err != nil {
			t.Fatal(err)
		}
		made[name] = gauge
		names = append(names, name)
		text += strings.Join([]string{name, formatted(width),
			formatted(height), formatted(grain), "x"}, ",") + "\n"
	}
	return text, names, made
}

// Every gauge in a table is the gauge its row says, found by its name.
func TestATableGivesEachGaugeItsRowSays(t *testing.T) {
	rapid.Check(t, tableGivesItsGauges)
}

func tableGivesItsGauges(t *rapid.T) {
	text, names, made := drawnTable(t)
	gauges := film.From(text)
	for _, name := range names {
		got, err := gauges.Named(name)
		if err != nil || got != made[name] {
			t.Fatalf("%s is %v, %v, want %v", name, got, err, made[name])
		}
	}
}

// anti-tests: names not in the table are refused, and the refusal lists
// the names that are. Tables that will not read say so too.
func TestAGaugeNotInTheTableIsRefusedWithTheNamesThatAre(t *testing.T) {
	text, names, _ := drawnForOnce(t)
	got, err := film.From(text).Named("celluloid")
	var zero film.Gauge
	if err == nil || got != zero ||
		!strings.Contains(err.Error(), "celluloid") {
		t.Fatalf("a gauge not in the table was found: %v, %v", got, err)
	}
	for _, name := range names {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the refusal %q does not say %s", err, name)
		}
	}
	_, err = film.From("").Named("any")
	if err == nil || !strings.Contains(err.Error(), "heading") {
		t.Errorf("a table with no heading was read: %v", err)
	}
}

// drawnForOnce is a table of gauges for a test that is not a property.
func drawnForOnce(t *testing.T) (text string, names []string,
	made map[string]film.Gauge) {
	t.Helper()
	text = "name,width_mm,height_mm,grain,source\n" +
		"one,10,5,0.001,x\ntwo,20,10,0.002,x\n"
	return text, []string{"one", "two"}, nil
}

// anti-tests: a table without a column the gauge needs is refused. So is
// one with a number that is not one. The refusal names the column.
func TestATableThatIsSpoiledIsRefusedByColumn(t *testing.T) {
	for _, column := range []string{"name", "width_mm", "height_mm",
		"grain"} {
		heading := strings.Replace("name,width_mm,height_mm,grain\n",
			column, "not_"+column, whole)
		_, err := film.From(heading + "g,1,1,0\n").Named("g")
		if err == nil || !strings.Contains(err.Error(), column) {
			t.Errorf("a table without %s was read: %v", column, err)
		}
	}
	for _, spoiled := range []string{"g,wide,1,0", "g,1,tall,0",
		"g,1,1,coarse"} {
		_, err := film.From("name,width_mm,height_mm,grain\n" + spoiled +
			"\n").Named("g")
		if err == nil || !strings.Contains(err.Error(), "not a number") {
			t.Errorf("%q was read as a gauge: %v", spoiled, err)
		}
	}
}

// The gauges as they ship are the gauges the audited copy had.
func TestTheShippedGaugesAreTheOnesTheyAlwaysWere(t *testing.T) {
	shipped := film.From(data.GaugesTable())
	for _, row := range table(t, knownFile) {
		name, _ := row.Text("name")
		got, err := shipped.Named(name)
		if err != nil {
			t.Fatal(err)
		}
		if got.Width() != number(t, row, "width_mm") ||
			got.Height() != number(t, row, "height_mm") ||
			got.Grain() != number(t, row, "grain") {
			t.Errorf("%s is %v", name, got)
		}
	}
}

// stopped says the test failed if the program it was deferred from did
// not stop.
func stopped(t *testing.T) {
	t.Helper()
	if recover() == nil {
		t.Error("a table without the gauge did not stop the program")
	}
}

// Gauges the code leans on are found in the table that ships, and a
// table without one stops the program.
func TestMustFindsTheGaugeOrStopsTheProgram(t *testing.T) {
	if got := film.Shipped().Must("super 35"); got.Name() != "super 35" {
		t.Errorf("the gauge found is %v", got)
	}
	defer stopped(t)
	film.From("name,width_mm,height_mm,grain\n").Must("super 35")
}
