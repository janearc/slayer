package manufacture

import (
	"fmt"
	birds "github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/theropods/birds"
)

var birdRaw = birds.Embedded()

// BirdProfiles is an immutable, checked book of bird species.
type BirdProfiles struct{ book *birds.Book }

// ReadBirdProfiles reads a bird book without changing other books.
func ReadBirdProfiles(file string, raw []byte) (*BirdProfiles, error) {
	book, err := birds.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &BirdProfiles{book: book}, nil
}

// NewBird selects a species and stance from this checked book.
func (book *BirdProfiles) NewBird(species, stance string) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("no bird book")
	}
	bird, err := book.book.New(species, stance)
	if err != nil {
		return Kind{}, err
	}
	return Kind{name: "bird-" + species + "-" + stance, how: birdKind{species: species, profile: &bird}}, nil
}

func (book *BirdProfiles) NewBlackFacedSpoonbill(stance string) (Kind, error) {
	return book.NewBird("black-faced-spoonbill", stance)
}
func (book *BirdProfiles) NewLittleEgret(stance string) (Kind, error) {
	return book.NewBird("little-egret", stance)
}
func (book *BirdProfiles) NewGreyHeron(stance string) (Kind, error) {
	return book.NewBird("grey-heron", stance)
}
func (book *BirdProfiles) NewPacificReefHeron(stance string) (Kind, error) {
	return book.NewBird("pacific-reef-heron", stance)
}
func (book *BirdProfiles) NewYellowLeggedGull(stance string) (Kind, error) {
	return book.NewBird("yellow-legged-gull", stance)
}
func (book *BirdProfiles) NewWhiteTern(stance string) (Kind, error) {
	return book.NewBird("white-tern", stance)
}
func (book *BirdProfiles) NewGreatFrigatebird(stance string) (Kind, error) {
	return book.NewBird("great-frigatebird", stance)
}
func (book *BirdProfiles) NewWhiteTailedTropicbird(stance string) (Kind, error) {
	return book.NewBird("white-tailed-tropicbird", stance)
}
