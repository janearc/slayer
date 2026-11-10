package main

import (
	"fmt"

	"github.com/janearc/slayer/manufacture"
)

// construction runs only on a stock miss. Static recipes retain their meshes.
func (stage *staging) construction(kind, recipe string,
) (*manufacture.Kind, error) {
	if recipe == "" {
		row, err := manufacture.KindNamed(kind)
		if err != nil {
			return nil, err
		}
		return &row, nil
	}
	row, found := stage.recipes.Recipe(recipe)
	if !found {
		return nil, fmt.Errorf("unknown construction recipe %q", recipe)
	}
	return row.Construction()
}

// constructed returns no stock on a construction refusal.
func (stage *staging) constructed(kind, recipe, detail string,
	seed uint64,
) (*kept, error) {
	row, err := stage.construction(kind, recipe)
	if err != nil {
		return nil, err
	}
	one, err := manufacturedKind(stage.given, *row, int(seed), detail)
	if err != nil {
		return nil, err
	}
	return &one, nil
}
