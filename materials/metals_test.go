package materials

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// publishedRow reads a row of a database file committed under testdata:
// the n, the k and the whole row as the file writes it, at the
// wavelength in micrometres the row gives. The database's tables are
// lines of three numbers; every other line of the file is refused as a
// row.
func publishedRow(t *testing.T, file, micrometres string) (wavelength, n, k float64, line string) {
	t.Helper()
	whole, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	at, err := strconv.ParseFloat(micrometres, 64)
	if err != nil {
		t.Fatalf("parsing %s: %v", micrometres, err)
	}
	for _, each := range strings.Split(string(whole), "\n") {
		fields := strings.Fields(each)
		if len(fields) != 3 {
			continue
		}
		rowW, errW := strconv.ParseFloat(fields[0], 64)
		rowN, errN := strconv.ParseFloat(fields[1], 64)
		rowK, errK := strconv.ParseFloat(fields[2], 64)
		if errW != nil || errN != nil || errK != nil {
			continue
		}
		if rowW == at {
			return rowW, rowN, rowK, strings.TrimSpace(each)
		}
	}
	t.Fatalf("no row at %s in %s", micrometres, file)
	return 0, 0, 0, ""
}

// TestTheMetalsMatchTheirPublishedRows reads every committed metal's n
// and k back out of the published database files under testdata and
// compares them, exactly, with what the package stores: the values must
// be the published ones, not near them.
func TestTheMetalsMatchTheirPublishedRows(t *testing.T) {
	metals := map[string]map[Band]string{
		"gold": {
			Red: "0.6168", Green: "0.5486", Blue: "0.4714",
		},
		"silver": {
			Red: "0.6168", Green: "0.5486", Blue: "0.4714",
		},
		"aluminium": {
			Red: "0.61359", Green: "0.54147", Blue: "0.45833",
		},
		"beryllium": {
			Red: "0.61045", Green: "0.54960", Blue: "0.46562",
		},
		"titanium": {
			Red: "0.617", Green: "0.549", Blue: "0.471",
		},
	}
	files := map[string]string{
		"gold":      "main/Au/nk/Johnson.yml",
		"silver":    "main/Ag/nk/Johnson.yml",
		"aluminium": "main/Al/nk/Rakic-BB.yml",
		"beryllium": "main/Be/nk/Rakic-LD.yml",
		"titanium":  "main/Ti/nk/Johnson.yml",
	}
	for name, rows := range metals {
		made, err := ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if made.Name() != name {
			t.Errorf("%s answers to the name %s", name, made.Name())
		}
		// aluminium's density is cited, from NIST's Table 1 (xray.go);
		// the others have none yet
		if name != "aluminium" && made.Density() != 0 {
			t.Errorf("%s has a density of %v with none cited", name, made.Density())
		}
		for band, at := range rows {
			rowW, rowN, rowK, line := publishedRow(t, files[name], at)
			read := made.Optics().At(band)
			if !made.Optics().Has(band) {
				t.Errorf("%s has no %s reading", name, band)
			}
			if read.N() != rowN {
				t.Errorf("%s %s: n %v, the published row has %v",
					name, band, read.N(), rowN)
			}
			if read.K() != rowK {
				t.Errorf("%s %s: k %v, the published row has %v",
					name, band, read.K(), rowK)
			}
			if math.Abs(read.Wavelength()-rowW*1000) > 1e-6 {
				t.Errorf("%s %s: wavelength %v nm, the published row has %v",
					name, band, read.Wavelength(), rowW*1000)
			}
			// where it was read is the row itself
			if read.Where().Row() != line {
				t.Errorf("%s %s: read from %q, not the row %q",
					name, band, read.Where().Row(), line)
			}
			if read.Where().Release() != release {
				t.Errorf("%s %s: read from %q, not %q",
					name, band, read.Where().Release(), release)
			}
			if !strings.Contains(read.Where().Material(), files[name]) {
				t.Errorf("%s %s: read from %q, which is not %s",
					name, band, read.Where().Material(), files[name])
			}
			if read.Where().Paper() == "" {
				t.Errorf("%s %s: read from no paper", name, band)
			}
		}
	}
}

func TestByNameRefusesAnUnknownName(t *testing.T) {
	// aluminium is no longer refused: its blue n was corrected to the
	// published row on 2026-11-11, by decision, so the package
	// carries it like the other metals. "yard steel" is a look's name,
	// not a material's: the material is steel, plainly.
	for _, name := range []string{"flan", "unobtainium", "yard steel"} {
		_, err := ByName(name)
		if err == nil {
			t.Errorf("a material named %q was made", name)
			continue
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the refusal of %q does not name it: %v", name, err)
		}
	}
}
