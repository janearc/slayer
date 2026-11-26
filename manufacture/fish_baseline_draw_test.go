//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The actual output of commit 324771b1, before fish profiles were external.
func TestFishKeepPreProfileOutput(t *testing.T) {
	file, err := os.Open("testdata/fish-pre-profile-324771b1-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows := fishHistoricalRows(t)
	counts := map[string]int{}
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		if len(fields) != 3 {
			t.Fatalf("invalid baseline row %q", scan.Text())
		}
		row, ok := rows[fields[0]]
		if !ok {
			t.Fatalf("unknown baseline fish %q", fields[0])
		}
		number, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		counts[fields[0]]++
		t.Run(fields[0]+"_"+fields[1], func(t *testing.T) {
			made, err := row.made(sow(number))
			if err != nil {
				t.Fatal(err)
			}
			if got := referenceDigest(t, made); got != fields[2] {
				t.Fatalf("got %s, want pre-profile %s", got, fields[2])
			}
		})
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if len(counts) != len(rows) {
		t.Fatalf("baseline covers %d of %d kinds", len(counts), len(rows))
	}
	for name := range rows {
		if counts[name] != 2 {
			t.Errorf("%s has %d baseline cases", name, counts[name])
		}
	}
}
