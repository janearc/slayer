package airs

import (
	"strings"
	"testing"
)

// refusedEarth is the error a changed copy of Earth's data gives, the
// test stopped where it gives none.
func refusedEarth(t *testing.T, changed earthData) string {
	t.Helper()
	err := changed.check()
	if err == nil {
		t.Fatal("the changed data was taken")
	}
	return err.Error()
}

// The file as shipped is read and checked, and every number in it has a
// note: check refuses one that has not, so a file that is read has.
func TestEarthShippedFileIsReadAndChecked(t *testing.T) {
	if earthError != nil {
		t.Fatal(earthError)
	}
	if err := earthRead.check(); err != nil {
		t.Error(err)
	}
	if len(earthRead.numbers()) < 12 || len(earthRead.Ozone.Layers) == 0 { // picked: far fewer than the file holds
		counted := len(earthRead.numbers())
		t.Errorf("the file was read as %d numbers", counted)
	}
}

// A number with no note is refused, one with both is refused, and so is
// a key the data has no field for.
func TestEarthNumberWithoutASourceNoteIsRefused(t *testing.T) {
	changed := copiedEarth(t)
	changed.Planet.Gravity.Read = ""
	if why := refusedEarth(t, changed); !strings.Contains(why, "gravity") ||
		!strings.Contains(why, "neither") {
		t.Errorf("no note: %s", why)
	}
	changed = copiedEarth(t)
	changed.Aerosol.Depth.Read = "both"
	if why := refusedEarth(t, changed); !strings.Contains(why, "depth") ||
		!strings.Contains(why, "both") {
		t.Errorf("both notes: %s", why)
	}
	peeked := strings.Replace(string(earthRaw), "dry-air-mass:", "dry-air-weight:", 1)
	if _, err := readEarth([]byte(peeked)); err == nil {
		t.Error("a key the data has no field for was taken")
	}
	if _, err := readEarth(earthRaw); err != nil {
		t.Errorf("the file as shipped: %v", err)
	}
}

// A number is held to its unit and to a size above nought where it must
// be, and the Angstrom exponent may be nought.
func TestEarthNumbersAreHeldToTheirUnitsAndSizes(t *testing.T) {
	changed := copiedEarth(t)
	changed.Planet.Radius.Unit = "km"
	if why := refusedEarth(t, changed); !strings.Contains(why, "km") {
		t.Errorf("a wrong unit: %s", why)
	}
	changed = copiedEarth(t)
	changed.Aerosol.ScaleHeight.Value = 0
	if why := refusedEarth(t, changed); !strings.Contains(why, "scale height") {
		t.Errorf("a scale height of nought: %s", why)
	}
	changed = copiedEarth(t)
	changed.Aerosol.Angstrom.Value = 0
	changed.Aerosol.Angstrom.Low, changed.Aerosol.Angstrom.High = 0, 0
	if err := changed.check(); err != nil {
		t.Errorf("an Angstrom exponent of nought is allowed: %v", err)
	}
	changed = copiedEarth(t)
	changed.Aerosol.Angstrom.Value = 2 // picked: over the range read, 1.02 to 1.71
	if why := refusedEarth(t, changed); !strings.Contains(why, "range") {
		t.Errorf("an exponent out of its range: %s", why)
	}
}

// The shells have a note, run up from the ground each with a thickness,
// and end at the top of the air.
func TestEarthShellsAreHeldToTheirShape(t *testing.T) {
	for name, change := range map[string]func(*earthData){
		"note":      func(each *earthData) { each.Shells.Picked = "" },
		"no runs":   func(each *earthData) { each.Shells.Runs = nil },
		"thickness": func(each *earthData) { each.Shells.Runs[1].Thickness = 0 },
		"order":     func(each *earthData) { each.Shells.Runs[1].To = each.Shells.Runs[0].To },
		"short":     func(each *earthData) { each.Shells.Runs = each.Shells.Runs[:2] },
	} {
		changed := copiedEarth(t)
		change(&changed)
		if why := refusedEarth(t, changed); !strings.Contains(why, "shell") &&
			!strings.Contains(why, "top of the air") {
			t.Errorf("%s: %s", name, why)
		}
	}
	if err := copiedEarth(t).check(); err != nil {
		t.Errorf("the shells as shipped: %v", err)
	}
}

// The ozone's layers run on from the pressure at the ground down, each
// noted, and make the total.
func TestEarthOzoneLayersAreContiguousAndMakeTheTotal(t *testing.T) {
	for name, change := range map[string]func(*earthData){
		"gap":       func(each *earthData) { each.Ozone.Layers[2].Bottom -= 1 },   // picked: a hectopascal
		"ground":    func(each *earthData) { each.Ozone.Layers[0].Bottom = 1000 }, // picked: not the ground's 1013.25
		"note":      func(each *earthData) { each.Ozone.Layers[3].Read = "" },
		"thickness": func(each *earthData) { each.Ozone.Layers[3].Top = each.Ozone.Layers[3].Bottom },
		"negative":  func(each *earthData) { each.Ozone.Layers[3].Amount = -1 }, // picked: under nought
		"total":     func(each *earthData) { each.Ozone.Total.Value = 310 },     // picked: ten over
	} {
		changed := copiedEarth(t)
		change(&changed)
		why := refusedEarth(t, changed)
		if !strings.Contains(why, "Umkehr") && !strings.Contains(why, "ozone") {
			t.Errorf("%s: %s", name, why)
		}
	}
}

// The ozone's cross sections are a table of figures above nought, from
// two points, that rise in wavelength.
func TestEarthOzoneCrossSectionsAreHeldToAShape(t *testing.T) {
	for name, change := range map[string]func(*tableData){
		"note":     func(each *tableData) { each.Read = "" },
		"one":      func(each *tableData) { each.Points = each.Points[:1] },
		"order":    func(each *tableData) { each.Points[5][0] = each.Points[4][0] },
		"nought":   func(each *tableData) { each.Points[5][1] = 0 },
		"negative": func(each *tableData) { each.Points[5][1] = -1e-21 }, // picked: under nought
	} {
		changed := copiedEarth(t)
		change(&changed.Ozone.CrossSection)
		if why := refusedEarth(t, changed); !strings.Contains(why, "ozone cross section") {
			t.Errorf("%s: %s", name, why)
		}
	}
}

// A table has a note, read or picked, and not both: the ground's picked
// albedo is taken, and the same with a read note too is refused.
func TestATableHasOneNoteReadOrPicked(t *testing.T) {
	if earthRead.Ground.Picked == "" || earthRead.Ground.Read != "" {
		t.Fatal("Earth's ground is not a picked table")
	}
	changed := copiedEarth(t)
	changed.Ground.Read = "both"
	if why := refusedEarth(t, changed); !strings.Contains(why, "ground") {
		t.Errorf("a ground with both notes: %s", why)
	}
	changed = copiedEarth(t)
	changed.Ground.Picked = ""
	if why := refusedEarth(t, changed); !strings.Contains(why, "ground") {
		t.Errorf("a ground with no note: %s", why)
	}
}

// A table's figure is linear between its points and flat past its ends:
// the aerosol's albedo at 510 nm is between its points at 470 and 550.
func TestATableIsLinearBetweenPointsAndFlatBeyond(t *testing.T) {
	albedo := earthRead.Aerosol.Albedo
	for name, each := range map[string][2]float64{
		"between": {510, 0.895}, // defined: halfway from 0.90 at 470 nm to 0.89 at 550
		"a point": {550, 0.89},
		"below":   {300, 0.90},
		"above":   {1000, 0.88},
	} {
		if got := albedo.at(each[0]); got-each[1] > 1e-12 || each[1]-got > 1e-12 { // picked: rounding only
			t.Errorf("%s: albedo %g at %g nm, want %g", name, got, each[0], each[1])
		}
	}
	if albedo.at(510) == albedo.at(470) || albedo.at(510) == albedo.at(550) {
		t.Error("the figure between two points is a point's own")
	}
}
