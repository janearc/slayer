package set

import (
	"math"

	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// spacecraftRecipe checks the template, dimensions and nested vocabulary.
func (reading reader) spacecraftRecipe(name, block *yaml.Node) error {
	recipe := Recipe{kind: "space-observatory"}
	designs := manufacture.SpacecraftDesigns()
	maximum := math.Sqrt(math.MaxFloat64)
	controls := map[string]step{
		"design": reading.word(&recipe.design, designs...),
		"aperture": reading.quantity(&recipe.aperture, "m",
			sceneScalarZero, maximum),
	}
	body := reading.block(controls)
	err := reading.keys(block, map[string]step{
		"kind":       reading.only("space-observatory"),
		"spacecraft": body,
	}, nil)
	if err != nil {
		return err
	}
	requiredError := reading.all(block, "space observatory recipe",
		"kind", "spacecraft")
	if requiredError != nil {
		return requiredError
	}
	if _, err := recipe.Construction(); err != nil {
		return reading.refuse(block, "%v", err)
	}
	reading.set.recipes[name.Value] = recipe
	return nil
}

// sceneScalarZero is defined as the additive identity of scalar inputs.
const sceneScalarZero = 0

// scenePositiveInfinity is defined as the positive sign for math.Inf.
const scenePositiveInfinity = 1
