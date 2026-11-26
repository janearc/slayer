package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/pinnipeds"
)

var pinnipedProfilesRaw = pinnipeds.Embedded()

type pinnipedProfile = pinnipeds.Profile
type PinnipedProfiles struct{ book *pinnipeds.Book }

// ReadPinnipedProfiles reads and checks a pinniped species book.
func ReadPinnipedProfiles(file string, raw []byte) (*PinnipedProfiles, error) {
	b, err := pinnipeds.Read(file, raw)
	if err != nil {
		return nil, err
	}
	return &PinnipedProfiles{book: b}, nil
}
func (book *PinnipedProfiles) profile(name string) (pinnipedProfile, error) {
	if book == nil {
		return pinnipedProfile{}, fmt.Errorf("manufacture: no pinniped book")
	}
	p, ok := book.book.Get(name)
	if !ok {
		return pinnipedProfile{}, fmt.Errorf("manufacture: no pinniped profile %q", name)
	}
	return p, nil
}

// NewPinniped constructs one of the supported species in the named pose.
// Harbor-loft forms support hauled and swimming; monk-loft forms support
// hauled; walrus forms support lying, hauled-out and swimming. Empty chooses
// the resting pose. The returned kind retains the authored species name.
func (book *PinnipedProfiles) NewPinniped(species, pose string) (Kind, error) {
	profile, err := book.profile(species)
	if err != nil {
		return Kind{}, err
	}
	switch profile.Form {
	case "harbor-loft", "monk-loft":
		loft, err := pinnipeds.NewLoft(profile, pose)
		if err != nil {
			return Kind{}, err
		}
		return Kind{name: species, how: sealKind{form: sealLoft, loft: &loft}}, nil
	case "walrus":
		walrus, err := pinnipeds.NewWalrus(profile, pose, "skin")
		if err != nil {
			return Kind{}, err
		}
		return Kind{name: species, how: sealKind{form: sealWalrus, walrus: walrus}}, nil
	default:
		return Kind{}, fmt.Errorf("manufacture: unsupported pinniped form %q", profile.Form)
	}
}

// NewPinnipedRepresentation makes skin, a rest skeleton, or rest volumes.
// Not every scaffold has every representation; unsupported poses are refused.
func (book *PinnipedProfiles) NewPinnipedRepresentation(species, pose, representation string) (Kind, error) {
	if representation == "" || representation == "skin" {
		return book.NewPinniped(species, pose)
	}
	profile, err := book.profile(species)
	if err != nil {
		return Kind{}, err
	}
	switch representation {
	case "bones":
		if pose != "" {
			return Kind{}, fmt.Errorf("manufacture: pinniped skeleton is at rest")
		}
		switch profile.Form {
		case "harbor-loft":
			if profile.AnkleForward != 0 {
				return Kind{}, fmt.Errorf("manufacture: harbor-loft form has no forward ankle turn")
			}
			return Kind{name: species, how: sealKind{form: sealBones, profile: &profile}}, nil
		case "walrus":
			walrus, err := pinnipeds.NewWalrus(profile, "", "bones")
			if err != nil {
				return Kind{}, err
			}
			return Kind{name: species, how: sealKind{form: sealWalrus, walrus: walrus}}, nil
		}
	case "volumes":
		if profile.Form != "walrus" || (pose != "" && pose != "lying") {
			return Kind{}, fmt.Errorf("manufacture: volumes require a resting walrus")
		}
		walrus, err := pinnipeds.NewWalrus(profile, pose, "volumes")
		if err != nil {
			return Kind{}, err
		}
		return Kind{name: species, how: sealKind{form: sealWalrus, walrus: walrus}}, nil
	default:
		return Kind{}, fmt.Errorf("manufacture: unknown pinniped representation %q", representation)
	}
	return Kind{}, fmt.Errorf("manufacture: %s has no %s representation", species, representation)
}

// NewPinnipedStudy selects a profile-declared, fixed drawn-rig study.
// These studies use their own rigs and do not use the loft skeleton counts.
func (book *PinnipedProfiles) NewPinnipedStudy(species, name string) (Kind, error) {
	profile, err := book.profile(species)
	if err != nil {
		return Kind{}, err
	}
	study, ok := profile.Studies[name]
	if !ok {
		return Kind{}, fmt.Errorf("manufacture: no pinniped study %q for %q", name, species)
	}
	switch study.Scaffold {
	case "detailed-rig":
		return Kind{name: species, how: sealKind{form: sealDetail, look: pinnipeds.SealLook{Wet: *study.Wet, Nostrils: *study.Nostrils}}}, nil
	case "bay-rig":
		return Kind{name: species, how: sealKind{form: sealBay}}, nil
	default:
		return Kind{}, fmt.Errorf("manufacture: unsupported pinniped study scaffold %q", study.Scaffold)
	}
}

// NewHarborSeal constructs a hauled-out harbor seal.
func (book *PinnipedProfiles) NewHarborSeal() (Kind, error) {
	return book.NewPinniped("harbor-seal", "hauled")
}

// NewHawaiianMonkSeal constructs a hauled-out monk seal.
func (book *PinnipedProfiles) NewHawaiianMonkSeal() (Kind, error) {
	return book.NewPinniped("hawaiian-monk-seal", "hauled")
}

// NewWalrus constructs a resting walrus.
func (book *PinnipedProfiles) NewWalrus() (Kind, error) {
	return book.NewPinniped("walrus", "lying")
}

var defaultPinnipedProfiles = &PinnipedProfiles{book: pinnipeds.Default()}

func defaultPinniped(name string) pinnipedProfile {
	p, err := defaultPinnipedProfiles.profile(name)
	if err != nil {
		panic(err)
	}
	return p
}
