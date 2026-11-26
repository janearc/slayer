package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/sauropods"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/snakes"
)

// AnimalOptions identifies one animal within a checked family book. Pointer
// fields distinguish an omitted option from an explicitly supplied zero.
type AnimalOptions struct {
	Species        string
	Pose           *string
	Count          *int
	Seconds        *float64
	Phase          *float64
	Representation *string
	Study          *string
}

// AnimalBook wraps one checked family book without registering its species.
type AnimalBook struct {
	family   string
	fish     *FishProfiles
	bird     *BirdProfiles
	pinniped *PinnipedProfiles
	theropod *TheropodProfiles
	squid    *SquidProfiles
	whale    *WhaleProfiles
	reptile  *ReptileProfiles
	snake    *SnakeProfiles
	sauropod *SauropodProfiles
}

// ReadAnimalBook checks and snapshots a family-specific authored book.
func ReadAnimalBook(family, file string, raw []byte) (*AnimalBook, error) {
	book := &AnimalBook{family: family}
	var err error
	switch family {
	case "fish":
		book.fish, err = ReadFishProfiles(file, raw)
	case "bird":
		book.bird, err = ReadBirdProfiles(file, raw)
	case "pinniped":
		book.pinniped, err = ReadPinnipedProfiles(file, raw)
	case "theropod":
		book.theropod, err = ReadTheropodProfiles(file, raw)
	case "squid":
		book.squid, err = ReadSquidProfiles(file, raw)
	case "whale":
		book.whale, err = ReadWhaleProfiles(file, raw)
	case "reptile":
		book.reptile, err = ReadReptileProfiles(file, raw)
	case "snake":
		book.snake, err = ReadSnakeProfiles(file, raw)
	case "sauropod":
		book.sauropod, err = ReadSauropodProfiles(file, raw)
	default:
		return nil, fmt.Errorf("manufacture: unknown animal family %q", family)
	}
	if err != nil {
		return nil, err
	}
	return book, nil
}

// New applies the family-specific pose and option rules to this book.
func (book *AnimalBook) New(options AnimalOptions) (Kind, error) {
	if book == nil {
		return Kind{}, fmt.Errorf("manufacture: no animal book")
	}
	if book.family != "squid" && options.Phase != nil {
		return Kind{}, fmt.Errorf("phase is only supported for squid")
	}
	if book.family != "pinniped" && book.family != "whale" && options.Representation != nil {
		return Kind{}, fmt.Errorf("representation is unsupported for %s", book.family)
	}
	if book.family != "pinniped" && book.family != "reptile" && options.Study != nil {
		return Kind{}, fmt.Errorf("study is unsupported for %s", book.family)
	}
	pose := ""
	if options.Pose != nil {
		if *options.Pose == "" {
			return Kind{}, fmt.Errorf("empty animal pose")
		}
		pose = *options.Pose
	}
	switch book.family {
	case "fish":
		if options.Seconds != nil || (pose != "" && pose != "swim" && pose != "glide" && pose != "bones") {
			return Kind{}, fmt.Errorf("invalid fish pose or time")
		}
		count := 0
		if options.Count != nil {
			count = *options.Count
		}
		return book.fish.NewFish(options.Species, FishOptions{Glide: pose == "glide", Bones: pose == "bones", School: count})
	case "bird":
		if options.Seconds != nil || options.Count != nil {
			return Kind{}, fmt.Errorf("bird does not take count or time")
		}
		if pose == "" {
			pose = "standing"
		}
		return book.bird.NewBird(options.Species, pose)
	case "pinniped":
		if options.Seconds != nil || options.Count != nil {
			return Kind{}, fmt.Errorf("pinniped does not take count or time")
		}
		if options.Study != nil {
			if *options.Study == "" || options.Pose != nil || options.Representation != nil {
				return Kind{}, fmt.Errorf("a pinniped study takes no pose or representation")
			}
			return book.pinniped.NewPinnipedStudy(options.Species, *options.Study)
		}
		representation := ""
		if options.Representation != nil {
			representation = *options.Representation
			if representation == "" {
				return Kind{}, fmt.Errorf("empty pinniped representation")
			}
		}
		return book.pinniped.NewPinnipedRepresentation(options.Species, pose, representation)
	case "squid":
		if options.Pose != nil || options.Count != nil || options.Seconds != nil {
			return Kind{}, fmt.Errorf("squid takes phase, not pose, count or time")
		}
		phase := 0.0
		if options.Phase != nil {
			phase = *options.Phase
		}
		return book.squid.NewSquidKind(options.Species, phase)
	case "whale":
		if options.Seconds != nil || options.Count != nil {
			return Kind{}, fmt.Errorf("whale does not take count or seconds")
		}
		representation := ""
		if options.Representation != nil {
			representation = *options.Representation
			if representation == "" {
				return Kind{}, fmt.Errorf("empty whale representation")
			}
		}
		return book.whale.NewWhale(options.Species, pose, representation)
	case "reptile":
		if options.Pose != nil || options.Count != nil || options.Seconds != nil {
			return Kind{}, fmt.Errorf("reptile study takes controls from its profile")
		}
		study := ""
		if options.Study != nil {
			study = *options.Study
			if study == "" {
				return Kind{}, fmt.Errorf("empty reptile study")
			}
		}
		return book.reptile.NewReptile(options.Species, study)
	case "snake":
		if options.Pose != nil || options.Count != nil || options.Seconds != nil {
			return Kind{}, fmt.Errorf("snake has no pose, count or time options")
		}
		return book.snake.NewSnake(options.Species)
	case "sauropod":
		if options.Pose != nil || options.Count != nil || options.Seconds != nil {
			return Kind{}, fmt.Errorf("sauropod has no pose, count or time options")
		}
		return book.sauropod.NewSauropod(options.Species)
	case "theropod":
		if options.Pose != nil || options.Count != nil {
			return Kind{}, fmt.Errorf("theropod takes time, not pose or count")
		}
		seconds := 0.
		if options.Seconds != nil {
			seconds = *options.Seconds
		}
		return book.theropod.NewTheropod(options.Species, seconds)
	default:
		return Kind{}, fmt.Errorf("manufacture: unknown animal family %q", book.family)
	}
}

// EmbeddedAnimalBook returns a copy of the named family's embedded profile
// book. Callers may validate or snapshot it without exposing mutable globals.
func EmbeddedAnimalBook(family string) (name string, raw []byte, err error) {
	switch family {
	case "fish":
		return "fish.yaml", append([]byte(nil), fishRaw...), nil
	case "bird":
		return "birds.yaml", append([]byte(nil), birdRaw...), nil
	case "pinniped":
		return "pinniped_profiles.yaml", append([]byte(nil), pinnipedProfilesRaw...), nil
	case "theropod":
		return "allosaurus-profiles.yaml", append([]byte(nil), allosaurusProfilesRaw...), nil
	case "squid":
		return "squid-profiles.yaml", append([]byte(nil), squidProfilesRaw...), nil
	case "whale":
		return "whale-profiles.yaml", append([]byte(nil), whaleProfilesRaw...), nil
	case "reptile":
		name, bytes := EmbeddedReptileBook()
		return name, bytes, nil
	case "snake":
		name, bytes := snakes.Embedded()
		return name, bytes, nil
	case "sauropod":
		name, bytes := sauropods.Embedded()
		return name, bytes, nil
	default:
		return "", nil, fmt.Errorf("manufacture: unknown animal family %q", family)
	}
}
