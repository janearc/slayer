package set_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/recipe-construction.yaml
var opticalConstructionRaw []byte

type opticalConstructionContract struct {
	Before, After string
	Diameter      float64
	Seed          uint64
}

func TestEmptyRecipeConstructionReturnsOnlyAnError(test *testing.T) {
	var recipe set.Recipe
	row, err := recipe.Construction()
	if row != nil || err == nil {
		test.Fatal("empty recipe exposed a construction", row, err)
	}
}

func TestOpticalRecipeConstructionRetainsItsDimensions(test *testing.T) {
	var data opticalConstructionContract
	if err := yamlfile.Read("recipe-construction.yaml",
		opticalConstructionRaw, &data); err != nil {
		test.Fatal(err)
	}
	for _, raw := range laserSources() {
		source := strings.ReplaceAll(raw, data.Before, data.After)
		if source == raw {
			test.Fatal("non-default enclosure was not installed")
		}
		scene, err := set.Read("optical.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		data.assertSolid(test, scene)
	}
}

func (data opticalConstructionContract) assertSolid(test *testing.T,
	scene set.Set,
) {
	test.Helper()
	recipe, found := scene.Recipe("ridge telescope")
	if !found {
		test.Fatal("missing optical recipe")
	}
	row, err := recipe.Construction()
	if err != nil {
		test.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(*row,
		manufacture.Sow(data.Seed), "quick")
	if err != nil {
		test.Fatal(err)
	}
	bounds := made.Bounds()
	if bounds.Max.X-bounds.Min.X != data.Diameter {
		test.Fatal("construction lost the enclosure diameter",
			bounds, data.Diameter)
	}
}
