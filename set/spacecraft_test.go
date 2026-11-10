package set_test

import (
	_ "embed"
	"math"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/spacecraft-contract.yaml
var spacecraftReaderRaw []byte

type spacecraftReaderContract struct {
	Seed                             uint64
	Block, Name, Placement           string
	Height, Aperture, Top, Tolerance float64
	Invalid                          []struct{ Before, After string }
}

func readSpacecraftReader(test *testing.T) spacecraftReaderContract {
	test.Helper()
	var data spacecraftReaderContract
	if err := yamlfile.Read("testdata/spacecraft-contract.yaml", spacecraftReaderRaw, &data); err != nil {
		test.Fatal(err)
	}
	return data
}

func TestSpacecraftRecipeAndHeightAgreeAcrossBothReaders(test *testing.T) {
	data := readSpacecraftReader(test)
	for _, raw := range laserSources() {
		source := strings.ReplaceAll(raw, "  recipes:\n", indentAtmosphere(data.Block))
		source = strings.ReplaceAll(source, "recipe: ridge telescope", data.Placement)
		scene, err := set.Read("spacecraft.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		data.assertRecipe(test, scene)
		placed := false
		for _, object := range scene.Things() {
			placed = placed || object.Recipe() == data.Name
			if object.Recipe() == data.Name && object.Height() != data.Height {
				test.Fatal("lost spacecraft lift", object.Height(), data.Height)
			}
		}
		if !placed {
			test.Fatal("missing spacecraft placement")
		}
		for _, invalid := range data.Invalid {
			broken := strings.ReplaceAll(source, invalid.Before, invalid.After)
			if broken == source {
				test.Fatal("ineffective spacecraft fault", invalid)
			}
			if _, err := set.Read("bad-spacecraft.yaml", []byte(broken)); err == nil {
				test.Fatal("accepted invalid spacecraft", invalid)
			}
		}
	}
}

func (data spacecraftReaderContract) assertRecipe(test *testing.T, scene set.Set) {
	test.Helper()
	recipe, found := scene.Recipe(data.Name)
	if !found {
		test.Fatal("missing spacecraft recipe")
	}
	row, err := recipe.Construction()
	if err != nil {
		test.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(data.Seed), "quick")
	if err != nil {
		test.Fatal(err)
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	found = false
	for _, part := range parts {
		if part.Material.Name != "primary mirror" {
			continue
		}
		found = true
		bounds := part.Shape.Bounds()
		if math.Abs(bounds.Max.X-bounds.Min.X-data.Aperture) > data.Tolerance {
			test.Fatal("YAML aperture did not reach the mirror", bounds)
		}
	}
	if !found || math.Abs(made.Bounds().Max.Y-data.Top) > data.Tolerance {
		test.Fatal("YAML design did not reach the blueprint", made.Bounds())
	}
}
