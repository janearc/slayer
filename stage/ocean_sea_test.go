package stage_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// edgelessText is a set whose ocean says no extent and no samples, and
// whatever else it is given among its keys.
func edgelessText(t *testing.T, more string) string {
	t.Helper()
	return strings.Replace(waterMaterialFixture(t), "    materials:\n",
		"    ocean:\n"+
			"      material: grass\n"+
			"      middle: [0, 0, 0]\n"+
			"      sea:\n"+
			"        wind: {value: 7, unit: m/s}\n"+
			"        fetch: {value: 12000, unit: m}\n"+
			"        depth: {value: 0, unit: m}\n"+
			"        toward: {value: 0.5, unit: rad}\n"+
			"      seed: 3\n"+
			"      time: {value: 0, unit: s}\n"+
			"      source: eye\n"+more+
			"    materials:\n", 1)
}

// edgelessSea is the ocean of that set.
func edgelessSea(t *testing.T, more string) set.WaveSurface {
	t.Helper()
	text := []byte(edgelessText(t, more))
	read, err := set.Read("edgeless-sea.yaml", text)
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	surface, has := said.OceanSurface()
	if !has {
		t.Fatal("the set has no ocean")
	}
	return surface
}

// seaMaterial is the material of the sea with no edge a set stages.
func seaMaterial(t *testing.T, more string) *thing.Material {
	t.Helper()
	text := []byte(edgelessText(t, more))
	read, err := set.Read("sea-material.yaml", text)
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range world.Things {
		if _, sea := item.Shape.(*arid.Sea); sea {
			return item.Material
		}
	}
	t.Fatal("the world holds no sea with no edge")
	return nil
}

// The sea's glint is Cox and Munk's when the set says none: its slope
// variance is the budget's unresolved part, the same as when it says so.
func TestSeaGlintIsCoxAndMunkByDefault(t *testing.T) {
	plain := seaMaterial(t, "")
	said := seaMaterial(t, "      glint: {model: cox-munk}\n")
	if !(plain.SlopeVariance > 0) {
		t.Fatalf("a sea that says no glint has slope variance %v",
			plain.SlopeVariance)
	}
	if plain.SlopeVariance != said.SlopeVariance {
		t.Errorf("slope variance %v with no glint, %v for cox-munk",
			plain.SlopeVariance, said.SlopeVariance)
	}
}

// Anti-test: an artistic glint is kept as authored: its roughness is the
// material's Satin and no Cox and Munk slope is put in.
func TestArtisticGlintIsKeptOnTheSea(t *testing.T) {
	const roughness = 0.2
	material := seaMaterial(t,
		"      glint: {model: artistic, roughness: 0.2}\n")
	if material.Satin != roughness || material.SlopeVariance != 0 {
		t.Errorf("satin %v, slope variance %v, want %v and nought",
			material.Satin, material.SlopeVariance, roughness)
	}
}

// A swell the set says reaches the sea oceantex is given, and a set that
// says none gives it none.
func TestSeaOfCarriesTheSwell(t *testing.T) {
	plain, err := stage.SeaOf(edgelessSea(t, ""))
	if err != nil || plain.Swell != nil {
		t.Fatalf("a set with no swell: swell %v, error %v", plain.Swell, err)
	}
	swelled, err := stage.SeaOf(edgelessSea(t, "      swell:\n"+
		"        height: {value: 2.5, unit: m}\n"+
		"        period: {value: 12, unit: s}\n"+
		"        toward: {value: 1.5, unit: rad}\n"))
	if err != nil || swelled.Swell == nil {
		t.Fatalf("a set with a swell: swell %v, error %v", swelled.Swell, err)
	}
	const within = 1e-9
	if math.Abs(swelled.Swell.Height()-2.5) > within ||
		math.Abs(swelled.Swell.Period()-12) > within ||
		math.Abs(swelled.Swell.Toward()-1.5) > within {
		t.Errorf("swell %v m, %v s, %v rad", swelled.Swell.Height(),
			swelled.Swell.Period(), swelled.Swell.Toward())
	}
	if !(swelled.Height() > plain.Height()) {
		t.Errorf("height %v with a swell, %v without", swelled.Height(),
			plain.Height())
	}
}
