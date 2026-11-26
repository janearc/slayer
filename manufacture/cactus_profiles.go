package manufacture

import (
	"sync"

	"github.com/janearc/slayer/manufacture/plants/cacti"
)

var cactusProfilesRaw = cacti.Embedded()

// CactusProfiles wraps a checked species book and its private spine cache.
type CactusProfiles struct {
	book   *cacti.Book
	spines sync.Map
}

func cactusPolicy() cacti.Policy {
	lookup := func(name string) float64 {
		switch name {
		case "maximum-cactus-metres", "maximum-cactus-count", "maximum-stems", "maximum-pleat-depth":
			return plantValue(name)
		default:
			return cactusValue(name)
		}
	}
	return cacti.Policy{Lookup: lookup, Details: Complexities(), MinimumSides: leastAround, MaximumSides: mostAround}
}

// ReadCactusProfiles checks and snapshots a cactus species book.
func ReadCactusProfiles(file string, raw []byte) (*CactusProfiles, error) {
	book, err := cacti.Read(file, raw, cactusPolicy())
	if err != nil {
		return nil, err
	}
	return &CactusProfiles{book: book}, nil
}

func (book *CactusProfiles) NewCactus(name string) (Kind, error) {
	if book == nil {
		return Kind{}, errKind
	}
	recipe, err := book.book.New(name)
	if err != nil {
		return Kind{}, err
	}
	return Kind{name: name, how: columnarCactus{name: name, recipe: recipe, spinesCache: &book.spines}}, nil
}
func (book *CactusProfiles) NewSaguaro() (Kind, error) { return book.NewCactus("saguaro") }
