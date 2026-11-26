package manufacture

import (
	"fmt"
	"sync"
	"testing"
)

// Plants used by geometry contracts are resolved from checked books. This is
// test-only: production KindNamed deliberately does not alias these names.
var testPlantBooks = sync.OnceValues(func() (map[string]*PlantBook, error) {
	books := make(map[string]*PlantBook)
	for _, family := range []string{"seasonal", "conifer", "fern", "cactus"} {
		file, raw, err := EmbeddedPlantBook(family)
		if err != nil {
			return nil, err
		}
		book, err := ReadPlantBook(family, file, raw)
		if err != nil {
			return nil, err
		}
		for _, name := range book.Species() {
			if _, exists := books[name]; exists {
				return nil, fmt.Errorf("duplicate test plant %q", name)
			}
			books[name] = book
		}
	}
	return books, nil
})

func testKind(name string) (kind, error) {
	switch name {
	case "orca-clean-05":
		return defaultWhaleProfiles.NewOrca("swim", "")
	case "orca-clean-05-spy-hop":
		return defaultWhaleProfiles.NewOrca("spy-hop", "")
	case "orca-clean-05-bones":
		return defaultWhaleProfiles.NewOrca("", "bones")
	}
	books, err := testPlantBooks()
	if err != nil {
		return kind{}, err
	}
	if name == "coast-redwood-shoot" || name == "giant-sequoia-shoot" {
		species := name[:len(name)-len("-shoot")]
		shoot := "shoot"
		return books[species].New(PlantOptions{Species: species, Representation: &shoot})
	}
	if book := books[name]; book != nil {
		return book.New(PlantOptions{Species: name})
	}
	return kindNamed(name)
}

// isTestPlant reports only names represented by the checked plant books.
func isTestPlant(name string) bool {
	if name == "coast-redwood-shoot" || name == "giant-sequoia-shoot" {
		return true
	}
	books, err := testPlantBooks()
	return err == nil && books[name] != nil
}

func TestPlantNamesAreNotFactoryKindAliases(t *testing.T) {
	books, err := testPlantBooks()
	if err != nil {
		t.Fatal(err)
	}
	for name := range books {
		if _, err := KindNamed(name); err == nil {
			t.Errorf("plant species %q is a factory alias", name)
		}
	}
	for _, name := range []string{"coast-redwood-shoot", "giant-sequoia-shoot"} {
		if _, err := KindNamed(name); err == nil {
			t.Errorf("plant representation %q is a factory alias", name)
		}
		if _, err := testKind(name); err != nil {
			t.Errorf("checked shoot %q: %v", name, err)
		}
	}
	if _, err := KindNamed("can"); err != nil {
		t.Fatalf("ordinary kind: %v", err)
	}
}
