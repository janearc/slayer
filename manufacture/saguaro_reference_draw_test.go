//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestSaguaroGeometryReference checks the existing author-recorded saguaro
// witnesses independently of the population check for the entire catalogue.
func TestSaguaroGeometryReference(t *testing.T) {
	file, err := os.Open("testdata/vegetation-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	count := 0
	for scan.Scan() {
		row := strings.Fields(scan.Text())
		if len(row) == 0 || row[0] != "saguaro" {
			continue
		}
		if len(row) != 4 {
			t.Fatalf("invalid saguaro reference: %q", scan.Text())
		}
		number, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		count++
		t.Run(row[1]+"_"+row[2], func(t *testing.T) { referenceSpecimen(t, row[0], number, row[2], row[3], "") })
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 8 {
		t.Fatalf("got %d saguaro references, want 8", count)
	}
}
