package biome_test

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/swatches"
)

const (
	nought     = 0  // defined: no amount
	unity      = 1  // defined: one
	floatWidth = 64 // defined: the bits in the float64 a column is read as
	wholeWidth = 0  // defined: strconv's width for an int of the machine's size
	decimal    = 10 // defined: the base whole numbers are written in
)

const (
	numbersFile = "testdata/numbers.yaml"
	bandsFile   = "testdata/bands.csv"
	glowsFile   = "testdata/glows.csv"
	goldenFile  = "testdata/golden/world.txt"
)

// reporter is what a test and a property test both do: say where a
// helper is, and stop or go on with a finding.
type reporter interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
}

// table is a csv file's rows, each by its header's names.
type table []map[string]string

// loading is a csv file to be read once.
type loading string

// load reads the file.
func (file loading) load() (table, error) { return load1(string(file)) }

// tables is every csv file, read once.
var (
	tablesLock sync.Mutex
	tables     = map[string]func() (table, error){}
)

// read is a csv file's rows, or a stop naming the file.
func read(test reporter, file string) table {
	test.Helper()
	tablesLock.Lock()
	load, ok := tables[file]
	if !ok {
		load = sync.OnceValues(loading(file).load)
		tables[file] = load
	}
	tablesLock.Unlock()
	rows, err := load()
	if err != nil {
		test.Fatal(err)
	}
	return rows
}

// load1 reads a csv file from disk.
func load1(file string) (table, error) {
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
	header, rows := all[nought], all[unity:]
	out := make(table, len(rows))
	for place, row := range rows {
		out[place] = map[string]string{}
		for column, name := range header {
			out[place][name] = row[column]
		}
	}
	return out, nil
}

// by is the row of a name, or a stop naming the file.
func (rows table) by(test reporter, file, name string) map[string]string {
	test.Helper()
	for _, row := range rows {
		if row["name"] == name {
			return row
		}
	}
	test.Fatalf("%s: no row called %q", file, name)
	return nil
}

// float is the number in a column of a row, or a stop.
func float(test reporter, row map[string]string, column string) float64 {
	test.Helper()
	got, err := strconv.ParseFloat(row[column], floatWidth)
	if err != nil {
		test.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// whole is the whole number in a column of a row, or a stop.
func whole(test reporter, row map[string]string, column string) int {
	test.Helper()
	got, err := strconv.ParseInt(row[column], decimal, wholeWidth)
	if err != nil {
		test.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return int(got)
}

// readNumbers reads numbers.yaml.
func readNumbers() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// chosen is numbers.yaml, read once.
var chosen = sync.OnceValues(readNumbers)

// picked is a number someone chose, by its name in numbers.yaml.
func picked(test reporter, name string) float64 {
	test.Helper()
	all, err := chosen()
	if err != nil {
		test.Fatal(err)
	}
	number, err := all.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return number.Value
}

// glow is a named colour of glows.csv.
func glow(test reporter, name string) maths.Light {
	test.Helper()
	row := read(test, glowsFile).by(test, glowsFile, name)
	return maths.RGB(float(test, row, "r"), float(test, row, "g"),
		float(test, row, "b"))
}

// dark is no light at all.
func dark() maths.Light { return maths.RGB(nought, nought, nought) }

// terran is the light of one of swatches.Terran's paints, by its place.
func terran(place int) maths.Light {
	return swatches.Light(swatches.Terran[place])
}

// the places in swatches.Terran of the paints a test names
const (
	deepSea    = 0  // defined: swatches.Terran's first, the deep sea
	shallowSea = 1  // defined: swatches.Terran's second, the shallow sea
	rock       = 9  // defined: swatches.Terran's tenth, rock
	snow       = 10 // defined: swatches.Terran's eleventh, snow
	lava       = 11 // defined: swatches.Terran's twelfth, lava
	ash        = 12 // defined: swatches.Terran's last, ash
)

// flat is the test map's tiling.
func flat(test reporter) hexes.Flat {
	test.Helper()
	//ninja:vow stranger hexes makes no tiling; a Flat is a plain value
	return hexes.Flat{Step: picked(test, "step"),
		Across: int(picked(test, "across")), Down: int(picked(test, "down"))}
}

// fields is the test world's fields under a sea, with no sprawl.
func fields(test reporter, sea float64) biome.Fields {
	test.Helper()
	return biome.Fields{Tiling: flat(test),
		Seed:  uint64(picked(test, "seed")),
		Scale: picked(test, "scale"), Sea: sea}
}

// above is how high a tile's middle stands above the sea of its fields, less
// than zero under it.
func above(made biome.Fields, tile hexes.Tile) float64 {
	return made.Height(made.Tiling.Centre(tile)) - made.Sea
}

// wet is the wet at a tile's middle.
func wet(made biome.Fields, tile hexes.Tile) float64 {
	return made.Wet(made.Tiling.Centre(tile))
}

// water is whether a paint is the sea's, deep or shallow, which is also
// a river's.
func water(paint maths.Light) bool {
	return paint == terran(deepSea) || paint == terran(shallowSea)
}

// lowest is a tile's lowest neighbour, and whether any is lower than the
// tile itself.
func lowest(made biome.Fields, tile hexes.Tile) (hexes.Tile, bool) {
	low, lower := tile, false
	for _, next := range made.Tiling.Neighbours(tile) {
		if above(made, next) < above(made, low) {
			low, lower = next, true
		}
	}
	return low, lower
}
