package section_test

import (
	"encoding/csv"
	"image"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// What the tests count in, each defined.
const (
	nothing = 0   // defined: no amount
	one     = 1   // defined: a single unit
	two     = 2   // defined: one and one
	percent = 100 // defined: the number of percent in a whole
	width64 = 64  // defined: the bits in the float64 a column is read as
)

// The files the tests read. Every number a test chooses is in one of
// them, and the tests ask for it by name.
const (
	numbersFile = "testdata/numbers.yaml"
	placesFile  = "testdata/places.csv"
	coloursFile = "testdata/colours.csv"
	splitsFile  = "testdata/splits.csv"
	windowsFile = "testdata/windows.csv"
)

// chosenNumbers is numbers.yaml, read once.
var chosenNumbers = sync.OnceValues(readNumbers)

// readNumbers reads numbers.yaml from disk.
func readNumbers() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return nil, err
	}
	return numbers.Read(numbersFile, raw)
}

// picked is a number someone chose, by its name in numbers.yaml.
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

// counted is a chosen number that is a whole number, by its name.
func counted(t *testing.T, name string) int {
	t.Helper()
	return int(picked(t, name))
}

// stream is a chosen number that names a stream of hashes.
func stream(t *testing.T, name string) uint64 {
	t.Helper()
	return uint64(picked(t, name))
}

// rows is the rows of a csv file, each by the names in its first row.
func rows(t *testing.T, file string) []map[string]string {
	t.Helper()
	opened, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	all, err := csv.NewReader(opened).ReadAll()
	if err != nil || len(all) == nothing {
		t.Fatalf("%s: %v, and no rows", file, err)
	}
	var out []map[string]string
	for _, row := range all[one:] {
		named := map[string]string{}
		for column, name := range all[nothing] {
			named[name] = row[column]
		}
		out = append(out, named)
	}
	return out
}

// float is the number in a column of a row.
func float(t *testing.T, each map[string]string, column string) float64 {
	t.Helper()
	got, err := strconv.ParseFloat(each[column], width64)
	if err != nil {
		t.Fatalf("%s of %q: %v", column, each["name"], err)
	}
	return got
}

// rect is the rectangle of a row of a csv file of rectangles.
func rect(t *testing.T, each map[string]string) image.Rectangle {
	t.Helper()
	left, top := float(t, each, "left"), float(t, each, "top")
	right, bottom := float(t, each, "right"), float(t, each, "bottom")
	return image.Rect(int(left), int(top), int(right), int(bottom))
}

// positive is whether an amount is above none.
func positive(amount float64) bool { return amount > nothing }

// eye is what the tests look through: a lens on a frame, from the
// origin.
type eye struct {
	lens       film.Sensor
	wide, high int
}

// lens is the pinhole every section test looks through.
func lens(t *testing.T) film.Pinhole {
	t.Helper()
	return film.Pinhole{Wide: picked(t, "lens-wide")}
}

// newEye is the eye through the lens, on the frame of the tests.
func newEye(t *testing.T) eye {
	t.Helper()
	return eye{lens: lens(t), wide: counted(t, "frame-wide"),
		high: counted(t, "frame-high")}
}

// look is the ray through a pixel of the frame.
func (view eye) look(x, y int) maths.Ray {
	var band maths.Band
	return film.Look(maths.Local(), view.lens, x, y, view.wide, view.high, band)
}
