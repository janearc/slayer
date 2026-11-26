package manufacture

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/whales"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

var whaleProfilesRaw = whales.Embedded()

type WhaleProfiles struct{ book *whales.Book }

// ReadWhaleProfiles reads and checks a whale species book.
func ReadWhaleProfiles(file string, raw []byte) (*WhaleProfiles, error) {
	book, err := whales.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &WhaleProfiles{book: book}, nil
}

// NewWhale selects a species and either its animated swim, still spy-hop or
// rest skeleton. The accepted anatomy is that of the selected profile.
func (book *WhaleProfiles) NewWhale(species, pose, representation string) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("whale: no profile book")
	}
	p, found := book.book.Get(species)
	if !found {
		return Kind{}, fmt.Errorf("whale: no species %q", species)
	}
	if representation == "bones" && pose == "" {
		return Kind{name: species, how: whaleStill{profile: p, bones: true}}, nil
	}
	if representation != "" && representation != "skin" {
		return Kind{}, fmt.Errorf("whale: unsupported representation %q", representation)
	}
	switch pose {
	case "", "swim":
		return Kind{name: species, how: whaleSwim{profile: p}}, nil
	case "spy-hop":
		return Kind{name: species, how: whaleStill{profile: p}}, nil
	default:
		return Kind{}, fmt.Errorf("whale: unsupported pose %q", pose)
	}
}

// NewOrca is the named convenience constructor for an orca profile.
func (book *WhaleProfiles) NewOrca(pose, representation string) (Kind, error) {
	return book.NewWhale("orca", pose, representation)
}

type whaleSwim struct{ profile whales.Profile }

func (row whaleSwim) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	parts, err := row.profile.Swim(0, maths.Local())
	return orcaMade(parts, err)
}

func (row whaleSwim) madeAtTime(from seed, detail string, at time.Duration) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if detail != "quick" {
		return good{}, fmt.Errorf("whale: unsupported complexity %q", detail)
	}
	parts, err := row.profile.Swim(2*math.Pi*at.Seconds()/row.profile.Cycle, maths.Local())
	return orcaMade(parts, err)
}

type whaleStill struct {
	profile whales.Profile
	bones   bool
}

func (row whaleStill) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if row.bones {
		return orcaMade(row.profile.Bones(maths.Local()))
	}
	parts, err := row.profile.SpyHop(maths.Local())
	return orcaMade(parts, err)
}

var defaultWhaleProfiles = &WhaleProfiles{book: whales.Default()}

func orcaSwimming(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	p, _ := whales.Default().Get("orca")
	return p.Swim(phase, at)
}

func orcaMade(parts []thing.Thing, err error) (good, error) {
	if err != nil {
		return good{}, err
	}
	box := parts[0].Shape.Bounds()
	return good{native: parts, extent: &box}, nil
}
