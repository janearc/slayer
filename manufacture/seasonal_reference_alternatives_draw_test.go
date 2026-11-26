//go:build draw && darwin && arm64

package manufacture

import (
	_ "embed"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

//go:embed testdata/chaparral-f11-alternates-arm64.txt
var chaparralAlternates string

//go:embed testdata/historical-stabilized-orca-alternatives.txt
var orcaAlternates string

//go:embed testdata/conifer-pre-profile-6ac6dc56-arm64.txt
var coniferAlternates string

// referenceAccepted keeps original records and accepts independently captured
// alternatives only for the explicitly named cases.
func referenceAccepted(name string, seed uint64, detail, got, original string) bool {
	if got == original {
		return true
	}
	var alternatives string
	switch name {
	case "chaparral-brush":
		alternatives = chaparralAlternates
	case "spruce", "norway", "coast-redwood", "giant-sequoia":
		alternatives = coniferAlternates
	case "gargoyle-orca-concrete", "gargoyle-orca-granite", "gargoyle-cornice-concrete", "gargoyle-cornice-granite":
		alternatives = orcaAlternates
	default:
		return false
	}
	for _, line := range strings.Split(alternatives, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 4 && fields[0] == name && fields[1] == strconv.FormatUint(seed, 10) && fields[2] == detail && fields[3] == got {
			return true
		}
	}
	return false
}

func TestChaparralReferenceAlternatesAreNarrow(t *testing.T) {
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(chaparralAlternates), "\n") {
		row := strings.Fields(line)
		if len(row) != 4 || row[0] != "chaparral-brush" || (row[1] != "5" && row[1] != "11") || (row[2] != "clumps" && row[2] != "quick" && row[2] != "demo") {
			t.Fatalf("unexpected alternate %q", line)
		}
		if raw, err := hex.DecodeString(row[3]); err != nil || len(raw) != 32 {
			t.Fatalf("invalid alternate digest %q", row[3])
		}
		key := row[1] + " " + row[2]
		if seen[key] {
			t.Fatalf("duplicate alternate %s", key)
		}
		seen[key] = true
		seed, _ := strconv.ParseUint(row[1], 10, 64)
		if !referenceAccepted(row[0], seed, row[2], row[3], "original") ||
			referenceAccepted(row[0], seed, row[2], strings.Repeat("0", 64), "original") ||
			referenceAccepted("other", seed, row[2], row[3], "original") {
			t.Fatalf("alternate did not constrain %s", key)
		}
	}
	if len(seen) != 6 {
		t.Fatalf("got %d chaparral alternatives, want 6", len(seen))
	}
}
