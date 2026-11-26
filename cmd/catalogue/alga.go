package main

import (
	"fmt"

	"github.com/janearc/slayer/manufacture"
)

// algaSpec identifies a species, its water and whether it grows in a forest.
type algaSpec struct {
	Family    string `json:"family" yaml:"family"`
	File      string `json:"file" yaml:"file"`
	Species   string `json:"species" yaml:"species"`
	Condition string `json:"condition,omitempty" yaml:"condition"`
	Forest    bool   `json:"forest,omitempty" yaml:"forest"`
}

func (entry *entry) bindAlga(books *modelBooks) error {
	if entry.Alga == nil {
		return nil
	}
	spec := *entry.Alga
	if spec.Family != "kelp" {
		return fmt.Errorf("unknown alga family %q", spec.Family)
	}
	raw, err := books.read("alga", spec.Family, spec.File)
	if err != nil {
		return err
	}
	book, err := cachedModelBook(&books.algae, spec.Family, spec.File, raw,
		func(_ string, file string, raw []byte) (*manufacture.KelpProfiles, error) {
			return manufacture.ReadKelpProfiles(file, raw)
		})
	if err != nil {
		return err
	}
	row, err := book.NewKelp(manufacture.KelpOptions{
		Species: spec.Species, Condition: spec.Condition, Forest: spec.Forest,
	})
	if err != nil {
		return err
	}
	return entry.bindSource("alga", raw, row)
}
