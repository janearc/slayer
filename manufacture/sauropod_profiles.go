package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/sauropods"
	"github.com/janearc/slayer/maths"
)

// SauropodProfiles owns a checked book of static sauropod profiles.
type SauropodProfiles struct{ book *sauropods.Book }

// ReadSauropodProfiles checks and snapshots an authored sauropod book.
func ReadSauropodProfiles(file string, raw []byte) (*SauropodProfiles, error) {
	book, err := sauropods.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &SauropodProfiles{book: book}, nil
}

// NewSauropod selects one species from this book as a static kind.
func (book *SauropodProfiles) NewSauropod(name string) (Kind, error) {
	if book == nil || book.book == nil {
		return Kind{}, fmt.Errorf("manufacture: no sauropod profile book")
	}
	profile, err := book.book.Profile(name)
	if err != nil {
		return Kind{}, err
	}
	return Kind{name: name, how: sauropodKind{profile: profile}}, nil
}

// NewAlamosaurus selects the Alamosaurus sanjuanensis profile.
func (book *SauropodProfiles) NewAlamosaurus() (Kind, error) {
	return book.NewSauropod("alamosaurus-sanjuanensis")
}

// NewDiplodocus selects the Diplodocus carnegii profile.
func (book *SauropodProfiles) NewDiplodocus() (Kind, error) {
	return book.NewSauropod("diplodocus-carnegii")
}

// NewApatosaurus selects the Apatosaurus louisae profile.
func (book *SauropodProfiles) NewApatosaurus() (Kind, error) {
	return book.NewSauropod("apatosaurus-louisae")
}

// NewBrachiosaurus selects the Brachiosaurus altithorax profile.
func (book *SauropodProfiles) NewBrachiosaurus() (Kind, error) {
	return book.NewSauropod("brachiosaurus-altithorax")
}

type sauropodKind struct{ profile sauropods.Profile }

func (row sauropodKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	result, err := sauropods.Build(row.profile)
	if err != nil {
		return good{}, err
	}
	return good{native: result.Parts, extent: &result.Bounds}, nil
}

func (row sauropodKind) extent(from seed) (maths.Box, error) {
	if !from.made() {
		return maths.Box{}, errGood
	}
	return sauropods.Extent(row.profile)
}
