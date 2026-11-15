package movie

import (
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
	first, last := rows[0], rows[len(rows)-1]
	if first.name != "3.0" || first.idc != 30 || last.idc <= highestLevel {
		t.Errorf("levels run %s (%d) to %s (%d)", first.name, first.idc,
			last.name, last.idc)
	}
}

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
