package main

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

const gaiaFixture = "source_id,ra,dec,phot_g_mean_mag\n1,0,0,0\n2,90,0,5\n3,0,90,0\n"

func TestGaiaPresentationPreservesDirectionsAndMagnitudeRatios(t *testing.T) {
	stars, err := readGaia(strings.NewReader(gaiaFixture), .1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if stars.Count() != 3 || stars.Light(maths.XAxis).R != 10 ||
		stars.Light(maths.YAxis).R != 10 || stars.Light(maths.ZAxis.Neg()).R != .1 {
		t.Fatal("positions or Pogson magnitude ratio changed")
	}
	if stars.Light(maths.XAxis.Neg()).R != 0 {
		t.Fatal("invented stars outside the source positions")
	}
}

func TestGaiaPresentationRejectsUnboundedInputs(t *testing.T) {
	for _, pair := range [][2]float64{{0, 1}, {2, 1}, {math.NaN(), 1}, {.1, 0}, {.1, math.Inf(1)}, {.1, math.NaN()}} {
		if _, err := readGaia(strings.NewReader(gaiaFixture), pair[0], pair[1]); err == nil {
			t.Fatal("invalid display settings accepted", pair)
		}
	}
	var tooMany strings.Builder
	tooMany.WriteString("source_id,ra,dec,phot_g_mean_mag\n")
	for id := 1; id <= 12001; id++ {
		fmt.Fprintf(&tooMany, "%d,0,0,1\n", id)
	}
	population := tooMany.String()
	if _, err := readGaia(strings.NewReader(population), .1, 1); err == nil {
		t.Fatal("unbounded star population accepted")
	}
	root := t.TempDir()
	path := filepath.Join(root, "oversize")
	if err := os.WriteFile(path, make([]byte, (4<<20)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (gaiaPlan{File: path}).load(); err == nil || !strings.Contains(err.Error(), "four MiB") {
		t.Fatal("oversize source did not fail before parsing", err)
	}
	if _, err := (gaiaPlan{File: root}).load(); err == nil {
		t.Fatal("directory accepted as Gaia data")
	}
	t.Setenv("SLAYER_SHELF", "")
	if _, err := (gaiaPlan{File: "relative.csv"}).load(); err == nil {
		t.Fatal("relative path accepted without source shelf")
	}
	t.Setenv("SLAYER_SHELF", root)
	if err := os.WriteFile(filepath.Join(root, "stars.csv"), []byte(gaiaFixture), 0600); err != nil {
		t.Fatal(err)
	}
	fixtureSum := sha256.Sum256([]byte(gaiaFixture))
	plan := gaiaPlan{File: "stars.csv", SHA256: fmt.Sprintf("%x", fixtureSum), Radius: .1, Zero: 1}
	if _, err := plan.load(); err != nil {
		t.Fatal("pinned source on shelf refused", err)
	}
}

// anti-test of TestGaiaPresentationPreservesDirectionsAndMagnitudeRatios
func TestGaiaPresentationRefusesInvalidRows(t *testing.T) {
	for _, row := range []string{
		"1,NaN,0,0", "1,0,Inf,0", "1,0,91,0", "1,360,0,0",
		"1,-1,0,0", "1,0,0,NaN", "1,0,0,999", "x,0,0,1",
		"1,0,0", "1,0,0,1\n1,20,30,1", "1,no,0,1",
	} {
		_, err := readGaia(strings.NewReader("source_id,ra,dec,phot_g_mean_mag\n"+row+"\n"), .1, 10)
		if err == nil {
			t.Fatalf("accepted invalid Gaia row %q", row)
		}
	}
	for _, raw := range []string{"", "ra,dec,mag\n", "source_id,ra,dec,phot_g_mean_mag\n"} {
		if _, err := readGaia(strings.NewReader(raw), .1, 10); err == nil {
			t.Fatalf("accepted incomplete catalogue %q", raw)
		}
	}
}

func TestGaiaPresentationRequiresThePinnedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gaia.csv")
	if err := os.WriteFile(path, []byte(gaiaFixture), 0600); err != nil {
		t.Fatal(err)
	}
	fixtureSum := sha256.Sum256([]byte(gaiaFixture))
	settings := gaiaPlan{File: path, SHA256: fmt.Sprintf("%x", fixtureSum),
		Radius: .1, Zero: 10}
	if _, err := settings.load(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(gaiaFixture+"4,1,1,1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.load(); err == nil {
		t.Fatal("changed catalogue silently changed the presentation")
	}
	settings.File = filepath.Join(t.TempDir(), "absent")
	if _, err := settings.load(); err == nil {
		t.Fatal("missing Gaia data silently replaced with invented stars")
	}
}
