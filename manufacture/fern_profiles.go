package manufacture

import (
	_ "embed"
	"fmt"

	"github.com/janearc/slayer/manufacture/plants/ferns"
)

//go:embed plants/ferns/fern-profiles.yaml
var fernProfilesRaw []byte

// FernProfiles is a checked, immutable book of authored fern profiles.
type FernProfiles struct{ profiles *ferns.Book }

// ReadFernProfiles reads species data without modifying the global registry.
// The book snapshots its input; invalid and excessive construction is refused.
func ReadFernProfiles(file string, raw []byte) (*FernProfiles, error) {
	profiles, err := ferns.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &FernProfiles{profiles: profiles}, nil
}

// NewFern returns a checked maker from one named profile in the book.
func (book *FernProfiles) NewFern(name string) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("manufacture: no fern book")
	}
	profile, ok := book.profiles.Get(name)
	if !ok {
		return Kind{}, fmt.Errorf("manufacture: no fern profile %q", name)
	}
	green := plain("frond", profile.Colour.R, profile.Colour.G, profile.Colour.B)
	row := fern{long: span{profile.FrondLength.Minimum, profile.FrondLength.Maximum},
		stalk:    span{profile.StalkRadius.Minimum, profile.StalkRadius.Maximum},
		fronds:   span{float64(profile.Fronds.Minimum), float64(profile.Fronds.Maximum)},
		leaflets: span{float64(profile.Leaflets.Minimum), float64(profile.Leaflets.Maximum)},
		dressed:  looks{body: green, leaf: green},
		anatomy:  profile.Anatomy}
	return Kind{name: name, how: row}, nil
}

// NewGenericFern selects the existing authored fern, not a named species.
func (book *FernProfiles) NewGenericFern() (Kind, error) { return book.NewFern("fern") }
