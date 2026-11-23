package movie

import (
	"slices"
	"strings"
	"testing"
)

// the embedded table reads, starts at 3.0 and runs past 5.2, so a
// refusal can name the level a request would need.
func TestTheLevelTableReads(t *testing.T) {
	rows, err := readLevels(levelTable)
	if err != nil {
		t.Fatal(err)
	}
	lowest := slices.MinFunc(rows, compareLevels)
	highest := slices.MaxFunc(rows, compareLevels)
	if lowest.name != "3.0" || lowest.idc != picked("lowest-level") ||
		highest.idc <= highestLevel {
		t.Errorf("levels run %s (%d) to %s (%d)", lowest.name, lowest.idc,
			highest.name, highest.idc)
	}
	if !slices.IsSortedFunc(rows, compareLevels) {
		t.Errorf("the embedded level table is not in order")
	}
}

// compareLevels orders levels by their level_idc.
func compareLevels(left, right level) int { return left.idc - right.idc }

// the anti-test: a table that is malformed anywhere is refused, by row.
func TestAMalformedLevelTableIsRefused(t *testing.T) {
	heading := "level,macroblocks_per_second,macroblocks_per_frame,kbits\n"
	refusals := map[string]string{
		heading:                      "no rows",
		"":                           "no heading",
		heading + "4.1,1,2\n":        "columns",
		heading + "four,1,2,3\n":     "not a level",
		heading + "4,1,2,3\n":        "not a level",
		heading + "4.x,1,2,3\n":      "not a level",
		heading + "4.1,1,lots,3\n":   "not a count",
		heading + "4.1,1,2,3,4\n":    "columns",
		"\"unclosed\n":               "level table",
		heading + "4.1,1,2,3\n5,1\n": "row 2",
	}
	for text, says := range refusals {
		_, err := readLevels(text)
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("%q: %v, want %q", text, err, says)
		}
	}
}

// the anti-test: levels that do not rise are refused, a repeat as much as
// a fall, since levelFor takes the first level that fits.
func TestALevelTableOutOfOrderIsRefused(t *testing.T) {
	heading := "level,macroblocks_per_second,macroblocks_per_frame,kbits\n"
	for _, rows := range []string{"4.1,1,2,3\n4.0,1,2,3\n",
		"4.1,1,2,3\n4.1,1,2,3\n"} {
		_, err := readLevels(heading + rows)
		if err == nil || !strings.Contains(err.Error(), "in order") {
			t.Errorf("%q: %v, want a refusal saying the levels must be in "+
				"order", rows, err)
		}
	}
	if _, err := readLevels(heading + "4.0,1,2,3\n4.1,1,2,3\n"); err != nil {
		t.Errorf("levels that rise: %v", err)
	}
}
