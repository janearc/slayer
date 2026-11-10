package manufacture

import (
	_ "embed"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed ground-choices.yaml
var groundChoices []byte

var groundNumbers = readGroundChoices(groundChoices)

// readGroundChoices refuses a broken embedded book before making any kind.
func readGroundChoices(raw []byte) numbers.Numbers {
	book, err := numbers.Read("ground-choices.yaml", raw)
	if err != nil {
		panic(err)
	}
	return book
}

// groundValue reads immutable authored choices, not measured dimensions.
func groundValue(name string) float64 {
	number, err := groundNumbers.Get(name)
	if err != nil {
		panic(err)
	}
	return number.Value
}
