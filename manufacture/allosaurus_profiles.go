package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/theropods"
	"github.com/janearc/slayer/maths"
)

var allosaurusProfilesRaw = theropods.Embedded()

// TheropodProfiles wraps a checked family book for manufacture recipes.
type TheropodProfiles struct{ book *theropods.TheropodProfiles }

// ReadTheropodProfiles reads and checks a theropod species book.
func ReadTheropodProfiles(file string, raw []byte) (*TheropodProfiles, error) {
	book, err := theropods.ReadTheropodProfiles(file, raw)
	if err != nil {
		return nil, err
	}
	return &TheropodProfiles{book: book}, nil
}

func (book *TheropodProfiles) NewTheropod(species string, seconds float64) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("no theropod book")
	}
	recipe, err := book.book.NewTheropod(species, seconds)
	if err != nil {
		return Kind{}, err
	}
	return Kind{name: species, how: theropodKind{recipe: recipe}}, nil
}

func (book *TheropodProfiles) NewAllosaurus(seconds float64) (Kind, error) {
	return book.NewTheropod("allosaurus", seconds)
}

type theropodKind struct{ recipe *theropods.Recipe }

func (row theropodKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if row.recipe == nil {
		return good{}, errKind
	}
	parts, box, err := row.recipe.Build(maths.Local())
	if err != nil {
		return good{}, err
	}
	return good{native: parts, extent: &box}, nil
}
