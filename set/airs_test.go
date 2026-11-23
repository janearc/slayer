package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// airsWeather is the laser fixture's weather line, before which a block of
// the scene is written.
const airsWeather = "  weather:\n"

// mistLines are a mist's particles, gas, lid and patches, written after
// the fixture's grove mist's seed.
const mistLines = "      angstrom: 1.4\n" +
	"      nitrogen-dioxide: {value: 300, unit: ppb}\n" +
	"      edge: {value: 5, unit: m}\n" +
	"      size: {value: 50, unit: m}\n"

// cloudLines are a cloud deck written after the fixture's grove mist.
const cloudLines = "    high cloud:\n" +
	"      kind: prescribed-cloud\n" +
	"      extinction: {value: 0.05, unit: m^-1}\n" +
	"      base: {value: 1500, unit: m}\n" +
	"      top: {value: 3000, unit: m}\n" +
	"      coverage: 0.4\n" +
	"      seed: 3\n" +
	"      size: {value: 800, unit: m}\n" +
	"      asymmetry: 0.85\n"

// layerLines are a layer over the terrain that reaches past it and is
// thickest over the observatory, written before the fixture's weather.
const layerLines = "  atmosphere:\n    layers:\n      air:\n" +
	"        weather: grove mist\n        extent: terrain\n" +
	"        albedo: 0.9\n        beyond: {value: 500, unit: m}\n" +
	"        toward: {object: the observatory, reach: " +
	"{value: 300, unit: m}, least: 0.2}\n"

// plumeLines are a plume from the observatory, written before the
// fixture's weather.
const plumeLines = "  plumes:\n    stack:\n" +
	"      from: {object: the observatory, east: {value: 1, unit: m}, " +
	"up: {value: 12, unit: m}, south: {value: 0, unit: m}}\n" +
	"      mouth: {value: 0.4, unit: m}\n" +
	"      exit: {value: 8, unit: m/s}\n" +
	"      temperature: {value: 600, unit: K}\n" +
	"      air-temperature: {value: 293, unit: K}\n" +
	"      rate: {value: 2, unit: g/s}\n" +
	"      mass-extinction: {value: 10, unit: m^2/g}\n" +
	"      albedo: 0.2\n" +
	"      asymmetry: 0.6\n" +
	"      angstrom: 1\n" +
	"      reach: {value: 600, unit: m}\n" +
	"      lid: {value: 300, unit: m}\n" +
	"      source: a test\n"

// airsSources is each laser fixture, with and without its schema, with
// lines written after its mist's seed and before its weather.
func airsSources(test *testing.T, afterSeed, beforeWeather string) []string {
	test.Helper()
	var sources []string
	for _, raw := range laserSources() {
		if !strings.Contains(raw, mistSeed) ||
			!strings.Contains(raw, airsWeather) {
			test.Fatal("the laser fixture's weather has moved")
		}
		raw = strings.Replace(raw, mistSeed, mistSeed+afterSeed, 1)
		raw = strings.Replace(raw, airsWeather, beforeWeather+airsWeather, 1)
		sources = append(sources, raw)
	}
	return sources
}

// TestAMistSaysItsParticlesItsGasAndItsLid: a mist's Angstrom exponent,
// nitrogen dioxide, edge and patch size are read on both reader paths.
func TestAMistSaysItsParticlesItsGasAndItsLid(test *testing.T) {
	for _, source := range airsSources(test, mistLines, "") {
		made, err := set.Read("airs.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		mist, _ := made.Mist("grove mist")
		exponent, said := mist.Angstrom()
		if !said || exponent != 1.4 || mist.NitrogenDioxide() != 300 ||
			mist.Edge() != 5 || mist.Size() != 50 {
			test.Fatalf("lost the mist's optics: %+v", mist)
		}
		if _, _, cloud := mist.Cloud(); cloud {
			test.Fatal("a mist was read as a cloud deck")
		}
	}
}

// TestAMistThatSaysNoParticlesIsGrey: unsaid, a mist has no exponent and
// no gas, as before the keys were there.
func TestAMistThatSaysNoParticlesIsGrey(test *testing.T) {
	for _, source := range airsSources(test, "", "") {
		made, err := set.Read("airs.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		mist, _ := made.Mist("grove mist")
		if _, said := mist.Angstrom(); said || mist.NitrogenDioxide() != 0 ||
			mist.Edge() != 0 {
			test.Fatalf("a plain mist says optics: %+v", mist)
		}
	}
}

// TestAMistsOpticsAreRefusedOutOfBounds: an exponent past two and a half,
// a gas in another unit, an edge deeper than the mist and a cloud's base
// on a mist are each refused.
func TestAMistsOpticsAreRefusedOutOfBounds(test *testing.T) {
	for _, lines := range []string{"      angstrom: 2.6\n",
		"      nitrogen-dioxide: {value: 3, unit: ppm}\n",
		"      edge: {value: 25, unit: m}\n",
		"      base: {value: 10, unit: m}\n",
		"      angstrom: -0.1\n"} {
		for _, source := range airsSources(test, lines, "") {
			if _, err := set.Read("airs.yaml", []byte(source)); err == nil {
				test.Errorf("%q was taken", lines)
			}
		}
	}
}

// TestACloudDeckSaysItsBaseAndTop: a cloud deck is read with its base,
// top, size and lobe, and its depth is the deck's.
func TestACloudDeckSaysItsBaseAndTop(test *testing.T) {
	for _, source := range airsSources(test, cloudLines, "") {
		made, err := set.Read("airs.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		cloud, found := made.Mist("high cloud")
		base, top, deck := cloud.Cloud()
		if !found || !deck || base != 1500 || top != 3000 ||
			cloud.Depth() != 1500 || cloud.Size() != 800 ||
			cloud.Asymmetry() != 0.85 {
			test.Fatalf("lost the cloud deck: %+v", cloud)
		}
	}
}

// TestACloudDeckIsRefusedWithoutItsShape: a top under the base, a depth or
// an edge on a deck, no cover and no top are each refused.
func TestACloudDeckIsRefusedWithoutItsShape(test *testing.T) {
	faults := [][2]string{
		{"top: {value: 3000, unit: m}", "top: {value: 1000, unit: m}"},
		{"size: {value: 800, unit: m}", "depth: {value: 800, unit: m}"},
		{"coverage: 0.4", "coverage: 0"},
		{"      top: {value: 3000, unit: m}\n", ""},
		{"seed: 3\n", "seed: 3\n      edge: {value: 9, unit: m}\n"},
	}
	for _, fault := range faults {
		before, after := fault[0], fault[1]
		broken := strings.Replace(cloudLines, before, after, 1)
		for _, source := range airsSources(test, broken, "") {
			if _, err := set.Read("airs.yaml", []byte(source)); err == nil {
				test.Errorf("%q for %q was taken", after, before)
			}
		}
	}
}

// TestALayerReachesBeyondAndThinsAwayFromAThing: a layer's reach past the
// terrain and the thing it is thickest over are read.
func TestALayerReachesBeyondAndThinsAwayFromAThing(test *testing.T) {
	for _, source := range airsSources(test, "", layerLines) {
		made, err := set.Read("airs.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		layer := made.AtmosphericLayers()[0]
		object, reach, least := layer.Toward()
		if layer.Beyond() != 500 || object != "the observatory" ||
			reach != 300 || least != 0.2 {
			test.Fatalf("lost the layer's reach: %+v", layer)
		}
	}
}

// TestALayerThinningTowardNothingIsRefused: an unknown thing, a least past
// one and a toward with no reach are each refused.
func TestALayerThinningTowardNothingIsRefused(test *testing.T) {
	faults := [][2]string{
		{"object: the observatory", "object: the moon"},
		{"least: 0.2", "least: 1.5"},
		{"reach: {value: 300, unit: m},", ""},
	}
	for _, fault := range faults {
		before, after := fault[0], fault[1]
		broken := strings.Replace(layerLines, before, after, 1)
		for _, source := range airsSources(test, "", broken) {
			if _, err := set.Read("airs.yaml", []byte(source)); err == nil {
				test.Errorf("%q for %q was taken", after, before)
			}
		}
	}
}

// TestAPlumeSaysWhatLeavesItsMouth: a plume's origin, stack, particles,
// reach and lid are read, and the slice is the caller's.
func TestAPlumeSaysWhatLeavesItsMouth(test *testing.T) {
	for _, source := range airsSources(test, "", plumeLines) {
		made, err := set.Read("airs.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		plume := made.Plumes()[0]
		object, east, up, _ := plume.Origin()
		mouth, exit, hot, air := plume.Stack()
		rate, extinction, albedo, asymmetry, exponent := plume.Particles()
		lid, said := plume.Lid()
		count, _, _ := plume.Mouths()
		if object != "the observatory" || east != 1 || up != 12 ||
			mouth != 0.4 || exit != 8 || hot != 600 || air != 293 ||
			rate != 2 || extinction != 10 || albedo != 0.2 ||
			asymmetry != 0.6 || exponent != 1 || plume.Reach() != 600 ||
			!said || lid != 300 || count != 1 {
			test.Fatalf("lost the plume: %+v", plume)
		}
		made.Plumes()[0] = set.Plume{}
		if made.Plumes()[0].Name() != "stack" {
			test.Fatal("the caller changed the set's plumes")
		}
	}
}

// TestAPlumeIsRefusedWithoutWhatCarriesIt: no wind, an unknown thing, many
// mouths with no spread, gas no hotter than the air and no source are each
// refused.
func TestAPlumeIsRefusedWithoutWhatCarriesIt(test *testing.T) {
	faults := [][2]string{
		{"object: the observatory", "object: the moon"},
		{"      albedo: 0.2\n", "      albedo: 0.2\n      count: 3\n"},
		{"temperature: {value: 600", "temperature: {value: 290"},
		{"      source: a test\n", ""},
		{"      mouth: {value: 0.4, unit: m}\n", ""},
	}
	for _, fault := range faults {
		before, after := fault[0], fault[1]
		broken := strings.Replace(plumeLines, before, after, 1)
		for _, source := range airsSources(test, "", broken) {
			if _, err := set.Read("airs.yaml", []byte(source)); err == nil {
				test.Errorf("%q for %q was taken", after, before)
			}
		}
	}
	for _, source := range airsSources(test, "", plumeLines) {
		still := strings.Replace(source, "speed: {value: 2, unit: m/s}",
			"speed: {value: 0, unit: m/s}", 1)
		if _, err := set.Read("airs.yaml", []byte(still)); err == nil {
			test.Error("a plume with no wind was taken")
		}
	}
}
