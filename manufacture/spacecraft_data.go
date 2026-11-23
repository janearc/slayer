package manufacture

import (
	_ "embed"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed spacecraft-choices.yaml
var spacecraftChoices []byte

var spacecraftNumbers, spacecraftNumbersError = numbers.Read(
	"spacecraft-choices.yaml", spacecraftChoices)

// spacecraftValue reads immutable authored geometry and appearance choices.
func spacecraftValue(name string) float64 {
	if spacecraftNumbersError != nil {
		panic(spacecraftNumbersError)
	}
	value, err := spacecraftNumbers.Get(name)
	if err != nil {
		panic(err)
	}
	noted(value)
	return value.Value
}

// primaValue reads the authored dimensions of the PRIMA visual concept.
func primaValue(name string) float64 {
	return spacecraftValue("prima-" + name)
}
