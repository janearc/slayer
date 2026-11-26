package manufacture

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/plants/seasonal"
)

var plantProfilesRaw = seasonal.Embedded()

//go:embed plant-choices.yaml
var plantChoicesRaw []byte

var plantChoices = readPlantChoices(plantChoicesRaw)
var defaultPlantProfiles = mustPlantProfiles(plantProfilesRaw)

const maximumPlantFigure = seasonal.MaximumFigure

type plantProfile = seasonal.Profile

type PlantProfiles struct{ book *seasonal.Book }

// ReadPlantProfiles reads and checks a seasonal plant species book.
func ReadPlantProfiles(file string, raw []byte) (*PlantProfiles, error) {
	policy := &seasonal.Policy{Value: plantValue, Sow: func(n uint64) branching.Seed { return sow(n) }}
	book, err := seasonal.Read(file, raw, policy)
	if err != nil {
		return nil, err
	}
	return &PlantProfiles{book: book}, nil
}

// NewPlant makes a species from this book, without consulting the registry.
func (book *PlantProfiles) NewPlant(name string) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("manufacture: no plant profile book")
	}
	profile, found := book.book.Get(name)
	if !found {
		return Kind{}, fmt.Errorf("manufacture: no plant profile %q", name)
	}
	return Kind{name: name, how: seasonalPlant{profile: profile, season: "summer"}}, nil
}

// Named species constructors use profile data from the caller's book.
func (b *PlantProfiles) NewValleyOak() (Kind, error)       { return b.NewPlant("valley-oak") }
func (b *PlantProfiles) NewAlder() (Kind, error)           { return b.NewPlant("alder") }
func (b *PlantProfiles) NewPoplar() (Kind, error)          { return b.NewPlant("poplar") }
func (b *PlantProfiles) NewManzanita() (Kind, error)       { return b.NewPlant("manzanita") }
func (b *PlantProfiles) NewChaparralBrush() (Kind, error)  { return b.NewPlant("chaparral-brush") }
func (b *PlantProfiles) NewJoshuaTree() (Kind, error)      { return b.NewPlant("joshua-tree") }
func (b *PlantProfiles) NewKandeliaObovata() (Kind, error) { return b.NewPlant("kandelia-obovata") }
func (b *PlantProfiles) NewAegicerasCorniculatum() (Kind, error) {
	return b.NewPlant("aegiceras-corniculatum")
}
func (b *PlantProfiles) NewAvicenniaMarina() (Kind, error) { return b.NewPlant("avicennia-marina") }
func (b *PlantProfiles) NewBruguieraGymnorhiza() (Kind, error) {
	return b.NewPlant("bruguiera-gymnorhiza")
}
func (b *PlantProfiles) NewSonneratiaApetala() (Kind, error) { return b.NewPlant("sonneratia-apetala") }
func (b *PlantProfiles) NewPandanusTectorius() (Kind, error) { return b.NewPlant("pandanus-tectorius") }
func (b *PlantProfiles) NewScaevolaTaccada() (Kind, error)   { return b.NewPlant("scaevola-taccada") }
func (b *PlantProfiles) NewMetrosiderosPolymorpha() (Kind, error) {
	return b.NewPlant("metrosideros-polymorpha")
}
func (b *PlantProfiles) NewBarringtoniaAsiatica() (Kind, error) {
	return b.NewPlant("barringtonia-asiatica")
}
func (b *PlantProfiles) NewQuercusIlex() (Kind, error)  { return b.NewPlant("quercus-ilex") }
func (b *PlantProfiles) NewEricaArborea() (Kind, error) { return b.NewPlant("erica-arborea") }
func (b *PlantProfiles) NewSalixAlba() (Kind, error)    { return b.NewPlant("salix-alba") }

// mustPlantProfiles retains the checked default book for existing callers.
func mustPlantProfiles(raw []byte) *PlantProfiles {
	book, err := ReadPlantProfiles("plant-profiles.yaml", raw)
	if err != nil {
		panic(err)
	}
	return book
}

// readPlantChoices keeps shared numerical policies in Puzzlemouth data.
func readPlantChoices(raw []byte) numbers.Numbers {
	book, err := numbers.Read("plant-choices.yaml", raw)
	if err != nil {
		panic(err)
	}
	return book
}

// plantValue reads a picked construction or detail policy.
func plantValue(name string) float64 {
	value, err := plantChoices.Get(name)
	if err != nil {
		panic(err)
	}
	noted(value)
	return value.Value
}
