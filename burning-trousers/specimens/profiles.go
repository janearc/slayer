// Package specimens supplies profile-backed kinds to the independent harness.
package specimens

import (
	"fmt"
	"maps"
	"slices"

	"github.com/janearc/slayer/manufacture"
	"gopkg.in/yaml.v3"
)

// Specimen names a checked kind and, when present, its pre-profile kind name.
// Legacy is only a test witness; it does not register an authored alias.
type Specimen struct {
	Name   string
	Kind   manufacture.Kind
	Legacy string
}

type plantBook interface {
	Species() []string
	New(manufacture.PlantOptions) (manufacture.Kind, error)
}

type animalBook interface {
	New(manufacture.AnimalOptions) (manufacture.Kind, error)
}

type algaBook interface {
	NewKelp(manufacture.KelpOptions) (manufacture.Kind, error)
}

type algaDocument struct {
	Species    string               `yaml:"species"`
	Conditions map[string]yaml.Node `yaml:"conditions"`
}

type animalDocument struct {
	Species  map[string]yaml.Node `yaml:"species"`
	Profiles []struct {
		Kind     string               `yaml:"kind"`
		Form     string               `yaml:"form"`
		Scaffold string               `yaml:"scaffold"`
		Studies  map[string]yaml.Node `yaml:"studies"`
	} `yaml:"profiles"`
}

// sources is a boundary around the authored books and their checked readers.
// Tests can supply broken inputs and verify that the inventory refuses them.
type sources struct {
	plant      func(string) (string, []byte, error)
	readPlant  func(string, string, []byte) (plantBook, error)
	animal     func(string) (string, []byte, error)
	readAnimal func(string, string, []byte) (animalBook, error)
	decode     func([]byte, *animalDocument) error
	alga       func(string) (string, []byte, error)
	readAlga   func(string, []byte) (algaBook, error)
	decodeAlga func([]byte, *algaDocument) error
}

func embeddedSources() sources {
	return sources{
		plant: manufacture.EmbeddedPlantBook,
		readPlant: func(family, file string, raw []byte) (plantBook, error) {
			return manufacture.ReadPlantBook(family, file, raw)
		},
		animal: manufacture.EmbeddedAnimalBook,
		readAnimal: func(family, file string, raw []byte) (animalBook, error) {
			return manufacture.ReadAnimalBook(family, file, raw)
		},
		decode: func(raw []byte, document *animalDocument) error {
			return yaml.Unmarshal(raw, document)
		},
		alga: manufacture.EmbeddedAlgaBook,
		readAlga: func(file string, raw []byte) (algaBook, error) {
			return manufacture.ReadKelpProfiles(file, raw)
		},
		decodeAlga: func(raw []byte, document *algaDocument) error {
			return yaml.Unmarshal(raw, document)
		},
	}
}

// Profiles constructs the embedded species and the distinct poses, studies,
// schools and representations formerly exercised by the harness. Reading the
// books each time keeps the inventory sensitive to species added later.
func Profiles() ([]Specimen, error) {
	return profiles(embeddedSources())
}

func profiles(input sources) ([]Specimen, error) {
	var all []Specimen
	for _, family := range []string{"seasonal", "conifer", "fern", "cactus"} {
		file, raw, err := input.plant(family)
		if err != nil {
			return nil, err
		}
		book, err := input.readPlant(family, file, raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		for _, species := range book.Species() {
			kind, err := book.New(manufacture.PlantOptions{Species: species})
			if err != nil {
				return nil, err
			}
			all = append(all, Specimen{"plant/" + family + "/" + species + "/whole", kind, species})
			if family == "conifer" && (species == "coast-redwood" || species == "giant-sequoia") {
				shoot := "shoot"
				kind, err = book.New(manufacture.PlantOptions{Species: species, Representation: &shoot})
				if err != nil {
					return nil, err
				}
				all = append(all, Specimen{"plant/conifer/" + species + "/shoot", kind, species + "-shoot"})
			}
		}
	}
	file, raw, err := input.alga("kelp")
	if err != nil {
		return nil, err
	}
	book, err := input.readAlga(file, raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	var kelp algaDocument
	if err := input.decodeAlga(raw, &kelp); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	for _, form := range []struct {
		name, legacy string
		forest       bool
	}{{"plant", "giant-kelp", false}, {"forest", "kelp-forest", true}} {
		conditions := append([]string{""}, slices.Sorted(maps.Keys(kelp.Conditions))...)
		for _, condition := range conditions {
			kind, err := book.NewKelp(manufacture.KelpOptions{
				Species: kelp.Species, Condition: condition, Forest: form.forest,
			})
			if err != nil {
				return nil, fmt.Errorf("kelp %s %s: %w", kelp.Species, condition, err)
			}
			variant, legacy := condition, ""
			if condition == "" {
				variant = "default"
			}
			if kelp.Species == "macrocystis-pyrifera" {
				switch condition {
				case "":
					legacy = form.legacy
				case "slack", "moderate", "surge":
					legacy = form.legacy + "-" + condition
				}
			}
			all = append(all, Specimen{"alga/kelp/" + kelp.Species + "/" + form.name + "/" + variant, kind, legacy})
		}
	}
	for _, family := range []string{"fish", "bird", "pinniped", "theropod", "squid", "whale", "reptile"} {
		file, raw, err := input.animal(family)
		if err != nil {
			return nil, err
		}
		book, err := input.readAnimal(family, file, raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		var document animalDocument
		if err := input.decode(raw, &document); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		add := func(species, variant, legacy string, opts manufacture.AnimalOptions) error {
			opts.Species = species
			kind, err := book.New(opts)
			if err != nil {
				return fmt.Errorf("%s %s %s: %w", family, species, variant, err)
			}
			all = append(all, Specimen{"animal/" + family + "/" + species + "/" + variant, kind, legacy})
			return nil
		}
		for _, species := range slices.Sorted(maps.Keys(document.Species)) {
			switch family {
			case "fish":
				for _, pose := range []string{"swim", "glide", "bones"} {
					legacy := species
					if pose != "swim" {
						legacy += "-" + pose
					}
					p := pose
					if err := add(species, pose, legacy, manufacture.AnimalOptions{Pose: &p}); err != nil {
						return nil, err
					}
				}
			case "bird":
				for _, pose := range []string{"standing", "gliding", "bones", "outline"} {
					p := pose
					if err := add(species, pose, "bird-"+species+"-"+pose, manufacture.AnimalOptions{Pose: &p}); err != nil {
						return nil, err
					}
				}
			}
		}
		if family == "fish" {
			for _, school := range []struct {
				species string
				count   int
			}{{"pacific-sardine", 500}, {"pacific-sardine", 5000}, {"mediterranean-chromis", 60}, {"flathead-grey-mullet", 120}} {
				if _, exists := document.Species[school.species]; !exists {
					return nil, fmt.Errorf("missing school species %s", school.species)
				}
				variant := fmt.Sprintf("school-%d", school.count)
				if err := add(school.species, variant, school.species+"-"+variant, manufacture.AnimalOptions{Count: &school.count}); err != nil {
					return nil, err
				}
			}
		}
		for _, profile := range document.Profiles {
			species := profile.Kind
			switch family {
			case "reptile":
				studies := map[string][]string{
					"pliosaur":    {"baseline", "continuous"},
					"crocodylian": {"study"},
					"pterosaur":   {"composite", "cage", "density"},
				}[profile.Scaffold]
				if len(studies) == 0 {
					return nil, fmt.Errorf("missing reptile studies for %s", profile.Scaffold)
				}
				for _, study := range studies {
					s := study
					if err := add(species, study, "", manufacture.AnimalOptions{Study: &s}); err != nil {
						return nil, err
					}
				}
			case "whale":
				for _, variant := range []struct{ name, legacy, pose, representation string }{
					{"swim", "orca-clean-05", "swim", ""}, {"spy-hop", "orca-clean-05-spy-hop", "spy-hop", ""}, {"bones", "orca-clean-05-bones", "", "bones"},
				} {
					options := manufacture.AnimalOptions{}
					if variant.pose != "" {
						p := variant.pose
						options.Pose = &p
					}
					if variant.representation != "" {
						r := variant.representation
						options.Representation = &r
					}
					if err := add(species, variant.name, variant.legacy, options); err != nil {
						return nil, err
					}
				}
			case "pinniped":
				variants, err := pinnipedVariants(species, profile.Form, profile.Studies)
				if err != nil {
					return nil, err
				}
				for _, variant := range variants {
					if err := add(species, variant.name, variant.legacy, variant.options); err != nil {
						return nil, err
					}
				}
			case "theropod", "squid":
				if err := add(species, "default", "", manufacture.AnimalOptions{}); err != nil {
					return nil, err
				}
			}
		}
	}
	return all, nil
}

type pinnipedVariant struct {
	name, legacy string
	options      manufacture.AnimalOptions
}

func pinnipedVariants(species, form string, studies map[string]yaml.Node) ([]pinnipedVariant, error) {
	var variants []pinnipedVariant
	pose := func(name, legacy string) {
		p := name
		variants = append(variants, pinnipedVariant{name, legacy, manufacture.AnimalOptions{Pose: &p}})
	}
	repr := func(name, legacy string) {
		r := name
		variants = append(variants, pinnipedVariant{name, legacy, manufacture.AnimalOptions{Representation: &r}})
	}
	switch form {
	case "harbor-loft":
		pose("hauled", species+"-hauled")
		pose("swimming", species+"-swimming")
		repr("bones", species+"-bones")
	case "monk-loft":
		pose("hauled", species+"-hauled")
	case "walrus":
		// These historical labels apply to the original walrus profile.
		prefix := species
		if species == "walrus" {
			prefix = "walrus-fresh-5"
		}
		pose("lying", prefix)
		pose("hauled-out", prefix+"-hauled-out")
		pose("swimming", prefix+"-swimming")
		repr("bones", prefix+"-skeleton")
		repr("volumes", prefix+"-volumes")
	default:
		return nil, fmt.Errorf("%s: unknown pinniped form %q", species, form)
	}
	for _, study := range slices.Sorted(maps.Keys(studies)) {
		legacy := ""
		if species == "harbor-seal" {
			switch study {
			case "detailed-wet":
				legacy = "harbor-seal"
			case "detailed-dry":
				legacy = "harbor-seal-dry"
			case "bay":
				legacy = "bay-seal"
			}
		}
		v := study
		variants = append(variants, pinnipedVariant{"study-" + study, legacy, manufacture.AnimalOptions{Study: &v}})
	}
	return variants, nil
}
