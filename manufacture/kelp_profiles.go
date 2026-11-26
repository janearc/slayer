package manufacture

import (
	"fmt"

	kelp "github.com/janearc/slayer/manufacture/algae/kelp"
	"github.com/janearc/slayer/manufacture/branching"
)

// KelpOptions selects one checked alga, water condition and optional forest.
type KelpOptions = kelp.Options

// KelpProfiles owns a checked kelp profile and its shared geometry.
type KelpProfiles struct{ book *kelp.Profiles }

var kelpRaw = kelp.Embedded()
var defaultKelpProfiles, kelpError = ReadKelpProfiles("kelp.yaml", kelpRaw)

// EmbeddedAlgaBook returns an independent copy of the default species book.
func EmbeddedAlgaBook(family string) (string, []byte, error) {
	if family != "kelp" {
		return "", nil, fmt.Errorf("manufacture: unknown alga family %q", family)
	}
	return "kelp.yaml", kelp.Embedded(), nil
}

// ReadKelpProfiles checks and freezes a book before any recipe uses it.
func ReadKelpProfiles(file string, raw []byte) (*KelpProfiles, error) {
	book, err := kelp.Read(file, raw, func(number uint64) branching.Seed { return sow(number) })
	if err != nil {
		return nil, err
	}
	return &KelpProfiles{book: book}, nil
}

// NewKelp selects a species and water without registering an alias.
func (profiles *KelpProfiles) NewKelp(options KelpOptions) (Kind, error) {
	if profiles == nil {
		return Kind{}, fmt.Errorf("manufacture: no kelp profiles")
	}
	specimen, err := profiles.book.New(options)
	if err != nil {
		return Kind{}, err
	}
	return Kind{name: "alga-profile", how: kelpRow{specimen: &specimen}}, nil
}

// NewGiantKelp selects the giant-kelp profile and its water and stand.
func (profiles *KelpProfiles) NewGiantKelp(condition string, forest bool) (Kind, error) {
	return profiles.NewKelp(KelpOptions{Species: "macrocystis-pyrifera", Condition: condition, Forest: forest})
}
