package movie

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// the files the tests read: the numbers they pick, and tables of cases
const (
	numbersFile  = "testdata/numbers.yaml"
	refusedFile  = "testdata/requests-refused.csv"
	takenFile    = "testdata/requests-taken.csv"
	movieName    = "movie.mp4"
	notMovieName = "notes.txt"
)

// chosen is the picked numbers as the file holds them, and why the file
// could not be read if it could not.
var chosen, unread = readChosen()

// readChosen reads the numbers file.
func readChosen() (numbers.Numbers, error) {
	raw, err := os.ReadFile(numbersFile)
	if err != nil {
		return numbers.Numbers{}, err
	}
	return numbers.Read(numbersFile, raw)
}

// pickedValue is the number of that name. An unknown name is a mistake
// in the tests; it stops them and says which.
func pickedValue(name string) float64 {
	number, err := chosen.Get(name)
	if err != nil {
		panic(errors.Join(unread, fmt.Errorf("movie: %w", err)))
	}
	return number.Value
}

// picked is the number of that name as a whole count.
func picked(name string) int {
	value := pickedValue(name)
	if value != math.Trunc(value) {
		panic(fmt.Errorf("movie: %s is %v, not a whole count", name, value))
	}
	return int(value)
}

// tableRow is one row of a table of cases, by the heading's names.
type tableRow map[string]string

// casesOf is the rows of a csv table of cases, or the test stops.
func casesOf(t *testing.T, name string) []tableRow {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	all, err := csv.NewReader(bytes.NewReader(text)).ReadAll()
	if err != nil || len(all) < 2 {
		t.Fatalf("%s: %v, %d rows", name, err, len(all))
	}
	var rows []tableRow
	for _, fields := range all[1:] {
		row := tableRow{}
		for place, column := range all[0] {
			row[column] = fields[place]
		}
		rows = append(rows, row)
	}
	return rows
}

// textIn is the text in a column, or the test stops.
func textIn(t *testing.T, row tableRow, column string) string {
	t.Helper()
	text, there := row[column]
	if !there {
		t.Fatalf("a row has no column called %q", column)
	}
	return text
}

// wholeIn is the whole number in a column, or the test stops.
func wholeIn(t *testing.T, row tableRow, column string) int {
	t.Helper()
	whole, err := strconv.Atoi(textIn(t, row, column))
	if err != nil {
		t.Fatal(err)
	}
	return whole
}

// truthIn is the true or false in a column, or the test stops.
func truthIn(t *testing.T, row tableRow, column string) bool {
	t.Helper()
	truth, err := strconv.ParseBool(textIn(t, row, column))
	if err != nil {
		t.Fatal(err)
	}
	return truth
}

// requestIn is the request a row of a table of cases describes, written
// to a fresh directory.
func requestIn(t *testing.T, row tableRow) Request {
	t.Helper()
	return Request{
		Path:   filepath.Join(t.TempDir(), movieName),
		Width:  wholeIn(t, row, "width"),
		Height: wholeIn(t, row, "height"),
		Rate:   wholeIn(t, row, "rate"),
		Frames: wholeIn(t, row, "frames"),
		Pad:    truthIn(t, row, "pad"),
	}
}

// smallRequest is a small movie in a fresh directory, with no sound.
func smallRequest(t *testing.T) Request {
	t.Helper()
	return Request{
		Path:   filepath.Join(t.TempDir(), movieName),
		Width:  picked("small-width"),
		Height: picked("small-height"),
		Rate:   picked("small-rate"),
		Frames: picked("small-frames"),
	}
}

// tone is a stereo sound of that many seconds: on the left a sine that
// starts at the test pitch and rises an octave a second, so no stretch
// of it repeats another; the right is a share of the left.
func tone(t *testing.T, rate int, seconds float64) Sound {
	t.Helper()
	count := int(seconds * float64(rate))
	loud := pickedValue("tone-loudness")
	share := pickedValue("right-share")
	start := pickedValue("tone-hertz")
	samples := make([]float32, 0, count*channels)
	for i := range count {
		at := float64(i) / float64(rate)
		turns := start*at + start*at*at/pair
		left := loud * math.Sin(2*math.Pi*turns)
		samples = append(samples, float32(left), float32(left*share))
	}
	sound, err := NewSound(rate, samples)
	if err != nil {
		t.Fatal(err)
	}
	return sound
}

// mustSettings is a checked request, or the test stops.
func mustSettings(t *testing.T, request Request) Settings {
	t.Helper()
	settings, err := NewSettings(request)
	if err != nil {
		t.Fatal(err)
	}
	return settings
}

// leftIn is the names left in the directory a path is in.
func leftIn(t *testing.T, path string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// nothingLeft stops the test when anything is left beside the path.
func nothingLeft(t *testing.T, path string) {
	t.Helper()
	if names := leftIn(t, path); len(names) != 0 {
		t.Fatalf("left behind: %v", names)
	}
}
