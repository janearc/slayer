package set

import (
	"crypto/sha256"
	"fmt"

	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// plantRecipe reads the named species from an external book exactly once.
// The bytes and checked book stay with the set for staging and cache keys.
func (reading reader) plantRecipe(name, block *yaml.Node) error {
	var file, species, season, family, representation string
	if err := reading.keys(block, map[string]step{
		"plant": reading.block(map[string]step{
			"family":         reading.word(&family, "seasonal", "conifer", "fern", "cactus"),
			"file":           reading.text(&file),
			"species":        reading.text(&species),
			"representation": reading.word(&representation, "shoot"),
		}),
		"season": reading.word(&season, manufacture.SeasonNames()...),
	}, nil); err != nil {
		return err
	}
	if err := reading.all(block, "a plant recipe", "plant"); err != nil {
		return err
	}
	if file == "" || species == "" {
		return fmt.Errorf("%s: plant recipe %s needs file and species", reading.file, name.Value)
	}
	if family == "" {
		family = "seasonal"
	}
	var path, builtin string
	var raw []byte
	var err error
	if file == BuiltinProfileBook {
		builtin = "plant:" + family
		path, raw, err = manufacture.EmbeddedPlantBook(family)
	} else {
		path, raw, err = reading.profileFile(file)
	}
	if err != nil {
		return err
	}
	book, err := manufacture.ReadPlantBook(family, path, raw)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	options := manufacture.PlantOptions{Species: species}
	if says(block, "season") {
		options.Season = &season
	}
	if says(valueNode(block, "plant"), "representation") {
		options.Representation = &representation
	}
	row, err := book.New(options)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	profilePath := path
	if builtin != "" {
		profilePath = ""
	}
	reading.set.recipes[name.Value] = Recipe{kind: "plant-profile", profileFamily: family,
		profileSpecies: species, profileKind: &row, profileDigest: sha256.Sum256(raw), profilePath: profilePath, profileBuiltin: builtin}

	return nil
}
