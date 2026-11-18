package manufacture

import (
	_ "embed"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed macro-choices.yaml
var macroChoices []byte

var macroNumbers, macroNumbersError = readMacroChoices(macroChoices)

// readMacroChoices checks positive dimensions, fractions and count budgets.
func readMacroChoices(raw []byte) (map[string]*numbers.Number, error) {
	book, err := numbers.Read("macro-choices.yaml", raw)
	if err != nil {
		return nil, err
	}
	indexed := map[string]*numbers.Number{}
	for i := range book {
		number := &book[i]
		if !positive(number.Value) || (number.Unit != "" && number.Unit != "m" && number.Unit != "deg") ||
			(strings.HasSuffix(number.Name, "-share") && number.Value >= 1) {
			return nil, fmt.Errorf("manufacture: invalid macro choice %s", number.Name)
		}
		dimensional := strings.HasSuffix(number.Name, "-width") ||
			strings.HasSuffix(number.Name, "-height") ||
			strings.HasSuffix(number.Name, "-depth") ||
			strings.HasSuffix(number.Name, "-radius") || number.Name == "maximum-length"
		if (dimensional && number.Unit != "m") ||
			(strings.HasSuffix(number.Name, "-degrees") && number.Unit != "deg") ||
			(strings.HasSuffix(number.Name, "-share") && number.Unit != "") {
			return nil, fmt.Errorf("manufacture: wrong units for macro choice %s", number.Name)
		}
		if strings.HasPrefix(number.Name,
			"segments-") || number.Name == "face-budget" || number.Name == "arcology-tiers" {
			if number.Unit != "" || number.Value != math.Trunc(number.Value) || number.Value > mostFaces {
				return nil, fmt.Errorf("manufacture: invalid macro count %s", number.Name)
			}
		}
		indexed[number.Name] = number
	}
	for _, name := range []string{"segments-hull", "segments-quick", "segments-demo",
		"arcology-tiers"} {
		if indexed[name] == nil || indexed[name].Value < 2 || indexed[name].Value > 64 {
			return nil, fmt.Errorf("manufacture: missing or excessive macro subdivisions %s", name)
		}
	}
	if indexed["face-budget"] == nil || indexed["face-budget"].Value > mostFaces {
		return nil, fmt.Errorf("manufacture: invalid macro face budget")
	}
	required := []string{"arcology-taper-share", "courtyard-share", "occupied-height-share",
		"park-thickness-share", "foundation-height-share", "rib-radius-share",
		"bridge-deck-height-share", "bridge-deck-thickness-share", "spire-width-share"}
	for _, name := range required {
		if indexed[name] == nil {
			return nil,
				fmt.Errorf("manufacture: missing macro proportion %s", name)
		}
	}
	value := func(name string) float64 { return indexed[name].Value }
	if 1-value("arcology-taper-share") <= value("courtyard-share")*1.1 ||
		value("park-thickness-share") >= (1-value("occupied-height-share"))/value("arcology-tiers") ||
		value("foundation-height-share") < 2*value("rib-radius-share") ||
		value("spire-width-share") > .2 ||
		value("bridge-deck-height-share")+value("bridge-deck-thickness-share")+value("park-thickness-share") >= 1 {
		return nil,
			fmt.Errorf("manufacture: macro proportions close a gallery or exceed the footprint")
	}
	return indexed, nil
}

// macroValue records authored dimensions in the normal maturity ledger.
func macroValue(name string) float64 {
	if macroNumbersError != nil {
		panic(macroNumbersError)
	}
	number := macroNumbers[name]
	if number == nil {
		panic("manufacture: missing macro choice " + name)
	}
	noted(number)
	return number.Value
}
