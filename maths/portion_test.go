package maths_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// portionRows are the rows of portions.csv for a call and an outcome. It
// stops if there are none, so that a test over them does not pass for want of
// rows.
func portionRows(test reporter, call, outcome string) table {
	test.Helper()
	var rows table
	for _, row := range read(test, portionsFile) {
		if row["call"] == call && row["outcome"] == outcome {
			rows = append(rows, row)
		}
	}
	if len(rows) == nought {
		test.Fatalf("%s: no %s rows that are %s", portionsFile, call, outcome)
	}
	return rows
}

// madePortion is the portion a row's call makes of the row's amount.
func madePortion(test reporter, row map[string]string) maths.Portion {
	test.Helper()
	amount := float(test, row, "amount")
	switch row["call"] {
	case "Percent":
		return maths.Percent(amount)
	case "Share":
		return maths.Share(amount)
	}
	test.Fatalf("%s: no call named %q", row["name"], row["call"])
	return maths.Portion{}
}

// saidPortions holds each said row of a call to its share, within the row's
// tolerance: nought is exact.
func saidPortions(test reporter, call string) {
	test.Helper()
	for _, row := range portionRows(test, call, "said") {
		got, err := madePortion(test, row).Share()
		want, tolerance := float(test, row, "share"), float(test, row, "tolerance")
		if err != nil || !(math.Abs(got-want) <= tolerance) {
			test.Fatalf("%s(%v) said %v, %v; want %v", call, row["amount"], got,
				err, want)
		}
	}
}

// refusedPortions holds each row of a call with an outcome to a complaint.
func refusedPortions(test reporter, call, outcome string) {
	test.Helper()
	for _, row := range portionRows(test, call, outcome) {
		if _, err := madePortion(test, row).Share(); err == nil {
			test.Fatalf("%s(%v) passed; %s", call, row["amount"], row["name"])
		}
	}
}

func TestPercentSaysPercents(test *testing.T) {
	saidPortions(test, "Percent")
}

func TestPercentNamesTheFixForAFraction(test *testing.T) {
	row := read(test, portionsFile).by(test, portionsFile, "half read as a fraction")
	_, err := madePortion(test, row).Share()
	if err == nil {
		test.Fatalf("%s(%v) passed", row["call"], row["amount"])
	}
	for _, fix := range strings.Split(row["says"], "|") {
		if !strings.Contains(err.Error(), fix) {
			test.Errorf("the complaint %q does not name %s", err, fix)
		}
	}
}

func TestPercentRefusesWhatReadsAsAFraction(test *testing.T) {
	refusedPortions(test, "Percent", "fraction")
}

func TestPercentRefusesWhatNoPortionIs(test *testing.T) {
	refusedPortions(test, "Percent", "refused")
}

func TestShareSaysFractions(test *testing.T) {
	saidPortions(test, "Share")
}

func TestShareRefusesOutsideTheWhole(test *testing.T) {
	refusedPortions(test, "Share", "refused")
}
