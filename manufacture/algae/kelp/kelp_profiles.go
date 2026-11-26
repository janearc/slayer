package kelp

import (
	"fmt"
	"sync"

	"github.com/janearc/slayer/manufacture/branching"
)

// KelpOptions selects one checked alga, water condition and optional forest.
type Options struct {
	Species   string
	Condition string
	Forest    bool
}

// KelpProfiles owns the figures and shared geometry of one kelp species.
type Profiles struct {
	book kelpBook
	kit  func() (*kelpKit, error)
}

// ReadKelpProfiles checks and freezes a book before any recipe uses it.
func Read(file string, raw []byte, factory SeedFactory) (*Profiles, error) {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return nil, fmt.Errorf("%s: empty or oversized kelp book", file)
	}
	if factory == nil {
		return nil, fmt.Errorf("%s: no kelp seed factory", file)
	}
	book, err := readKelp(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return makeKelpProfiles(book, factory), nil
}

func makeKelpProfiles(book kelpBook, factory SeedFactory) *Profiles {
	return &Profiles{book: book, kit: sync.OnceValues(func() (*kelpKit, error) {
		return newKelpKit(book, factory)
	})}
}

// SeedFactory supplies deterministic seeds for the fixed shared blade variants.
type SeedFactory func(uint64) branching.Seed

// Specimen is a checked kelp species, water condition and stand selection.
type Specimen struct {
	profile *Profiles
	water   string
	forest  bool
}

// New selects one checked species and water condition.
func (profiles *Profiles) New(options Options) (Specimen, error) {
	if profiles == nil {
		return Specimen{}, fmt.Errorf("manufacture: no kelp profiles")
	}
	if options.Species != profiles.book.Species {
		return Specimen{}, fmt.Errorf("manufacture: unknown kelp species %q", options.Species)
	}
	condition := options.Condition
	if condition == "" {
		condition = "moderate"
	}
	if _, ok := profiles.book.Conditions[condition]; !ok {
		return Specimen{}, fmt.Errorf("manufacture: unknown kelp water %q", condition)
	}
	return Specimen{profile: profiles, water: condition, forest: options.Forest}, nil
}
