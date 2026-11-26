package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/janearc/slayer/manufacture"
)

// objectPlant is an explicit checked model source, never a factory alias.
type objectPlant struct {
	Family         string  `yaml:"family" json:"family"`
	File           string  `yaml:"file" json:"file"`
	SHA256         string  `yaml:"sha256" json:"sha256,omitempty"`
	Species        string  `yaml:"species" json:"species"`
	Season         *string `yaml:"season" json:"season,omitempty"`
	Representation *string `yaml:"representation" json:"representation,omitempty"`
	binding        *plantBinding
}

type plantBinding struct{ identity, source string }

type modelSourceRecord struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type placedPlant struct {
	Family         string  `json:"family"`
	Species        string  `json:"species"`
	SHA256         string  `json:"sha256"`
	Season         *string `json:"season,omitempty"`
	Representation *string `json:"representation,omitempty"`
}

const maxPlantBookBytes = 1 << 20
const maxPlantSourceBytes = 16 << 20

// bindPlantObjects reads each path once, validates the exact captured bytes,
// and binds each option set to a content-derived prototype identity.
func (plan *plan) bindPlantObjects(sceneDir string) error {
	if plan.Site == nil {
		return nil
	}
	paths := make(map[string][]byte)
	books := make(map[string]*manufacture.PlantBook)
	plan.modelSources = make(map[string][]byte)
	plan.Site.plantRows = make(map[string]manufacture.Kind)
	total := 0
	for i := range plan.Site.Objects {
		plant := plan.Site.Objects[i].Plant
		if plant == nil {
			continue
		}
		path := plant.File
		if !filepath.IsAbs(path) {
			path = filepath.Join(sceneDir, path)
		}
		path = filepath.Clean(path)
		raw, seen := paths[path]
		if !seen {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("island: plant book %q: %w", path, err)
			}
			raw, err = io.ReadAll(io.LimitReader(file, maxPlantBookBytes+1))
			closeErr := file.Close()
			if err != nil {
				return fmt.Errorf("island: plant book %q: %w", path, err)
			}
			if closeErr != nil {
				return fmt.Errorf("island: plant book %q: %w", path, closeErr)
			}
			if len(raw) == 0 || len(raw) > maxPlantBookBytes {
				return fmt.Errorf("island: empty or oversized plant book %q", path)
			}
		}
		var source string
		var err error
		raw, source, err = keepPlantSource(plan.modelSources, raw, &total)
		if err != nil {
			return err
		}
		if plant.SHA256 != "" && plant.SHA256 != source {
			return fmt.Errorf("island: plant book hash changed: %s", path)
		}
		paths[path] = raw
		bookKey := plant.Family + "/" + source
		book := books[bookKey]
		if book == nil {
			var err error
			book, err = manufacture.ReadPlantBook(plant.Family, path, raw)
			if err != nil {
				return err
			}
			books[bookKey] = book
		}
		row, err := book.New(manufacture.PlantOptions{Species: plant.Species, Season: plant.Season, Representation: plant.Representation})
		if err != nil {
			return fmt.Errorf("island: object %q plant: %w", plan.Site.Objects[i].Name, err)
		}
		identityBytes, err := json.Marshal(struct {
			Source, Family, Species string
			Season, Representation  *string
		}{source, plant.Family, plant.Species, plant.Season, plant.Representation})
		if err != nil {
			return err
		}
		identitySum := sha256.Sum256(identityBytes)
		identity := hex.EncodeToString(identitySum[:])
		plant.binding = &plantBinding{identity: identity, source: source}
		plan.Site.plantRows[identity] = row
	}
	return nil
}

// keepPlantSource retains only one copy of identical content across paths.
func keepPlantSource(sources map[string][]byte, raw []byte, total *int) ([]byte, string, error) {
	sum := sha256.Sum256(raw)
	source := hex.EncodeToString(sum[:])
	if held, exists := sources[source]; exists {
		return held, source, nil
	}
	if len(raw) > maxPlantSourceBytes-*total {
		return nil, "", fmt.Errorf("island: plant books exceed snapshot budget")
	}
	*total += len(raw)
	sources[source] = raw
	return raw, source, nil
}

func modelFilename(source string) string { return "models/" + source + ".yaml" }

// modelSourceRecords lists the exact books kept for a render, stably sorted.
func (plan *plan) modelSourceRecords() []modelSourceRecord {
	names := make([]string, 0, len(plan.modelSources))
	for source := range plan.modelSources {
		names = append(names, source)
	}
	slices.Sort(names)
	var records []modelSourceRecord
	for _, source := range names {
		records = append(records, modelSourceRecord{File: modelFilename(source), SHA256: source})
	}
	return records
}

func (plan *plan) writeModelSources(destination string) error {
	if len(plan.modelSources) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(destination, "models"), 0755); err != nil {
		return err
	}
	for source, raw := range plan.modelSources {
		if err := os.WriteFile(filepath.Join(destination, modelFilename(source)), raw, 0644); err != nil {
			return err
		}
	}
	return nil
}

// replayScene writes frozen source bytes and gives the scene paths local to
// its own output directory. An ordinary kind-only scene stays byte-identical.
func (plan *plan) replayScene(destination string) ([]byte, error) {
	if len(plan.modelSources) == 0 {
		return plan.raw, nil
	}
	if err := plan.writeModelSources(destination); err != nil {
		return nil, err
	}
	replay := *plan
	site := *plan.Site
	site.Objects = slices.Clone(plan.Site.Objects)
	for i := range site.Objects {
		plant := site.Objects[i].Plant
		if plant == nil {
			continue
		}
		if plant.binding == nil {
			return nil, fmt.Errorf("island: plant object has no snapshot")
		}
		copy := *plant
		copy.File = modelFilename(plant.binding.source)
		copy.SHA256 = plant.binding.source
		site.Objects[i].Plant = &copy
	}
	replay.Site = &site
	raw, err := json.MarshalIndent(replay, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
