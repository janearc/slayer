//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestConifersKeepPreProfileOutput checks the actual 6ac6dc56 output. This
// witness does not replace the independently maintained catalogue records.
func TestConifersKeepPreProfileOutput(t *testing.T) {
	file, err := os.Open("testdata/conifer-pre-profile-6ac6dc56-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	names := map[string]int{"spruce": 0, "norway": 0, "coast-redwood": 0, "giant-sequoia": 0}
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if len(fields) != 4 {
			t.Fatalf("invalid baseline: %q", scan.Text())
		}
		if _, found := names[fields[0]]; !found {
			t.Fatalf("unknown baseline kind %q", fields[0])
		}
		number, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		names[fields[0]]++
		t.Run(fields[0]+"_"+fields[1]+"_"+fields[2], func(t *testing.T) {
			referenceSpecimen(t, fields[0], number, fields[2], fields[3], "")
		})
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	for name, n := range names {
		if n != 8 {
			t.Errorf("%s has %d baseline cases, want 8", name, n)
		}
	}
}
