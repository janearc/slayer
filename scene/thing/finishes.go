package thing

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

// A finish is what was done to a thing's surface: how much it glosses,
// how broad a sheen it takes, a colour it has picked up, and, for an
// oxide, the clear film grown over it. Every number is picked for how
// it looks, not measured; finishes.yaml says so beside every finish,
// as ground-choices.yaml does for its numbers.

//go:embed finishes.yaml
var finishesRaw []byte

// finishes is the book of finishes, by name.
var finishes = readFinishes(finishesRaw)

// finishRow is one finish of the book: a few numbers, and what the finish
// says and how its numbers were chosen, which the file keeps for its
// reader.
type finishRow struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Name string `yaml:"name"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Gloss float64 `yaml:"gloss"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Satin float64 `yaml:"satin"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Colour []float64 `yaml:"colour"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Paint bool `yaml:"paint"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	FilmIndex float64 `yaml:"film-index"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	FilmThickness float64 `yaml:"film-thickness"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Says string `yaml:"says"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Kind string `yaml:"kind"`
}

// readFinishes refuses a broken embedded book before making any look.
func readFinishes(raw []byte) map[string]finishRow {
	var book struct{ Finishes []finishRow }
	if err := yamlfile.Read("finishes.yaml", raw, &book); err != nil {
		panic(err)
	}
	rows := make(map[string]finishRow, len(book.Finishes))
	for _, row := range book.Finishes {
		parts := len(row.Colour)
		if parts != metalNone && parts != metalBands {
			message := fmt.Sprintf(
				"finishes.yaml: %s wears %d parts of colour", row.Name, parts)
			panic(message)
		}
		rows[row.Name] = row
	}
	return rows
}

// Finish is what was done to a surface: one of the finishes the book
// names, and, for paint, the colour it was painted. A thing named in
// yaml says its finish by name, and a painted one says its colour with
// Painted.
type Finish struct {
	name   string
	colour maths.Light
}

// FinishOf is the finish a thing names, by name. It refuses a name the
// book does not carry, by name, and paint, whose colour Painted carries.
func FinishOf(name string) (Finish, error) {
	if _, known := finishes[name]; !known {
		return Finish{}, fmt.Errorf("thing: no finish %q", name)
	}
	if finishes[name].Paint {
		return Finish{}, fmt.Errorf(
			"thing: paint needs a colour, and Painted gives it")
	}
	return Finish{name: name}, nil
}

// Painted is paint of a colour, the one finish a material with no
// optics can wear: it covers whatever the thing is made of.
func Painted(colour maths.Light) Finish {
	return Finish{name: "painted", colour: colour}
}
