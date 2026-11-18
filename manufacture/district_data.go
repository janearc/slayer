package manufacture

import (
	_ "embed"
	"math"
	"strings"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed district-choices.yaml
var districtChoices []byte

var districtNumbers, districtNumbersError = numbers.Read("district-choices.yaml", districtChoices)

// districtValue reads checked authored dimensions and records provenance.
func districtValue(name string) float64 {
	if districtNumbersError != nil {
		panic(districtNumbersError)
	}
	number, err := districtNumbers.Get(name)
	if err != nil {
		panic(err)
	}
	if !positive(number.Value) || (strings.HasSuffix(name, "-share") && number.Value >= 1) {
		panic("manufacture: invalid district choice " + name)
	}
	count := name == "blocks-across" || name == "building-count" || name == "tree-prototypes"
	if count && number.Value != math.Trunc(number.Value) {
		panic("manufacture: fractional district count " + name)
	}
	noted(number)
	return number.Value
}
