package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/hatzegopteryx"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/pliosaurs"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// ReptileProfiles selects the separate authored pliosaur, crocodylian and
// pterosaur studies. The scaffolds are not general models of their clades.
type ReptileProfiles struct{ book *reptiles.Book }

// EmbeddedReptileBook returns an independent default profile document.
func EmbeddedReptileBook() (string, []byte) { return reptiles.Embedded() }

// ReadReptileProfiles checks and snapshots the whole authored book.
func ReadReptileProfiles(file string, raw []byte) (*ReptileProfiles, error) {
	book, err := reptiles.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &ReptileProfiles{book: book}, nil
}

// NewReptile selects a species and a distinct authored study. An empty study
// selects the original composite for pliosaurs and pterosaurs, and the sole
// Purussaurus study. Pose controls come from the checked profile.
func (book *ReptileProfiles) NewReptile(species, study string) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("reptile: no profile book")
	}
	p, found := book.book.Get(species)
	if !found {
		return Kind{}, fmt.Errorf("reptile: no species %q", species)
	}
	switch p.Scaffold {
	case "pliosaur":
		if study == "" {
			study = "baseline"
		}
		if study != "baseline" && study != "continuous" {
			return Kind{}, fmt.Errorf("reptile: unsupported pliosaur study %q", study)
		}
	case "crocodylian":
		if study == "" {
			study = "study"
		}
		if study != "study" {
			return Kind{}, fmt.Errorf("reptile: unsupported crocodylian study %q", study)
		}
	case "pterosaur":
		if study == "" {
			study = "composite"
		}
		if study != "composite" && study != "cage" && study != "density" {
			return Kind{}, fmt.Errorf("reptile: unsupported pterosaur study %q", study)
		}
	default:
		return Kind{}, fmt.Errorf("reptile: unsupported scaffold %q", p.Scaffold)
	}
	return Kind{name: species, how: reptileStudy{profile: p, study: study}}, nil
}

func (book *ReptileProfiles) NewPliosaur(study string) (Kind, error) {
	return book.NewReptile("pliosaur", study)
}
func (book *ReptileProfiles) NewPurussaurus() (Kind, error) {
	return book.NewReptile("purussaurus", "")
}
func (book *ReptileProfiles) NewHatzegopteryx(study string) (Kind, error) {
	return book.NewReptile("hatzegopteryx", study)
}

type reptileStudy struct {
	profile reptiles.Profile
	study   string
}

func (row reptileStudy) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	var parts []thing.Thing
	var err error
	p := row.profile
	switch p.Scaffold {
	case "pliosaur":
		if row.study == "continuous" {
			parts, err = pliosaurContinuousSkin(p, p.Phase)
		} else {
			parts, err = pliosaurs.NewComposite(p, p.Phase, maths.Local())
		}
	case "crocodylian":
		parts, err = purussaurusStudy(p, p.Phase, p.Yaw, p.Posture)
	case "pterosaur":
		switch row.study {
		case "composite":
			parts, err = hatzegopteryx.NewComposite(p, p.Fold, p.Seconds, maths.Local())
		case "cage":
			parts, err = hatzegopteryx.NewCage(p, p.Fold, p.Seconds, maths.Local())
		case "density":
			parts, err = hatzDensity(p, p.Phase, true, maths.Local())
		default:
			return good{}, fmt.Errorf("reptile: unsupported pterosaur study %q", row.study)
		}
	default:
		return good{}, fmt.Errorf("reptile: unsupported scaffold %q", p.Scaffold)
	}
	if err != nil {
		return good{}, err
	}
	return good{native: parts}, nil
}

var defaultReptileProfiles = func() *ReptileProfiles {
	file, raw := EmbeddedReptileBook()
	book, err := ReadReptileProfiles(file, raw)
	if err != nil {
		panic(err)
	}
	return book
}()

// PliosaurStudy supplies the default composite to the gargoyle constructor.
func PliosaurStudy(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	profile, _ := defaultReptileProfiles.book.Get("pliosaur")
	return pliosaurs.NewComposite(profile, phase, at)
}
