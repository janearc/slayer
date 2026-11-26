package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/fishes"
)

// FishOptions chooses a single fish, a glide, its bones, or a school.
type FishOptions = fishes.FishOptions

// FishProfiles wraps a checked family book for manufacture recipes.
type FishProfiles struct{ book *fishes.FishProfiles }

// ReadFishProfiles reads and checks a fish species book.
func ReadFishProfiles(file string, raw []byte) (*FishProfiles, error) {
	book, err := fishes.ReadFishProfiles(file, raw)
	if err != nil {
		return nil, err
	}
	return &FishProfiles{book: book}, nil
}

func (book *FishProfiles) NewFish(name string, options FishOptions) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("manufacture: no fish profile book")
	}
	recipe, err := book.book.NewFish(name, options)
	if err != nil {
		return Kind{}, err
	}
	return Kind{name: name, how: fishRow{recipe}}, nil
}

func (b *FishProfiles) NewPacificBluefinTuna(o FishOptions) (Kind, error) {
	return b.NewFish("pacific-bluefin-tuna", o)
}
func (b *FishProfiles) NewCaliforniaYellowtail(o FishOptions) (Kind, error) {
	return b.NewFish("california-yellowtail", o)
}
func (b *FishProfiles) NewCaliforniaBarracuda(o FishOptions) (Kind, error) {
	return b.NewFish("california-barracuda", o)
}
func (b *FishProfiles) NewPacificSardine(o FishOptions) (Kind, error) {
	return b.NewFish("pacific-sardine", o)
}
func (b *FishProfiles) NewConvictTang(o FishOptions) (Kind, error) {
	return b.NewFish("convict-tang", o)
}
func (b *FishProfiles) NewDuskyGrouper(o FishOptions) (Kind, error) {
	return b.NewFish("dusky-grouper", o)
}
func (b *FishProfiles) NewMediterraneanChromis(o FishOptions) (Kind, error) {
	return b.NewFish("mediterranean-chromis", o)
}
func (b *FishProfiles) NewEuropeanPerch(o FishOptions) (Kind, error) {
	return b.NewFish("european-perch", o)
}
func (b *FishProfiles) NewFlatheadGreyMullet(o FishOptions) (Kind, error) {
	return b.NewFish("flathead-grey-mullet", o)
}
