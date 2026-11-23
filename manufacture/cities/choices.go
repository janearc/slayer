package cities

import (
	_ "embed"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed yurt.yaml
var yurtChoices []byte

//go:embed winnebago.yaml
var winnebagoChoices []byte

//go:embed bivouac.yaml
var bivouacChoices []byte

//go:embed scavenged.yaml
var scavengedChoices []byte

//go:embed slope.yaml
var slopeChoices []byte

//go:embed towerlattice.yaml
var towerLatticeChoices []byte

//go:embed frontage.yaml
var frontageChoices []byte

// choiceFile is one design's data file: its name, for the messages that
// refuse it, and what it holds.
type choiceFile struct {
	name string
	raw  []byte
}

// choiceBook is the named numbers of every design's data, read once. The
// designs share one book, so a name is one design's alone: the circle
// side counts are the only names that are everyone's, and are in the
// yurt's file.
var choiceBook, choiceBookError = readBook(
	choiceFile{"yurt.yaml", yurtChoices},
	choiceFile{"winnebago.yaml", winnebagoChoices},
	choiceFile{"bivouac.yaml", bivouacChoices},
	choiceFile{"scavenged.yaml", scavengedChoices},
	choiceFile{"slope.yaml", slopeChoices},
	choiceFile{"towerlattice.yaml", towerLatticeChoices},
	choiceFile{"frontage.yaml", frontageChoices})

// roleLooks is the material of each role the book colours.
var roleLooks, roleLooksError = readMaterials(choiceBook)

// readBook is the named numbers of several data files as one book. A name
// that two files give is refused, naming the second file, since a design
// would then be reading another's choice.
func readBook(files ...choiceFile) (map[string]numbers.Number, error) {
	merged := map[string]numbers.Number{}
	for _, file := range files {
		book, err := readChoices(file.name, file.raw)
		if err != nil {
			return nil, err
		}
		for name, number := range book {
			if _, twice := merged[name]; twice {
				return nil, fmt.Errorf("cities: %s names %s, which "+
					"another design's data already names", file.name, name)
			}
			merged[name] = number
		}
	}
	return merged, nil
}

// sidesPrefix begins the name of each circle's side count: the rest of
// the name is the detail level it is cut at.
const sidesPrefix = "circle-sides-"

// leastSides is the fewest flat sides that close a ring.
const leastSides = 3

// The ends of the names of a role's colour and gloss. A role is whatever
// data gives a red to: the code names the roles a design uses, and data
// says what each is made of.
const (
	redSuffix   = "-red"
	greenSuffix = "-green"
	blueSuffix  = "-blue"
	glossSuffix = "-gloss"
)

// readChoices is the named numbers of a data file, by name. Each must
// have the provenance numbers asks for, and a side count must be a whole
// number that closes a ring.
func readChoices(file string, raw []byte) (map[string]numbers.Number, error) {
	book, err := numbers.Read(file, raw)
	if err != nil {
		return nil, err
	}
	named := map[string]numbers.Number{}
	for _, number := range book {
		sides := strings.HasPrefix(number.Name, sidesPrefix)
		if sides && (number.Value != math.Trunc(number.Value) ||
			number.Value < leastSides) {
			return nil, fmt.Errorf("cities: %s is %v, and a ring has a "+
				"whole number of sides, at least %d", number.Name,
				number.Value, leastSides)
		}
		named[number.Name] = number
	}
	return named, nil
}

// choice is the value of a named number in the designs' data, or why
// there is none.
func choice(name string) (float64, error) {
	if choiceBookError != nil {
		return 0, choiceBookError
	}
	number, known := choiceBook[name]
	if !known {
		return 0, fmt.Errorf("cities: no choice named %q", name)
	}
	return number.Value, nil
}

// sidesAt is how many flat sides a circle is cut into at a level.
func sidesAt(level Detail) (int, error) {
	sides, err := choice(sidesPrefix + string(level))
	if err != nil {
		return 0, fmt.Errorf("cities: no circle at detail %q: %w", level, err)
	}
	return int(sides), nil
}

// readMaterials is a look for each role the book colours, shared: every
// ask for a role's look gets the same one, so the manufactory can tell
// two things of one material.
func readMaterials(book map[string]numbers.Number) (map[partRole]*thing.Material,
	error,
) {
	looks := map[partRole]*thing.Material{}
	for name := range book {
		role, coloured := strings.CutSuffix(name, redSuffix)
		if !coloured {
			continue
		}
		look, err := lookOf(book, role)
		if err != nil {
			return nil, err
		}
		looks[partRole(role)] = look
	}
	return looks, nil
}

// lookOf is one role's material, from its channels and its gloss. Each
// is a share of the light, from zero to one.
func lookOf(book map[string]numbers.Number, role string) (*thing.Material,
	error,
) {
	reader := &shareReader{book: book}
	colour := maths.RGB(reader.of(role+redSuffix),
		reader.of(role+greenSuffix), reader.of(role+blueSuffix))
	gloss := reader.of(role + glossSuffix)
	if reader.trouble != nil {
		return nil, reader.trouble
	}
	return &thing.Material{Name: "cities " + role, Colour: colour,
		Gloss: gloss}, nil
}

// shareReader reads named shares of the light from a book, and keeps the
// first it could not read.
type shareReader struct {
	book    map[string]numbers.Number
	trouble error
}

// of is a named number that is a share, from zero to one. After a
// refusal it reads nothing more and gives nought.
func (reader *shareReader) of(name string) float64 {
	number, known := reader.book[name]
	if reader.trouble != nil {
		return 0
	}
	if !known || !(number.Value >= 0 && number.Value <= 1) {
		reader.trouble = fmt.Errorf("cities: %s is missing, or not a share "+
			"of the light from zero to one", name)
		return 0
	}
	return number.Value
}

// material is what a role is made of, as the data says, or why there is
// no such role. Asking again for a role gives the same material.
func material(role partRole) (*thing.Material, error) {
	if choiceBookError != nil {
		return nil, choiceBookError
	}
	if roleLooksError != nil {
		return nil, roleLooksError
	}
	look, known := roleLooks[role]
	if !known {
		return nil, fmt.Errorf("cities: nothing is made of the role %q", role)
	}
	return look, nil
}
