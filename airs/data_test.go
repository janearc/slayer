package airs

import (
	"slices"
	"strings"
	"testing"
)

// refused is the error a changed copy of the data gives, the test stopped
// where it gives none.
func refused(t *testing.T, changed data) string {
	t.Helper()
	err := changed.check()
	if err == nil {
		t.Fatal("the changed data was taken")
	}
	return err.Error()
}

// copied is the data as shipped, with its layers and absorbers copied,
// so that a test may change them.
func copied(t *testing.T) data {
	t.Helper()
	if venusError != nil {
		t.Fatal(venusError)
	}
	changed := venusRead
	changed.Layers = slices.Clone(venusRead.Layers)
	changed.Absorbers = slices.Clone(venusRead.Absorbers)
	changed.Ground.Points = slices.Clone(venusRead.Ground.Points)
	return changed
}

// The file as shipped is read and checked, and every number in it has a
// note: check refuses one that has not, so a file that is read has.
func TestShippedFileIsReadAndChecked(t *testing.T) {
	if venusError != nil {
		t.Fatal(venusError)
	}
	if err := venusRead.check(); err != nil {
		t.Error(err)
	}
	if len(venusRead.Layers) == 0 || len(venusRead.numbers()) < 20 { // picked: far fewer than the file holds
		t.Errorf("the file was read as %d layers", len(venusRead.Layers))
	}
}

// A number with no note, read or picked, is refused, and one with both
// is refused: the note is how the number was got.
func TestANumberWithoutASourceNoteIsRefused(t *testing.T) {
	changed := copied(t)
	changed.Planet.Gravity.Picked = ""
	if why := refused(t, changed); !strings.Contains(why, "gravity") ||
		!strings.Contains(why, "neither") {
		t.Errorf("no note: %s", why)
	}
	changed = copied(t)
	changed.Planet.Radius.Picked = "both"
	if why := refused(t, changed); !strings.Contains(why, "radius") ||
		!strings.Contains(why, "both") {
		t.Errorf("both notes: %s", why)
	}
	changed = copied(t)
	changed.Layers[1].Modes = slices.Clone(changed.Layers[1].Modes)
	changed.Layers[1].Modes[0].Width.Picked = ""
	if why := refused(t, changed); !strings.Contains(why, "width") {
		t.Errorf("a width with no note: %s", why)
	}
	changed = copied(t)
	changed.Layers[0].Read = ""
	if why := refused(t, changed); !strings.Contains(why, "lower haze") {
		t.Errorf("a layer with no note: %s", why)
	}
}

// A file whose text has a missing note is refused as it is read, too.
func TestAFileWithNoNoteIsRefusedAsItIsRead(t *testing.T) {
	const note = "    picked: >-\n      the value commonly given for " +
		"Venus's surface; no source for it\n      is in the tree or in " +
		"the facts file. To be pinned.\n"
	text := strings.Replace(string(venusRaw), note, "", 1)
	if text == string(venusRaw) {
		t.Fatal("the file has no such text to change")
	}
	if _, err := readData([]byte(text)); err == nil ||
		!strings.Contains(err.Error(), "gravity") {
		t.Errorf("gravity with no note was taken: %v", err)
	}
	peeked := strings.Replace(string(venusRaw), "picked: >-\n      the value",
		"peeked: >-\n      the value", 1)
	if _, err := readData([]byte(peeked)); err == nil {
		t.Error("a key the data has no field for was taken")
	}
	if _, err := readData(venusRaw); err != nil {
		t.Errorf("the file as shipped: %v", err)
	}
}

// A number is held to the unit it is read in, to a size above nought
// where it must, and to its range where it has one.
func TestNumbersAreHeldToTheirUnitsSizesAndRanges(t *testing.T) {
	changed := copied(t)
	changed.Planet.Radius.Unit = "km"
	if why := refused(t, changed); !strings.Contains(why, "km") {
		t.Errorf("a wrong unit: %s", why)
	}
	changed = copied(t)
	changed.Gas.ScaleHeight.Value = 0
	if why := refused(t, changed); !strings.Contains(why, "scale height") {
		t.Errorf("a scale height of nought: %s", why)
	}
	changed = copied(t)
	changed.Gas.Absorption.Value = -1
	if why := refused(t, changed); !strings.Contains(why, "absorption") {
		t.Errorf("a negative absorption: %s", why)
	}
	changed = copied(t)
	changed.Layers[1].Depth.Value = 13 // picked: over Table 1's 12 for the lower cloud
	if why := refused(t, changed); !strings.Contains(why, "lower cloud") ||
		!strings.Contains(why, "range") {
		t.Errorf("a depth out of its range: %s", why)
	}
	changed = copied(t)
	changed.Gas.Absorption.Value = 0
	if err := changed.check(); err != nil {
		t.Errorf("an absorption of nought is allowed: %v", err)
	}
}

// The layers run on from one to the next, from above the ground to the
// top of the air, each with a range of depth and modes of its own.
func TestLayersAreContiguousAndWholeOrRefused(t *testing.T) {
	changed := copied(t)
	changed.Layers[2].Bottom += 100 // picked: a gap of a hundred metres
	if why := refused(t, changed); !strings.Contains(why, "middle cloud") ||
		!strings.Contains(why, "gap") {
		t.Errorf("a gap: %s", why)
	}
	changed = copied(t)
	changed.Layers[0].Bottom = 0
	if why := refused(t, changed); !strings.Contains(why, "above the ground") {
		t.Errorf("a layer on the ground: %s", why)
	}
	changed = copied(t)
	changed.Layers[len(changed.Layers)-1].Top -= 1000 // picked: a kilometre short
	if why := refused(t, changed); !strings.Contains(why, "top of the air") {
		t.Errorf("a short top: %s", why)
	}
	changed = copied(t)
	changed.Layers[3].Top = changed.Layers[3].Bottom
	if why := refused(t, changed); !strings.Contains(why, "upper cloud") {
		t.Errorf("a layer with no thickness: %s", why)
	}
	changed = copied(t)
	changed.Layers[1].Depth.Low, changed.Layers[1].Depth.High = 0, 0
	if why := refused(t, changed); !strings.Contains(why, "range") {
		t.Errorf("a depth with no range: %s", why)
	}
	changed = copied(t)
	changed.Layers[1].Modes = nil
	if why := refused(t, changed); !strings.Contains(why, "modes") {
		t.Errorf("a layer with no modes: %s", why)
	}
}

// A mode is a size, a number and a width above one, with a note.
func TestModesAreHeldToTheirSizes(t *testing.T) {
	for name, change := range map[string]func(*modeData){
		"diameter": func(mode *modeData) { mode.Diameter = 0 },
		"number":   func(mode *modeData) { mode.Number = 0 },
		"width":    func(mode *modeData) { mode.Width.Value = 1 },
		"note":     func(mode *modeData) { mode.Read = "" },
	} {
		changed := copied(t)
		changed.Layers[1].Modes = slices.Clone(changed.Layers[1].Modes)
		change(&changed.Layers[1].Modes[1])
		if why := refused(t, changed); !strings.Contains(why, "lower cloud") ||
			!strings.Contains(why, "mode 2") {
			t.Errorf("%s: %s", name, why)
		}
	}
}

// The gas's shares make a whole, its index has its limits and its
// conditions, and nothing is read from nowhere.
func TestGasIsHeldToItsShares(t *testing.T) {
	changed := copied(t)
	changed.Gas.Nitrogen.Value = 0.05 // picked: shares of 1.015
	if why := refused(t, changed); !strings.Contains(why, "share") {
		t.Errorf("shares over one: %s", why)
	}
	changed = copied(t)
	changed.Gas.Refraction.Terms = nil
	if why := refused(t, changed); !strings.Contains(why, "index") {
		t.Errorf("an index with no terms: %s", why)
	}
	changed = copied(t)
	changed.Gas.Refraction.Longest = changed.Gas.Refraction.Shortest
	if why := refused(t, changed); !strings.Contains(why, "index") {
		t.Errorf("an index with no range: %s", why)
	}
	changed = copied(t)
	changed.Gas.Refraction.Pressure = 0
	if why := refused(t, changed); !strings.Contains(why, "index") {
		t.Errorf("an index at no pressure: %s", why)
	}
	changed = copied(t)
	changed.Gas.Refraction.Read = ""
	if why := refused(t, changed); !strings.Contains(why, "index") {
		t.Errorf("an index with no note: %s", why)
	}
}

// The absorbers are in the air, below a wavelength, and noted.
func TestAbsorbersAreHeldToTheAir(t *testing.T) {
	for name, change := range map[string]func(*absorberData){
		"inverted": func(each *absorberData) { each.Bottom = each.Top },
		"above":    func(each *absorberData) { each.Top = 1e6 }, // picked: far above the air
		"limit":    func(each *absorberData) { each.Below = 0 },
		"note":     func(each *absorberData) { each.Read = "" },
	} {
		changed := copied(t)
		changed.Absorbers[0] = venusRead.Absorbers[0]
		change(&changed.Absorbers[0])
		if why := refused(t, changed); !strings.Contains(why, "ultraviolet") {
			t.Errorf("%s: %s", name, why)
		}
	}
}

// The ground's albedo is a table: noted, ascending, between nought and
// one, of two points at least.
func TestGroundTableIsHeldToItsShape(t *testing.T) {
	for name, change := range map[string]func(*groundData){
		"note":      func(each *groundData) { each.Read = "" },
		"one point": func(each *groundData) { each.Points = each.Points[:1] },
		"order":     func(each *groundData) { each.Points[1][0] = each.Points[0][0] },
		"albedo":    func(each *groundData) { each.Points[2][1] = 1.5 },
		"negative":  func(each *groundData) { each.Points[2][1] = -0.1 },
	} {
		changed := copied(t)
		change(&changed.Ground)
		if why := refused(t, changed); !strings.Contains(why, "ground") {
			t.Errorf("%s: %s", name, why)
		}
	}
}
