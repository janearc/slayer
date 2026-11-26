package set

import (
	"crypto/sha256"
	"fmt"

	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// algaRecipe freezes the checked species book with the recipe.
func (reading reader) algaRecipe(name, block *yaml.Node) error {
	var family, file, species, condition string
	var forest bool
	if err := reading.keys(block, map[string]step{
		"alga": reading.block(map[string]step{
			"family":    reading.word(&family, "kelp"),
			"file":      reading.text(&file),
			"species":   reading.text(&species),
			"condition": reading.text(&condition),
			"forest":    reading.boolean(&forest),
		}),
	}, nil); err != nil {
		return err
	}
	if err := reading.all(block, "an alga recipe", "alga"); err != nil {
		return err
	}
	if family == "" || file == "" || species == "" {
		return fmt.Errorf("%s: alga recipe %s needs family, file and species", reading.file, name.Value)
	}
	path, raw, err := reading.profileFile(file)
	if err != nil {
		return err
	}
	book, err := manufacture.ReadKelpProfiles(path, raw)
	if err != nil {
		return err
	}
	row, err := book.NewKelp(manufacture.KelpOptions{Species: species, Condition: condition, Forest: forest})
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	reading.set.recipes[name.Value] = Recipe{kind: "alga-profile", profileFamily: family,
		profileSpecies: species, profileKind: &row, profileDigest: sha256.Sum256(raw), profilePath: path}
	return nil
}
