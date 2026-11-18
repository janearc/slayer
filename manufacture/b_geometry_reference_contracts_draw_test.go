//go:build draw && darwin && arm64

package manufacture

import (
	"bufio"
	_ "embed"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed testdata/geometry-records.yaml
var geometryCountsRaw []byte

// referenceReason is defined by the four fields before a refusal's reason.
const referenceReason = 4

func TestAuthoredCatalogueMatchesItsGeometryRecords(t *testing.T) {
	file, err := os.Open("testdata/geometry-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	additions, err := os.Open("testdata/spacecraft-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer additions.Close()
	ground, err := os.Open("testdata/ground-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer ground.Close()
	survey, err := os.Open("testdata/survey-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer survey.Close()
	large, err := os.Open("testdata/large-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer large.Close()
	civic, err := os.Open("testdata/civic-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer civic.Close()
	personal, err := os.Open("testdata/personal-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer personal.Close()
	yard, err := os.Open("testdata/yard-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer yard.Close()
	summit, err := os.Open("testdata/summit-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer summit.Close()
	launcher, err := os.Open("testdata/launcher-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer launcher.Close()
	vegetation, err := os.Open("testdata/vegetation-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer vegetation.Close()
	stage, err := os.Open("testdata/stage-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	vehicles, err := os.Open("testdata/vehicle-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer vehicles.Close()
	saucers, err := os.Open("testdata/saucer-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer saucers.Close()
	stacks, err := os.Open("testdata/stack-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer stacks.Close()
	later, err := os.Open("testdata/later-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer later.Close()
	names := map[string]bool{}
	seen := map[string]bool{}
	scan := bufio.NewScanner(io.MultiReader(file, additions, ground, survey, large, civic,
		personal, yard, summit, launcher, vegetation, stage, vehicles, saucers,
		stacks, later))
	cases, geometries, refusals := 0, 0, 0
	for scan.Scan() {
		row := strings.Fields(scan.Text())
		if len(row) < 4 {
			t.Fatalf("invalid geometry reference row: %q", scan.Text())
		}
		name, seedText, level := row[0], row[1], row[2]
		key := strings.Join(row[:3], " ")
		if seen[key] {
			t.Fatalf("duplicate geometry reference %s", key)
		}
		seen[key] = true
		names[name] = true
		cases++
		seed, err := strconv.ParseUint(seedText, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		expected := row[3]
		if expected == "REFUSED" {
			refusals++
		} else {
			geometries++
		}
		t.Run(name+"_"+seedText+"_"+level, func(t *testing.T) {
			referenceSpecimen(t, name, seed, level, expected,
				strings.Join(row[referenceReason:], " "))
		})
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	referencePopulation(t, names, cases, geometries, refusals)
}

// referencePopulation refuses incomplete records or uncatalogued kinds.
func referencePopulation(t *testing.T, names map[string]bool,
	cases, geometries, refusals int,
) {
	t.Helper()
	var expected struct{ Cases, Geometries, Refusals int }
	if err := yamlfile.Read("testdata/geometry-records.yaml", geometryCountsRaw, &expected); err != nil {
		t.Fatal(err)
	}
	if cases != expected.Cases || geometries != expected.Geometries || refusals != expected.Refusals {
		t.Fatalf("incomplete reference: %d cases, %d geometries, %d refusals", cases, geometries, refusals)
	}
	actual := KindNames()
	if len(actual) != len(names) {
		t.Fatal("catalogue and geometry reference have different kind counts")
	}
	for _, name := range actual {
		if !names[name] {
			t.Fatalf("kind %s has no reference", name)
		}
	}
}

// referenceSpecimen checks one authored shape and its material record.
func referenceSpecimen(t *testing.T, name string, seed uint64,
	level, expected, reason string,
) {
	t.Helper()
	kind, err := KindNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(seed), level)
	if expected == "REFUSED" {
		want := reason
		if err == nil || err.Error() != want {
			t.Fatalf("unsupported level: got %v, want %s", err, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	referenceFingerprint(t, made, expected)
}

// referenceFingerprint holds the authored geometry and display material bytes.
func referenceFingerprint(t *testing.T, made good, expected string) {
	t.Helper()
	if got := referenceDigest(t, made); got != expected {
		t.Fatalf("authored geometry changed: %s, recorded %s", got, expected)
	}
}
