package manufacture

import (
	"fmt"
)

// PlantOptions identifies a species and optional seasonal or shoot form.
// Pointers distinguish an omitted option from an explicitly empty value.
type PlantOptions struct {
	Species        string
	Season         *string
	Representation *string
}

// PlantBook wraps one checked family book without registering its species.
type PlantBook struct {
	family   string
	seasonal *PlantProfiles
	conifer  *ConiferProfiles
	fern     *FernProfiles
	cactus   *CactusProfiles
}

// ReadPlantBook checks and snapshots a family-specific authored book.
func ReadPlantBook(family, file string, raw []byte) (*PlantBook, error) {
	book := &PlantBook{family: family}
	var err error
	switch family {
	case "seasonal":
		book.seasonal, err = ReadPlantProfiles(file, raw)
	case "conifer":
		book.conifer, err = ReadConiferProfiles(file, raw)
	case "fern":
		book.fern, err = ReadFernProfiles(file, raw)
	case "cactus":
		book.cactus, err = ReadCactusProfiles(file, raw)
	default:
		return nil, fmt.Errorf("manufacture: unknown plant family %q", family)
	}
	if err != nil {
		return nil, err
	}
	return book, nil
}

// HasSpecies reports whether this checked book contains the named species.
func (book *PlantBook) HasSpecies(name string) bool {
	if book == nil {
		return false
	}
	switch book.family {
	case "seasonal":
		_, ok := book.seasonal.book.Get(name)
		return ok
	case "conifer":
		return book.conifer.book.Has(name)
	case "fern":
		_, ok := book.fern.profiles.Get(name)
		return ok
	case "cactus":
		_, err := book.cactus.book.New(name)
		return err == nil
	}
	return false
}

// Species lists the names in this checked book in a stable order.
func (book *PlantBook) Species() []string {
	if book == nil {
		return nil
	}
	switch book.family {
	case "seasonal":
		return book.seasonal.book.Species()
	case "conifer":
		return book.conifer.book.Species()
	case "fern":
		return book.fern.profiles.Species()
	case "cactus":
		return book.cactus.book.Species()
	}
	return nil
}

// New applies the family's species and representation rules. An omitted
// representation selects the whole plant; "shoot" selects a loose conifer
// shoot. Seasons apply only to seasonal plants.
func (book *PlantBook) New(options PlantOptions) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("manufacture: no plant book")
	}
	if options.Season != nil && book.family != "seasonal" {
		return Kind{}, fmt.Errorf("season is only supported for seasonal plants")
	}
	if options.Representation != nil {
		if book.family != "conifer" || *options.Representation != "shoot" {
			return Kind{}, fmt.Errorf("unsupported plant representation %q for family %q", *options.Representation, book.family)
		}
		return book.conifer.NewConiferShoot(options.Species)
	}
	var row Kind
	var err error
	switch book.family {
	case "seasonal":
		row, err = book.seasonal.NewPlant(options.Species)
	case "conifer":
		row, err = book.conifer.NewConifer(options.Species)
	case "fern":
		row, err = book.fern.NewFern(options.Species)
	case "cactus":
		row, err = book.cactus.NewCactus(options.Species)
	default:
		return Kind{}, fmt.Errorf("manufacture: unknown plant family %q", book.family)
	}
	if err != nil {
		return Kind{}, err
	}
	if options.Season != nil {
		return WithSeason(row, *options.Season)
	}
	return row, nil
}

// EmbeddedPlantBook returns an independent copy of a family's default book.
func EmbeddedPlantBook(family string) (name string, raw []byte, err error) {
	switch family {
	case "seasonal":
		return "plant-profiles.yaml", append([]byte(nil), plantProfilesRaw...), nil
	case "conifer":
		return "conifer-profiles.yaml", append([]byte(nil), defaultConiferProfilesRaw...), nil
	case "fern":
		return "fern-profiles.yaml", append([]byte(nil), fernProfilesRaw...), nil
	case "cactus":
		return "cactus-profiles.yaml", append([]byte(nil), cactusProfilesRaw...), nil
	default:
		return "", nil, fmt.Errorf("manufacture: unknown plant family %q", family)
	}
}
