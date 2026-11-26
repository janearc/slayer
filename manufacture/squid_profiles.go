package manufacture

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/animals/molluscs/squid"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed animals/molluscs/squid/squid-profiles.yaml
var squidProfilesRaw []byte

// SquidProfiles is an immutable, checked book of illustrative squid profiles.
type SquidProfiles struct{ book *squid.Book }

// ReadSquidProfiles checks and owns the data in one bounded species book.
func ReadSquidProfiles(file string, raw []byte) (*SquidProfiles, error) {
	book, err := squid.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &SquidProfiles{book: book}, nil
}

// NewSquid builds the named profile with an illustrative phase in radians.
func (book *SquidProfiles) NewSquid(species string, phase float64) ([]thing.Thing, error) {
	if book == nil {
		return nil, fmt.Errorf("squid: no profile book")
	}
	profile, found := book.book.Get(species)
	if !found {
		return nil, fmt.Errorf("squid: no species %q", species)
	}
	return profile.Build(phase)
}

// NewSquidKind makes a reusable scene recipe at an authored phase in radians.
// Phase is bounded as a numerical input; it is not a time or a gait.
func (book *SquidProfiles) NewSquidKind(species string, phase float64) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("squid: no profile book")
	}
	profile, found := book.book.Get(species)
	if !found {
		return Kind{}, fmt.Errorf("squid: no species %q", species)
	}
	if math.IsNaN(phase) || math.IsInf(phase, 0) || math.Abs(phase) > 100000 {
		return Kind{}, fmt.Errorf("squid: invalid phase")
	}
	return Kind{name: species, how: squidKind{profile: profile, phase: phase}}, nil
}

type squidKind struct {
	profile squid.Recipe
	phase   float64
}

func (row squidKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	parts, err := row.profile.Build(row.phase)
	if err != nil {
		return good{}, err
	}
	return good{native: parts}, nil
}

// NewHumboldtSquid selects the Humboldt profile from this book.
func (book *SquidProfiles) NewHumboldtSquid(phase float64) ([]thing.Thing, error) {
	return book.NewSquid("humboldt-squid", phase)
}

var defaultSquidProfiles = func() *SquidProfiles {
	book, err := ReadSquidProfiles("squid-profiles.yaml", squidProfilesRaw)
	if err != nil {
		panic(err)
	}
	return book
}()
