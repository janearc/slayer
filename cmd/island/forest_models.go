package main

import (
	"fmt"
	"sync"

	"github.com/janearc/slayer/manufacture"
)

type forestBook struct {
	family string
	book   *manufacture.PlantBook
}

// forestBooks fixes the embedded model snapshot for the lifetime of the scene.
// Plant species use these books; other mixture kinds retain registry lookup.
var forestBooks = sync.OnceValues(func() ([]forestBook, error) {
	var books []forestBook
	for _, family := range []string{"seasonal", "conifer", "fern", "cactus"} {
		name, raw, err := manufacture.EmbeddedPlantBook(family)
		if err != nil {
			return nil, err
		}
		book, err := manufacture.ReadPlantBook(family, name, raw)
		if err != nil {
			return nil, err
		}
		books = append(books, forestBook{family, book})
	}
	return books, nil
})

func forestPlant(name string) (manufacture.Kind, error) {
	books, err := forestBooks()
	if err != nil {
		return manufacture.Kind{}, err
	}
	return forestPlantFrom(books, name)
}

// forestPlantFrom rejects duplicate species and old shoot-selector collisions
// before constructing anything. A recognized shoot with a missing species is
// refused by its conifer book rather than falling back to the kind registry.
func forestPlantFrom(books []forestBook, name string) (manufacture.Kind, error) {
	var selected *manufacture.PlantBook
	var options manufacture.PlantOptions
	selectBook := func(book *manufacture.PlantBook, option manufacture.PlantOptions) error {
		if selected != nil {
			return fmt.Errorf("island: ambiguous forest plant %q", name)
		}
		selected, options = book, option
		return nil
	}
	shootSpecies := map[string]string{
		"coast-redwood-shoot": "coast-redwood",
		"giant-sequoia-shoot": "giant-sequoia",
	}
	for _, family := range books {
		if family.book.HasSpecies(name) {
			if err := selectBook(family.book, manufacture.PlantOptions{Species: name}); err != nil {
				return manufacture.Kind{}, err
			}
		}
		if species, found := shootSpecies[name]; found && family.family == "conifer" {
			shoot := "shoot"
			if err := selectBook(family.book, manufacture.PlantOptions{Species: species, Representation: &shoot}); err != nil {
				return manufacture.Kind{}, err
			}
		}
	}
	if selected != nil {
		return selected.New(options)
	}
	return manufacture.KindNamed(name)
}
