package main

import (
	"github.com/janearc/slayer/manufacture"
)

// plantSpec identifies a checked species and optional shoot or season.
type plantSpec struct {
	Family         string  `json:"family" yaml:"family"`
	File           string  `json:"file" yaml:"file"`
	Species        string  `json:"species" yaml:"species"`
	Season         *string `json:"season,omitempty" yaml:"season"`
	Representation *string `json:"representation,omitempty" yaml:"representation"`
}

func (entry *entry) bindPlant(books *modelBooks) error {
	if entry.Plant == nil {
		return nil
	}
	spec := *entry.Plant
	raw, err := books.read("plant", spec.Family, spec.File)
	if err != nil {
		return err
	}
	book, err := cachedModelBook(&books.plants, spec.Family, spec.File, raw, manufacture.ReadPlantBook)
	if err != nil {
		return err
	}
	row, err := book.New(manufacture.PlantOptions{Species: spec.Species, Season: spec.Season, Representation: spec.Representation})
	if err != nil {
		return err
	}
	return entry.bindSource("plant", raw, row)
}
