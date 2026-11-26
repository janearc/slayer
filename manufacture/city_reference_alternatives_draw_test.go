//go:build draw && darwin && arm64

package manufacture

import (
	_ "embed"
	"os"
	"strconv"
	"strings"
	"testing"
)

//go:embed testdata/historical-city-demo-arm64.txt
var cityDemoAlternatives string

// referenceRefusalAccepted requires the recorded successful city digest and
// preserves the exact historical error for every other refused case.
func referenceRefusalAccepted(name string, seed uint64, level string, err error, digest, reason string) bool {
	supported := false
	switch name {
	case "city-apartment", "city-clustered-10000", "city-clustered-green-10000", "city-clustered-night-10000", "city-grid-10000", "city-rowhouse", "city-tower":
		supported = (seed == 5 || seed == 11) && level == "demo"
	}
	if !supported {
		return err != nil && err.Error() == reason
	}
	if err != nil {
		return false
	}
	for _, line := range strings.Split(cityDemoAlternatives, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 4 && fields[0] == name && fields[1] == strconv.FormatUint(seed, 10) && fields[2] == level && fields[3] == digest {
			return true
		}
	}
	return false
}

func TestCityDemoReferenceAlternatesAreNarrow(t *testing.T) {
	names := map[string]bool{"city-apartment": true, "city-clustered-10000": true, "city-clustered-green-10000": true, "city-clustered-night-10000": true, "city-grid-10000": true, "city-rowhouse": true, "city-tower": true}
	original, err := os.ReadFile("testdata/later-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	originals := map[string]referenceRecord{}
	for _, line := range strings.Split(strings.TrimSpace(string(original)), "\n") {
		row, err := parseReferenceRecord(line)
		if err != nil {
			t.Fatal(err)
		}
		originals[row.name+" "+row.seedText+" "+row.level] = row
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(cityDemoAlternatives), "\n") {
		row, err := parseReferenceRecord(line)
		if err != nil {
			t.Fatal(err)
		}
		key := row.name + " " + row.seedText + " " + row.level
		old := originals[key]
		if !names[row.name] || row.level != "demo" || row.expected == "REFUSED" || seen[key] || old.expected != "REFUSED" || old.reason == "" {
			t.Fatalf("unexpected city alternative %s", key)
		}
		seen[key] = true
		if !referenceRefusalAccepted(row.name, row.seed, row.level, nil, row.expected, old.reason) ||
			referenceRefusalAccepted(row.name, row.seed, row.level, nil, strings.Repeat("0", 64), old.reason) ||
			referenceRefusalAccepted("unlisted", row.seed, row.level, nil, row.expected, old.reason) ||
			referenceRefusalAccepted(row.name, 99, row.level, nil, row.expected, old.reason) ||
			referenceRefusalAccepted(row.name, row.seed, "quick", nil, row.expected, old.reason) ||
			referenceRefusalAccepted(row.name, row.seed, row.level, testRefusal(old.reason), "", old.reason) ||
			!referenceRefusalAccepted("unlisted", row.seed, row.level, testRefusal(old.reason), "", old.reason) ||
			referenceRefusalAccepted("unlisted", row.seed, row.level, testRefusal(old.reason+" changed"), "", old.reason) ||
			referenceRefusalAccepted(row.name, row.seed, row.level, testRefusal(old.reason+" changed"), "", old.reason) {
			t.Fatalf("city alternative weakened %s", key)
		}
	}
	if len(seen) != 14 {
		t.Fatalf("got %d city alternatives, want 14", len(seen))
	}
}

type testRefusal string

func (e testRefusal) Error() string { return string(e) }
