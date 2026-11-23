package manufacture

import _ "embed"

// voyagerNumbersRaw is voyager-numbers.yaml: Voyager's single chosen
// numbers, each with what it is, its unit and why it is what it is.
//
//go:embed voyager-numbers.yaml
var voyagerNumbersRaw []byte

// voyagerPointsRaw is voyager-points.csv: the placements and sizes of
// Voyager that are three numbers together, and its tables.
//
//go:embed voyager-points.csv
var voyagerPointsRaw []byte

// voyagerLooksRaw is voyager-looks.csv: what each material slot of Voyager
// is dressed in.
//
//go:embed voyager-looks.csv
var voyagerLooksRaw []byte

// voyagerBook is Voyager's numbers and points read and checked once, or the
// reason they cannot be. The assembly refuses with that reason.
var voyagerBook, voyagerBookError = readBook("voyager-numbers.yaml",
	voyagerNumbersRaw, "voyager-points.csv", voyagerPointsRaw)

// voyagerSlots are the words voyager-looks.csv gives the slots of Voyager.
var voyagerSlots = map[string]slot{
	"body":      body,
	"mirror":    spaceMirror,
	"structure": spaceStructure,
	"foil":      spaceFoil,
	"secondary": spaceSecondary,
	"panel":     spacePanel,
	"gold":      spaceGold,
}

// voyagerDress is voyager-looks.csv read and checked once, or the reason it
// cannot be. Each making dresses its own new materials from it.
var voyagerDress, voyagerDressError = readLooks("voyager-looks.csv",
	voyagerLooksRaw, bookUnits, voyagerSlots)
