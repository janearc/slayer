//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed testdata/geometry-records.yaml
var geometryCountsRaw []byte

// referenceRecord retains the full refusal text, including its spacing.
type referenceRecord struct {
	name, seedText, level, expected, reason string
	seed                                    uint64
}

func parseReferenceRecord(line string) (referenceRecord, error) {
	fields := strings.SplitN(line, " ", 4)
	if len(fields) != 4 || fields[0] == "" || (fields[1] != "5" && fields[1] != "11") {
		return referenceRecord{}, fmt.Errorf("invalid geometry reference row %q", line)
	}
	switch fields[2] {
	case "hull", "clumps", "quick", "demo":
	default:
		return referenceRecord{}, fmt.Errorf("invalid geometry detail in %q", line)
	}
	seed, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return referenceRecord{}, err
	}
	row := referenceRecord{name: fields[0], seedText: fields[1], seed: seed, level: fields[2], expected: fields[3]}
	if strings.HasPrefix(fields[3], "REFUSED ") {
		row.expected, row.reason = "REFUSED", strings.TrimPrefix(fields[3], "REFUSED ")
		if row.reason == "" {
			return referenceRecord{}, fmt.Errorf("empty refusal in %q", line)
		}
	} else if raw, err := hex.DecodeString(fields[3]); err != nil || len(raw) != 32 {
		return referenceRecord{}, fmt.Errorf("invalid geometry digest in %q", line)
	}
	return row, nil
}

// referenceFiles keeps independent historical sources separate.
var referenceFiles = []string{
	"geometry-reference-arm64.txt", "spacecraft-reference-arm64.txt",
	"ground-reference-arm64.txt", "survey-reference-arm64.txt", "large-reference-arm64.txt",
	"civic-reference-arm64.txt", "personal-reference-arm64.txt", "yard-reference-arm64.txt",
	"summit-reference-arm64.txt", "launcher-reference-arm64.txt", "vegetation-reference-arm64.txt",
	"stage-reference-arm64.txt", "vehicle-reference-arm64.txt", "saucer-reference-arm64.txt",
	"stack-reference-arm64.txt", "later-reference-arm64.txt", "historical-plants-reference-arm64.txt",
	"historical-public-partial.txt", "historical-bridge-demo-arm64.txt", "historical-corrected-scavenged-arm64.txt",
}

func TestAuthoredCatalogueMatchesItsGeometryRecords(t *testing.T) {
	var sources []io.Reader
	for _, name := range referenceFiles {
		file, err := os.Open("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		sources = append(sources, file)
	}
	scan := bufio.NewScanner(io.MultiReader(sources...))
	var rows []referenceRecord
	for scan.Scan() {
		row, err := parseReferenceRecord(scan.Text())
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	// Refuse an incomplete catalogue before attempting any expensive models.
	referencePopulation(t, rows)
	for _, row := range rows {
		t.Run(row.name+"_"+row.seedText+"_"+row.level, func(t *testing.T) {
			referenceSpecimen(t, row.name, row.seed, row.level, row.expected, row.reason)
		})
	}
}

// referencePopulation refuses missing, duplicate or uncatalogued cases.
func referencePopulation(t *testing.T, rows []referenceRecord) {
	t.Helper()
	var counts referenceCounts
	if err := yamlfile.Read("testdata/geometry-records.yaml", geometryCountsRaw, &counts); err != nil {
		t.Fatal(err)
	}
	actual := append(KindNames(), "spruce", "norway", "coast-redwood", "giant-sequoia",
		"coast-redwood-shoot", "giant-sequoia-shoot", "fern", "saguaro",
		"valley-oak", "alder", "poplar", "manzanita", "chaparral-brush", "joshua-tree",
		"kandelia-obovata", "aegiceras-corniculatum", "avicennia-marina", "bruguiera-gymnorhiza", "sonneratia-apetala",
		"pandanus-tectorius", "scaevola-taccada", "metrosideros-polymorpha", "barringtonia-asiatica", "quercus-ilex", "erica-arborea", "salix-alba")
	if err := validateReferencePopulation(rows, actual, counts); err != nil {
		t.Fatal(err)
	}
}

type referenceCounts struct{ Cases, Geometries, Refusals int }

func validateReferencePopulation(rows []referenceRecord, actual []string, counts referenceCounts) error {
	names := make(map[string]bool, len(actual))
	for _, name := range actual {
		if names[name] {
			return fmt.Errorf("duplicate catalogue kind %q", name)
		}
		names[name] = true
	}
	if len(names) == 0 {
		return fmt.Errorf("catalogue has no kinds")
	}
	seen := make(map[string]bool, len(rows))
	geometries, refusals := 0, 0
	for _, row := range rows {
		if !names[row.name] {
			return fmt.Errorf("uncatalogued geometry record %q", row.name)
		}
		key := row.name + " " + row.seedText + " " + row.level
		if seen[key] {
			return fmt.Errorf("duplicate geometry reference %s", key)
		}
		seen[key] = true
		if row.expected == "REFUSED" {
			refusals++
		} else {
			geometries++
		}
	}
	if len(rows) != counts.Cases || geometries != counts.Geometries || refusals != counts.Refusals {
		return fmt.Errorf("incomplete reference: %d cases, %d geometries, %d refusals", len(rows), geometries, refusals)
	}
	for name := range names {
		for _, seed := range []string{"5", "11"} {
			for _, level := range []string{"hull", "clumps", "quick", "demo"} {
				key := name + " " + seed + " " + level
				if !seen[key] {
					return fmt.Errorf("missing geometry reference %s", key)
				}
			}
		}
	}
	return nil
}

// referenceSpecimen checks one authored shape and its material record.
func referenceSpecimen(t *testing.T, name string, seed uint64,
	level, expected, reason string,
) {
	t.Helper()
	kind, err := testKind(name)
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(seed), level)
	if expected == "REFUSED" {
		var digest string
		if err == nil {
			digest = referenceDigest(t, made)
		}
		if !referenceRefusalAccepted(name, seed, level, err, digest, reason) {
			t.Fatalf("unsupported level: got error %v and digest %q, want refusal %q or the required recorded city success", err, digest, reason)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if got := referenceDigest(t, made); !referenceAccepted(name, seed, level, got, expected) {
		t.Fatalf("authored geometry changed: %s, recorded %s", got, expected)
	}
}

// referenceFingerprint holds the authored geometry and display material bytes.
func referenceFingerprint(t *testing.T, made good, expected string) {
	t.Helper()
	if got := referenceDigest(t, made); got != expected {
		t.Fatalf("authored geometry changed: %s, recorded %s", got, expected)
	}
}
