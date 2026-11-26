package set

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// BuiltinProfileBook selects the versioned profile book embedded in Slayer.
const BuiltinProfileBook = "@builtin"

// profileFile snapshots a bounded book relative to the set that names it.
func (reading reader) profileFile(file string) (string, []byte, error) {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(reading.file), path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("%s: profile: %w", reading.file, err)
	}
	raw, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	closeErr := f.Close()
	if err != nil {
		return "", nil, fmt.Errorf("%s: profile: %w", path, err)
	}
	if closeErr != nil {
		return "", nil, fmt.Errorf("%s: profile: %w", path, closeErr)
	}
	if len(raw) > 1<<20 {
		return "", nil, fmt.Errorf("%s: profile exceeds 1 MiB", path)
	}
	return path, raw, nil
}

// animalRecipe selects a checked species without using its name as behavior.
func (reading reader) animalRecipe(name, block *yaml.Node) error {
	var family, file, species, pose, representation, study string
	var count int
	var seconds, phase float64
	if err := reading.keys(block, map[string]step{
		"animal": reading.block(map[string]step{
			"family": reading.word(&family, "fish", "bird", "theropod", "pinniped", "squid", "whale", "reptile", "snake", "sauropod"),
			"file":   reading.text(&file), "species": reading.text(&species),
			"pose":           reading.word(&pose, "swim", "glide", "bones", "standing", "gliding", "outline", "hauled", "hauled-out", "lying", "swimming", "spy-hop"),
			"count":          reading.integer(&count, 0, 5000),
			"representation": reading.word(&representation, "skin", "bones", "volumes"),
			"study": func(name, value *yaml.Node) error {
				if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
					return reading.refuse(value, "%s takes a study name", name.Value)
				}
				return reading.text(&study)(name, value)
			},
			"phase": func(name, value *yaml.Node) error {
				if value.Kind != yaml.ScalarNode || (value.Tag != "!!int" && value.Tag != "!!float") {
					return reading.refuse(value, "%s takes a number", name.Value)
				}
				return reading.number(&phase, -100000, 100000)(name, value)
			},
			"seconds": func(name, value *yaml.Node) error {
				if value.Kind != yaml.ScalarNode || (value.Tag != "!!int" && value.Tag != "!!float") {
					return reading.refuse(value, "%s takes a number", name.Value)
				}
				return reading.number(&seconds, -100000, 100000)(name, value)
			},
		}),
	}, nil); err != nil {
		return err
	}
	if err := reading.all(block, "an animal recipe", "animal"); err != nil {
		return err
	}
	if family == "" || file == "" || species == "" {
		return fmt.Errorf("%s: animal recipe %s needs family, file and species", reading.file, name.Value)
	}
	spec := valueNode(block, "animal")
	var path, builtin string
	var raw []byte
	var err error
	if file == BuiltinProfileBook {
		builtin = "animal:" + family
		path, raw, err = manufacture.EmbeddedAnimalBook(family)
	} else {
		path, raw, err = reading.profileFile(file)
	}
	if err != nil {
		return err
	}
	book, err := manufacture.ReadAnimalBook(family, path, raw)
	if err != nil {
		return err
	}
	options := manufacture.AnimalOptions{Species: species}
	if says(spec, "pose") {
		options.Pose = &pose
	}
	if says(spec, "count") {
		options.Count = &count
	}
	if says(spec, "seconds") {
		options.Seconds = &seconds
	}
	if says(spec, "phase") {
		options.Phase = &phase
	}
	if says(spec, "representation") {
		options.Representation = &representation
	}
	if says(spec, "study") {
		options.Study = &study
	}
	row, err := book.New(options)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	profilePath := path
	if builtin != "" {
		profilePath = ""
	}
	reading.set.recipes[name.Value] = Recipe{kind: "animal-profile", profileFamily: family, profileSpecies: species,
		profileKind: &row, profileDigest: sha256.Sum256(raw), profilePath: profilePath, profileBuiltin: builtin}
	return nil
}
