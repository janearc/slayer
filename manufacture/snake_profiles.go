package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/snakes"
	"github.com/janearc/slayer/maths"
)

// SnakeProfiles owns a checked book of snake profiles.
type SnakeProfiles struct{ book *snakes.Book }

// ReadSnakeProfiles checks and snapshots an authored snake book.
func ReadSnakeProfiles(file string, raw []byte) (*SnakeProfiles, error) {
	book, err := snakes.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &SnakeProfiles{book: book}, nil
}

// NewSnake selects one species from this book as a static kind.
func (book *SnakeProfiles) NewSnake(name string) (Kind, error) {
	if book == nil || book.book == nil {
		return Kind{}, fmt.Errorf("manufacture: no snake profile book")
	}
	profile, err := book.book.Profile(name)
	if err != nil {
		return Kind{}, err
	}
	return Kind{name: name, how: snakeKind{profile: profile}}, nil
}

// NewPalaeophisColossaeus selects the Palaeophis colossaeus profile.
func (book *SnakeProfiles) NewPalaeophisColossaeus() (Kind, error) {
	return book.NewSnake("palaeophis-colossaeus")
}

// NewTitanoboaCerrejonensis selects the Titanoboa cerrejonensis profile.
func (book *SnakeProfiles) NewTitanoboaCerrejonensis() (Kind, error) {
	return book.NewSnake("titanoboa-cerrejonensis")
}

// NewEunectesMurinus selects the Eunectes murinus profile.
func (book *SnakeProfiles) NewEunectesMurinus() (Kind, error) {
	return book.NewSnake("eunectes-murinus")
}

// NewBoaConstrictorOccidentalis selects the Boa constrictor occidentalis profile.
func (book *SnakeProfiles) NewBoaConstrictorOccidentalis() (Kind, error) {
	return book.NewSnake("boa-constrictor-occidentalis")
}

// NewMalayopythonReticulatus selects the Malayopython reticulatus profile.
func (book *SnakeProfiles) NewMalayopythonReticulatus() (Kind, error) {
	return book.NewSnake("malayopython-reticulatus")
}

// NewPythonBivittatus selects the Python bivittatus profile.
func (book *SnakeProfiles) NewPythonBivittatus() (Kind, error) {
	return book.NewSnake("python-bivittatus")
}

// NewBitisGabonica selects the Bitis gabonica profile.
func (book *SnakeProfiles) NewBitisGabonica() (Kind, error) {
	return book.NewSnake("bitis-gabonica")
}

// NewPythonSebae selects the Python sebae profile.
func (book *SnakeProfiles) NewPythonSebae() (Kind, error) {
	return book.NewSnake("python-sebae")
}

// NewMadtsoiaBai selects the Madtsoia bai profile.
func (book *SnakeProfiles) NewMadtsoiaBai() (Kind, error) {
	return book.NewSnake("madtsoia-bai")
}

type snakeKind struct{ profile snakes.Profile }

func (row snakeKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	result, err := snakes.Build(row.profile)
	if err != nil {
		return good{}, err
	}
	return good{native: result.Parts, extent: &result.Bounds}, nil
}

// extent uses the checked profile's conservative bound without making skin.
func (row snakeKind) extent(from seed) (maths.Box, error) {
	if !from.made() {
		return maths.Box{}, errGood
	}
	return snakes.Extent(row.profile)
}
