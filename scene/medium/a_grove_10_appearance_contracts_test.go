package medium

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The files the authored patch test reads: the weather, the two canopies and
// the record of densities.
const (
	aGrove10AppearanceContractsWeatherFile = "testdata/" +
		"a-grove-10-appearance-contracts-weather.csv"
	aGrove10AppearanceContractsCanopiesFile = "testdata/" +
		"a-grove-10-appearance-contracts-canopies.csv"
	aGrove10AppearanceContractsRecordFile = "testdata/grove-appearance.txt"
)

// aGrove10AppearanceContractsSlope is the ground the grove stands on: a level
// that rises with x and falls with z.
type aGrove10AppearanceContractsSlope struct{ level, acrossX, acrossZ float64 }

// height is the ground at x and z.
func (slope aGrove10AppearanceContractsSlope) height(x, z float64) float64 {
	return slope.level + slope.acrossX*x - slope.acrossZ*z
}

// aGrove10AppearanceContractsPoint is the place a row names in three
// columns, for x, y and z in that order.
func aGrove10AppearanceContractsPoint(t *testing.T, row map[string]string,
	columnX, columnY, columnZ string) maths.Vec {
	t.Helper()
	x := cell(t, row, columnX)
	y := cell(t, row, columnY)
	z := cell(t, row, columnZ)
	return maths.V(x, y, z)
}

// aGrove10AppearanceContractsWeatherRead is the authored weather.
func aGrove10AppearanceContractsWeatherRead(t *testing.T) GroveWeather {
	t.Helper()
	row := readSheet(t, aGrove10AppearanceContractsWeatherFile).row(t,
		aGrove10AppearanceContractsWeatherFile,
		"a-grove-10-appearance-contracts-authored")
	wind := aGrove10AppearanceContractsPoint(t, row, "windx", "windy", "windz")
	return GroveWeather{
		Extinction:  cell(t, row, "extinction"),
		Depth:       cell(t, row, "depth"),
		Coverage:    cell(t, row, "coverage"),
		Seed:        uint64(cellCount(t, row, "seed")),
		Wind:        wind,
		GroundFloor: cell(t, row, "floor"),
		Fade:        true,
	}
}

// aGrove10AppearanceContractsTreesRead is the two authored canopies.
func aGrove10AppearanceContractsTreesRead(t *testing.T) []Canopy {
	t.Helper()
	trees := []Canopy{}
	for _, name := range []string{"first", "second"} {
		row := readSheet(t, aGrove10AppearanceContractsCanopiesFile).row(t,
			aGrove10AppearanceContractsCanopiesFile,
			"a-grove-10-appearance-contracts-"+name)
		foot := aGrove10AppearanceContractsPoint(t, row, "x", "y", "z")
		trees = append(trees, Canopy{
			At:     foot,
			Radius: cell(t, row, "radius"),
			Height: cell(t, row, "height"),
		})
	}
	return trees
}

// aGrove10AppearanceContractsGrove is the authored grove: its weather, its
// canopies and the slope of its ground.
func aGrove10AppearanceContractsGrove(t *testing.T) *Grove {
	t.Helper()
	slope := aGrove10AppearanceContractsSlope{
		level:   picked(t, "a-grove-10-appearance-contracts-level"),
		acrossX: picked(t, "a-grove-10-appearance-contracts-across-x"),
		acrossZ: picked(t, "a-grove-10-appearance-contracts-across-z"),
	}
	grove, err := NewGrove(aGrove10AppearanceContractsWeatherRead(t),
		aGrove10AppearanceContractsTreesRead(t), slope.height)
	if err != nil {
		t.Fatal(err)
	}
	return grove
}

// aGrove10AppearanceContractsCheck reads one line of the record, a time, a
// place and the density authored there, and holds the grove to it.
func aGrove10AppearanceContractsCheck(t *testing.T, grove *Grove, line string,
	sample int) {
	t.Helper()
	var at time.Duration
	var x, y, z, want float64
	targets := []any{&at, &x, &y, &z, &want}
	if count, err := fmt.Sscan(line, targets...); err != nil ||
		count != len(targets) {
		t.Fatalf("sample %d: %v", sample, err)
	}
	got := grove.Tick(at).Density(maths.V(x, y, z))
	tolerance := picked(t, "a-grove-10-appearance-contracts-tolerance")
	if !(math.Abs(got-want) <= tolerance) {
		t.Fatalf("sample %d density %.17g, authored %.17g", sample, got, want)
	}
}

// TestAuthoredGrovePatchNoiseAtFixedCoordinates holds the grove to the
// densities authored for it at fixed times and places.
func TestAuthoredGrovePatchNoiseAtFixedCoordinates(t *testing.T) {
	grove := aGrove10AppearanceContractsGrove(t)
	file, err := os.Open(aGrove10AppearanceContractsRecordFile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	rows := nought
	for scan.Scan() {
		if strings.HasPrefix(scan.Text(), "#") {
			continue
		}
		aGrove10AppearanceContractsCheck(t, grove, scan.Text(), rows)
		rows++
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if rows != pickedCount(t, "a-grove-10-appearance-contracts-samples") {
		t.Fatalf("incomplete authored patch record: %d", rows)
	}
}
