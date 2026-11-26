package set

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/puzzlemouth/yamlfile"
)

// covLegacy reads the laser fixture with its schema line removed, so only
// the domain reader judges it, after replacing each before text with the
// after text that follows it in edits.
func covLegacy(t *testing.T, edits ...string) (Set, error) {
	t.Helper()
	raw, err := os.ReadFile("testdata/laser.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "$schema: "+EcosystemSchema+"\n", "", 1)
	source := covEdit(t, text, edits...)
	return Read("testdata/legacy.yaml", []byte(source))
}

// covEdit replaces each before text with the after text that follows it,
// failing the test when a before text is not there.
func covEdit(t *testing.T, text string, edits ...string) string {
	t.Helper()
	for i := 0; i+1 < len(edits); i += 2 {
		if !strings.Contains(text, edits[i]) {
			t.Fatalf("the text has no %q to change", edits[i])
		}
		text = strings.Replace(text, edits[i], edits[i+1], 1)
	}
	return text
}

// covRefused checks that reading fails with a message holding the text.
func covRefused(t *testing.T, name string, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("%s: error %v, want one holding %q", name, err, want)
	}
}

// covMist is the weather block of a small set that has a mist and no terrain.
const covMist = `  weather:
    mist:
      kind: prescribed-mist
      extinction: {value: 0.006, unit: m^-1}
      depth: {value: 20, unit: m}
      coverage: 0.4
      seed: 23
`

// covSmall is a small legacy set with the lines added under scene.
func covSmall(add string) (Set, error) {
	return Read("small.yaml", []byte("scene:\n  name: t\n  light: ambient\n"+add))
}

// TestAtmosphereLayersAreCheckedWhereTheyAreRead checks the refusals of an
// atmosphere block that names layers, and the ones that wait for the whole
// set to be read.
func TestAtmosphereLayersAreCheckedWhereTheyAreRead(t *testing.T) {
	layer := "      deck:\n        weather: mist\n        extent: terrain\n        albedo: 0.5\n"
	_, err := covSmall(covMist + "  atmosphere:\n    layers: {}\n")
	covRefused(t, "no layers", err, "atmosphere needs at least one layer")
	_, err = covSmall(covMist + "  atmosphere:\n    layers:\n      deck:\n        weather: mist\n        extent: terrain\n")
	covRefused(t, "layer without albedo", err, "albedo")
	_, err = covSmall(covMist + "  atmosphere:\n    layers:\n" + layer)
	covRefused(t, "terrain layer without terrain", err, "a terrain atmosphere needs terrain")
	_, err = covSmall(covMist + "  atmosphere:\n    layers:\n" + strings.Replace(layer, "weather: mist", "weather: fog", 1))
	covRefused(t, "unknown weather", err, `atmosphere names unknown weather "fog"`)
}

// TestWorldAtmosphereWithNoGroundIsFlatAir checks that a version 2 layer
// over the world, in a set that has no globe and no terrain, is read, and
// is the set's flat air. Before flat air it was refused, as a world layer
// with no globe; a set with terrain and no globe still is, in
// TestWorldAtmosphereOverTerrainNeedsAGlobe.
func TestWorldAtmosphereWithNoGroundIsFlatAir(t *testing.T) {
	made, err := readV2(covMist + "  atmosphere:\n    layers:\n      deck:\n        weather: mist\n        extent: world\n        albedo: 0.5\n")
	if err != nil {
		t.Fatal(err)
	}
	flat := made.FlatAir()
	if len(flat) != 1 || flat[0].Weather() != "mist" || flat[0].Albedo() != 0.5 {
		t.Errorf("flat air %v, want the one layer of mist at albedo 0.5", flat)
	}
}

// covRegion is a region block for the legacy fixture, with the lines added
// under the region's name.
func covRegion(add string) string {
	return `  ecosystems:
    mixed grove:
      seed: 3
      cell: {value: 100, unit: m}
      density: {value: 1, unit: ha^-1}
      variety: 2
      bounds: {west: 0, east: 200, north: 0, south: -200}
      mixture: {redwoods: 1}
` + add + "  terrain:"
}

// TestEcosystemKeepsItsControlsAndHabitat checks each accessor of a region
// against the values the set wrote.
func TestEcosystemKeepsItsControlsAndHabitat(t *testing.T) {
	made, err := covLegacy(t, "  terrain:", covRegion(`      complexity: demo
      above: 10
      below: 500
      exclude: {around: the observatory, clearance: {value: 30, unit: m}}
      atmosphere: {weather: grove mist, extent: canopy-and-ground, fade-at-boundary: true}
`))
	if err != nil {
		t.Fatal(err)
	}
	regions := made.Ecosystems()
	if len(regions) != 1 {
		t.Fatalf("%d regions", len(regions))
	}
	region := regions[0]
	if region.Name() != "mixed grove" || region.Seed() != 3 || region.Complexity() != "demo" {
		t.Errorf("name %q, seed %d, complexity %q", region.Name(), region.Seed(), region.Complexity())
	}
	if above, below := region.Band(); above != 10 || below != 500 {
		t.Errorf("band %v to %v", above, below)
	}
	if around, clearance := region.Exclude(); around != "the observatory" || clearance != 30 {
		t.Errorf("exclusion %q %v", around, clearance)
	}
	if weather, fade := region.Atmosphere(); weather != "grove mist" || !fade {
		t.Errorf("atmosphere %q %v", weather, fade)
	}
}

// TestEcosystemRefusalsSayWhatIsWrong checks the refusals of the region
// reader and of the checks that wait for the whole set.
func TestEcosystemRefusalsSayWhatIsWrong(t *testing.T) {
	cases := []struct {
		name  string
		edits []string
		want  string
	}{
		{"weight below zero", []string{"  terrain:", strings.Replace(covRegion(""), "{redwoods: 1}", "{redwoods: -1}", 1)}, "redwoods"},
		{"no seed", []string{"  terrain:", strings.Replace(covRegion(""), "      seed: 3\n", "", 1)}, "seed"},
		{"empty habitat", []string{"  terrain:", covRegion("      above: 5\n      below: 5\n")}, "habitat height band is empty"},
		{"exclusion without clearance", []string{"  terrain:", covRegion("      exclude: {around: the observatory}\n")}, "clearance"},
		{"atmosphere without fade", []string{"  terrain:", covRegion("      atmosphere: {weather: grove mist, extent: canopy-and-ground}\n")}, "fade-at-boundary"},
		{"a stand's name", []string{"  terrain:", strings.Replace(covRegion(""), "mixed grove", "the forest", 1)}, "already used by a population"},
		{"unknown exclusion", []string{"  terrain:", covRegion("      exclude: {around: the shed, clearance: {value: 1, unit: m}}\n")}, "unknown exclusion the shed"},
		{"unknown mist", []string{"  terrain:", covRegion("      atmosphere: {weather: fog, extent: canopy-and-ground, fade-at-boundary: false}\n")}, "unknown mist fog"},
		{"unknown recipe", []string{"  terrain:", strings.Replace(covRegion(""), "{redwoods: 1}", "{willows: 1}", 1)}, "needs a solid recipe willows"},
	}
	for _, each := range cases {
		_, err := covLegacy(t, each.edits...)
		covRefused(t, each.name, err, each.want)
	}
}

// TestEcosystemNeedsTerrain checks that a region with no terrain under it is
// refused.
func TestEcosystemNeedsTerrain(t *testing.T) {
	text := covRegion("")
	text = strings.TrimSuffix(text, "  terrain:")
	_, err := covSmall(`  recipes:
    redwoods:
      kind: boulder
` + text)
	covRefused(t, "no terrain", err, "ecosystem mixed grove needs terrain")
}

// TestRecipeSeasonsApplyOnlyToPlants checks that a plant recipe keeps a
// season and a recipe of any other kind is refused with it.
func TestRecipeSeasonsApplyOnlyToPlants(t *testing.T) {
	made, err := covSmall(`  recipes:
    bare:
      plant: {family: seasonal, file: ../manufacture/plants/seasonal/profiles.yaml, species: alder}
      season: winter
`)
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := made.Recipe("bare")
	if !found || !recipe.IsPlant() || recipe.PlantSpecies() != "alder" {
		t.Fatal("seasonal plant recipe lost")
	}
	row, err := recipe.Construction()
	if err != nil || row == nil {
		t.Fatalf("construction %v, %v", row, err)
	}
	_, err = covSmall("  recipes:\n    rock:\n      kind: boulder\n      season: winter\n")
	covRefused(t, "season on a boulder", err, "has no foliage season")
}

// covDomainOnly makes the version 2 schema accept any text for the rest of
// the test, so that the domain reader alone judges a version 2 set.
func covDomainOnly(t *testing.T) {
	t.Helper()
	permissive, err := yamlfile.NewSchema(ecosystemV2Schema, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	previous := ecosystemV2
	ecosystemV2 = permissive
	t.Cleanup(func() { ecosystemV2 = previous })
}

// TestVersionTwoSchemaFailureIsPreserved checks that a schema which could
// not be built is the error of every version 2 read.
func TestVersionTwoSchemaFailureIsPreserved(t *testing.T) {
	_, invalid := yamlfile.NewSchema(ecosystemV2Schema, []byte(`{"type":"not-a-type"}`))
	if invalid == nil {
		t.Fatal("invalid schema compiled")
	}
	previous := ecosystemV2Error
	ecosystemV2Error = invalid
	t.Cleanup(func() { ecosystemV2Error = previous })
	if _, err := readV2(""); !errors.Is(err, invalid) {
		t.Fatalf("lost schema construction cause: %v", err)
	}
}

// TestLaserLimitsFailLoudlyWhenTheirBookIsBroken checks that a limit
// that cannot be looked up stops the program, as the code promises.
func TestLaserLimitsFailLoudlyWhenTheirBookIsBroken(t *testing.T) {
	previous := laserLimits
	t.Cleanup(func() { laserLimits = previous })
	cases := map[string]func() (numbers.Numbers, error){
		"unreadable": func() (numbers.Numbers, error) { return nil, errors.New("book gone") },
		"missing name": func() (numbers.Numbers, error) {
			return numbers.Read("other.yaml", []byte("numbers:\n  - name: other\n    value: 1\n    says: one\n    kind: picked\n    by: test\n"))
		},
	}
	for name, book := range cases {
		laserLimits = book
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: laserLimit did not panic", name)
				}
			}()
			laserLimit("minimum-radius")
		}()
	}
}

// TestUndrawnFeaturesKeepTheirPlaceInTheFile checks the feature, the word,
// the line and the column kept for a key that is read and not yet drawn.
func TestUndrawnFeaturesKeepTheirPlaceInTheFile(t *testing.T) {
	text := v2Text(defaultFilm, one("a", "star", ""))
	made, err := Read("t.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	var kinds []Undrawn
	for _, each := range made.Undrawn() {
		if each.Feature() == "thing.kind" {
			kinds = append(kinds, each)
		}
	}
	if len(kinds) != 1 {
		t.Fatalf("%d undrawn kinds: %+v", len(kinds), made.Undrawn())
	}
	at := strings.Index(text, "kind: star")
	line := strings.Count(text[:at], "\n") + 1
	column := at - strings.LastIndex(text[:at], "\n")
	got := kinds[0]
	if got.Feature() != "thing.kind" || got.Says() != "star" || got.Line() != line || got.Column() != column {
		t.Errorf("undrawn %s %q at %d:%d, want thing.kind \"star\" at %d:%d",
			got.Feature(), got.Says(), got.Line(), got.Column(), line, column)
	}
}

// TestVersionTwoDomainRefusalsSayWhatIsWrong checks refusals the schema
// would otherwise make first, so only the domain reader can be judged.
func TestVersionTwoDomainRefusalsSayWhatIsWrong(t *testing.T) {
	covDomainOnly(t)
	cases := []struct {
		name string
		add  string
		want string
	}{
		{"east without north", "  setting: space\n  set:\n    things:\n      a:\n        kind: can\n        seed: 1\n        east: 5\n", "east and north together, or neither"},
		{"unknown kind", one("a", "wizard", ""), "kind takes"},
		{"a corona that reaches nowhere", one("a", "star", "        corona: {reach: 0, streamers: 1, seed: 1}\n"), "a corona reaches past the star"},
		{"a turning that is not yes or no", one("a", "lantern", "        turning: sometimes\n"), "turning"},
		{"a neon past one", one("a", "nebula", "        neon: 1.5\n"), "neon"},
		{"a sense of two", one("a", "culture-orbital", "        sense: 2\n"), "sense"},
		{"a sense of nought", one("a", "culture-orbital", "        sense: 0\n"), "sense takes 1 or -1"},
		{"an extent past the world", "  atmosphere:\n    layers:\n      deck:\n        weather: mist\n        extent: sky\n        albedo: 0.5\n", "extent"},
		{"a space observatory with no craft", "  recipes:\n    scope:\n      kind: space-observatory\n", "space observatory recipe says"},
	}
	for _, each := range cases {
		add := each.add
		if strings.Contains(add, "atmosphere") {
			add = covMist + add
		}
		_, err := readV2(add)
		covRefused(t, each.name, err, each.want)
	}
}

// TestAKeyOfAnotherKindIsRefusedEvenAfterAValueWithItsName checks that a
// value which only spells another kind's key does not hide that key.
func TestAKeyOfAnotherKindIsRefusedEvenAfterAValueWithItsName(t *testing.T) {
	covDomainOnly(t)
	_, err := readV2(one("a", "can", "        stands_in_for: radius\n        radius: {value: 1, unit: m}\n"))
	covRefused(t, "radius on a can", err, "radius is a key of a culture-orbital")
	if _, err := readV2(one("a", "can", "        stands_in_for: radius\n")); err != nil {
		t.Errorf("a value spelling a key was refused: %v", err)
	}
}

// TestAttitudeAndLookMustSayOneThing checks that a pointing or a look that
// says both a thing and a place on the sky is refused, and each alone is
// read.
func TestAttitudeAndLookMustSayOneThing(t *testing.T) {
	covDomainOnly(t)
	ship := func(more string) string {
		return "  set:\n    things:\n" + base + satellite("scope", "        around: base\n"+more)
	}
	_, err := readV2(ship("        attitude: {point: {sky: {ra: 1, dec: 1}, object: base}}\n"))
	covRefused(t, "point at both", err, "a point says a place on the sky or a thing, one of them")
	if _, err := readV2(ship("        attitude: {point: {object: base}}\n")); err != nil {
		t.Errorf("a point at a thing: %v", err)
	}
	moves := func(look string) (Set, error) {
		return withCamera("      moves:\n        - look: " + look + "\n          duration: {value: 4, unit: s}\n")
	}
	_, err = moves("{object: sun, sky: {ra: 1, dec: 1}}")
	covRefused(t, "look at both", err, "a look says a thing or a place on the sky, one of them")
	_, err = moves("{offset: {value: 1, unit: m}}")
	covRefused(t, "look at neither", err, "a look says a thing or a place on the sky, one of them")
	_, err = withCamera("      moves: []\n")
	covRefused(t, "no moves", err, "moves takes a list of at least one move")
	_, err = withCamera("      moves:\n        - look: {object: sun}\n")
	covRefused(t, "move with no duration", err, "duration")
}
