package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/manufacture/plants/conifers"
)

var defaultConiferProfilesRaw = conifers.Embedded()
var defaultConiferProfiles = mustConiferProfiles(defaultConiferProfilesRaw)

func mustConiferProfiles(raw []byte) *ConiferProfiles {
	book, err := ReadConiferProfiles("conifer-profiles.yaml", raw)
	if err != nil {
		panic(err)
	}
	return book
}

// ConiferProfiles wraps one checked family book and its shoot cache.
type ConiferProfiles struct {
	book   *conifers.Book
	shoots *shootStore
}

// ReadConiferProfiles checks a family book without registering its species.
func ReadConiferProfiles(file string, raw []byte) (*ConiferProfiles, error) {
	book, err := conifers.Read(file, raw, shootValue, maximumPlantFigure)
	if err != nil {
		return nil, err
	}
	return &ConiferProfiles{book: book, shoots: &shootStore{}}, nil
}

func (book *ConiferProfiles) selectTree(name string) (conifer, conifers.Profile, error) {
	if book == nil {
		return conifer{}, conifers.Profile{}, fmt.Errorf("manufacture: no conifer profile book")
	}
	profile, tree, err := book.book.Select(name)
	if err != nil {
		return conifer{}, conifers.Profile{}, err
	}
	measured, picked := profile.Notes()
	row := conifer{Tree: tree,
		dressed: looks{body: plain(profile.BarkName, profile.BarkColour[0], profile.BarkColour[1], profile.BarkColour[2]),
			leaf: plain(profile.LeafName, profile.LeafColour[0], profile.LeafColour[1], profile.LeafColour[2])},
		shootConfig: &shootConfig{profile: profile.Needle.Geometry(), store: book.shoots, measured: measured, picked: picked}}
	return row, profile, nil
}

// NewConifer constructs one species from the checked book.
func (book *ConiferProfiles) NewConifer(name string) (Kind, error) {
	row, profile, err := book.selectTree(name)
	if err != nil {
		return Kind{}, err
	}
	if profile.Foliage == "needles" {
		return Kind{name: name, how: physicalSpruce{legacy: row}}, nil
	}
	return Kind{name: name, how: row}, nil
}

func (book *ConiferProfiles) NewSpruce() (Kind, error)       { return book.NewConifer("spruce") }
func (book *ConiferProfiles) NewNorwaySpruce() (Kind, error) { return book.NewConifer("norway") }
func (book *ConiferProfiles) NewCoastRedwood() (Kind, error) { return book.NewConifer("coast-redwood") }
func (book *ConiferProfiles) NewGiantSequoia() (Kind, error) { return book.NewConifer("giant-sequoia") }

func noteConiferFoliage(config *shootConfig) {
	if config == nil {
		return
	}
	for i := range config.measured {
		noted(&config.measured[i])
	}
	for i := range config.picked {
		noted(&config.picked[i])
	}
}
