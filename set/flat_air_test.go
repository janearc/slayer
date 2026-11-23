package set

import (
	"os"
	"strings"
	"testing"
)

// worldLayer is an atmosphere block of one layer over the world, for the
// laser fixture, whose weather block names grove mist.
const worldLayer = "  atmosphere:\n    layers:\n      deck:\n" +
	"        weather: grove mist\n        extent: world\n" +
	"        albedo: 0.5\n"

// overTerrain is the laser fixture, which has terrain, as a version 2
// file with the text added before its terrain block.
func overTerrain(t *testing.T, add string) (Set, error) {
	t.Helper()
	raw, err := os.ReadFile("testdata/laser.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), EcosystemSchema, ecosystemV2Schema, 1)
	text = covEdit(t, text, "  terrain:\n", add+"  terrain:\n")
	return Read("over-terrain.yaml", []byte(text))
}

// TestWorldAtmosphereOverTerrainNeedsAGlobe is the anti-test of flat air:
// a layer over the world in a set that has terrain and no globe is not
// flat air, and is still refused, as it was before flat air.
func TestWorldAtmosphereOverTerrainNeedsAGlobe(t *testing.T) {
	_, err := overTerrain(t, worldLayer)
	covRefused(t, "world layer over terrain", err,
		"a world atmosphere needs a globe")
}

// TestASetWithTerrainHasNoFlatAir checks that the layers of a set with
// terrain are the terrain's: its flat air is empty.
func TestASetWithTerrainHasNoFlatAir(t *testing.T) {
	made, err := covLegacy(t)
	if err != nil {
		t.Fatal(err)
	}
	if flat := made.FlatAir(); len(flat) != 0 {
		t.Errorf("a set with terrain has flat air %v", flat)
	}
}

// TestFlatAirBelongsToTheCaller checks that changing the slice FlatAir
// gives changes nothing in the set.
func TestFlatAirBelongsToTheCaller(t *testing.T) {
	made, err := readV2(covMist + "  atmosphere:\n    layers:\n" +
		"      deck:\n        weather: mist\n        extent: world\n" +
		"        albedo: 0.5\n")
	if err != nil {
		t.Fatal(err)
	}
	flat := made.FlatAir()
	flat[0] = AtmosphereLayer{}
	if again := made.FlatAir(); again[0].Weather() != "mist" {
		t.Errorf("the caller changed the set's layers: %v", again)
	}
}

// TestATerrainLayerOfASetWithNoTerrainIsStillRefused checks that a layer
// that says terrain extent is not flat air: with no terrain it is refused
// as it was.
func TestATerrainLayerOfASetWithNoTerrainIsStillRefused(t *testing.T) {
	_, err := readV2(covMist + "  atmosphere:\n    layers:\n" +
		"      deck:\n        weather: mist\n        extent: terrain\n" +
		"        albedo: 0.5\n")
	covRefused(t, "terrain layer with no terrain", err,
		"a terrain atmosphere needs terrain")
}
