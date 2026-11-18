package manufacture

import (
	_ "embed"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed city-choices.yaml
var cityChoices []byte

var cityNumbers, cityNumbersError = readCityChoices(cityChoices)

// readCityChoices checks the same provenance records used by other families.
// Dimensions are positive metres; counts are positive integers and shares
// lie strictly between zero and one. Invalid embedded data makes no model.
func readCityChoices(raw []byte) (map[string]*numbers.Number, error) {
	book, err := numbers.Read("city-choices.yaml", raw)
	if err != nil {
		return nil, err
	}
	indexed := map[string]*numbers.Number{}
	for i := range book {
		number := &book[i]
		if !positive(number.Value) || (number.Unit != "" && number.Unit != "m") ||
			(strings.HasSuffix(number.Name, "-share") && number.Value >= 1) {
			return nil, fmt.Errorf("manufacture: invalid city choice %s", number.Name)
		}
		indexed[number.Name] = number
	}
	if err := checkCityChoices(indexed); err != nil {
		return nil, err
	}
	return indexed, nil
}

// cityValue records a named authored choice in the ordinary maturity ledger.
func cityValue(name string) float64 {
	if cityNumbersError != nil {
		panic(cityNumbersError)
	}
	number := cityNumbers[name]
	if number == nil {
		panic("manufacture: missing city choice " + name)
	}
	noted(number)
	return number.Value
}

// cityRange picks between two authored dimensional endpoints.
func cityRange(name string, share float64) float64 {
	low, high := cityValue(name+"-min"), cityValue(name+"-max")
	return low + (high-low)*share
}

// cityCount gives each integer in a family's storey range equal probability.
func cityCount(name string, share float64) int {
	low, high := cityValue(name+"-min"), cityValue(name+"-max")
	return int(low + math.Floor((high-low+1)*share))
}

// checkCityChoices checks units, counts, ranges and the finite bake budget.
func checkCityChoices(book map[string]*numbers.Number) error {
	dimensional := "block-length road-width alley-width setback roundabout-radius minimum-lot " +
		"width-min width-max depth-min depth-max storey-min storey-max roof-thickness " +
		"roof-seat gable-rise joint-overlap glazing-thickness glazing-offset glazing-height " +
		"island-height pavement-drop"
	counts := "blocks-across lots-across junction-interval population-limit chunk-faces " +
		"house-floors-min house-floors-max apartment-floors-min apartment-floors-max " +
		"tower-floors-min tower-floors-max glazing-bands circle-arc-steps"
	for _, name := range strings.Fields(dimensional + " " + counts) {
		number := book[name]
		unit := ""
		if strings.Contains(" "+dimensional+" ", " "+name+" ") {
			unit = "m"
		}
		if number == nil || number.Unit != unit ||
			(unit == "" && number.Value != math.Trunc(number.Value)) {
			return fmt.Errorf("manufacture: city choice %s has missing or invalid units/count", name)
		}
	}
	shares := "podium-height shaft-width shaft-height crown-width crown-base " +
		"crown-height crown-roof glazing-width glazing-depth island-radius"
	for _, name := range strings.Fields(shares) {
		if book[name+"-share"] == nil || book[name+"-share"].Unit != "" {
			return fmt.Errorf("manufacture: missing city share %s", name)
		}
	}
	projection := book["glazing-offset"].Value +
		book["glazing-thickness"].Value/2
	width := math.Max(book["width-max"].Value, book["depth-max"].Value)
	if width+2*projection > book["minimum-lot"].Value ||
		book["roof-seat"].Value >= book["roof-thickness"].Value ||
		book["roof-seat"].Value >= book["gable-rise"].Value {
		return fmt.Errorf("manufacture: city building envelope exceeds its lot or roof")
	}
	for _, name := range []string{"width", "depth", "storey", "house-floors",
		"apartment-floors", "tower-floors"} {
		if book[name+"-min"].Value > book[name+"-max"].Value {
			return fmt.Errorf("manufacture: reversed city range %s", name)
		}
	}
	if book["chunk-faces"].Value > mostFaces || book["chunk-faces"].Value < 256 ||
		book["population-limit"].Value > 100_000 || book["tower-floors-max"].Value > 64 {
		return fmt.Errorf("manufacture: city recipe exceeds construction budgets")
	}
	return checkCityDetailChoices(book)
}
