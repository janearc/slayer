package manufacture

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/janearc/puzzlemouth/numbers"
)

// cityDetailLengths are the street level's choices in metres.
const cityDetailLengths = "facade-projection reveal-depth module-height " +
	"bay-pitch rowhouse-window-width rowhouse-window-height " +
	"apartment-window-width apartment-window-height tower-window-width " +
	"tower-window-height sill-thickness sill-projection sill-overhang " +
	"mullion-width mullion-depth lintel-height lintel-projection " +
	"door-width door-height canopy-gap canopy-thickness canopy-overhang " +
	"pane-offset cornice-height parapet-height parapet-thickness " +
	"setback-depth plant-gap overrun-width overrun-depth overrun-height " +
	"cooler-width cooler-depth cooler-height tank-radius tank-height " +
	"mast-radius mast-height chimney-width chimney-depth " +
	"chimney-clearance louvre-height louvre-projection"

// cityDetailCounts are the street level's whole-number choices.
const cityDetailCounts = "setback-floors coolers plant-sides louvre-bands"

// cityPierReveals is how many reveal depths a pier keeps beside each
// opening's sill or canopy, so corner piers never reach an opening.
const cityPierReveals = 1.5

// checkCityDetailChoices checks the street level's units and counts,
// then that its parts fit: openings in their bays and strips, strips in
// the shortest storey, plant under the old roof top, a bay on every face.
func checkCityDetailChoices(book map[string]*numbers.Number) error {
	for _, name := range strings.Fields(cityDetailLengths + " " +
		cityDetailCounts) {
		number := book[name]
		unit := "m"
		if strings.Contains(cityDetailCounts, name) {
			unit = ""
		}
		if number == nil || number.Unit != unit || (unit == "" &&
			number.Value != math.Trunc(number.Value)) {
			return fmt.Errorf("manufacture: city choice %s has missing "+
				"or invalid units/count", name)
		}
	}
	if book["sill-share"] == nil {
		return fmt.Errorf("manufacture: missing city share sill")
	}
	value := func(name string) float64 { return book[name].Value }
	for _, fits := range cityDetailFits(value) {
		if !fits {
			return fmt.Errorf("manufacture: street-level city parts do " +
				"not fit their bays, storeys or roofs")
		}
	}
	return nil
}

// cityDetailFits is each condition the street level's parts need.
func cityDetailFits(value func(string) float64) []bool {
	pitch, tall := value("bay-pitch"), value("module-height")
	reveal, out := value("reveal-depth"), value("facade-projection")
	pier := func(width, over float64) bool {
		return (pitch-width)*spaceHalf-over >= cityPierReveals*reveal
	}
	fits := []bool{tall < value("storey-min"),
		pier(value("door-width"), value("canopy-overhang")),
		value("door-height")+value("canopy-gap")+
			value("canopy-thickness") <= tall,
		value("pane-offset") < value("mullion-depth"),
		value("mullion-depth") < reveal,
		slices.Max([]float64{value("sill-projection"),
			value("lintel-projection"), value("louvre-projection")}) <= out,
		value("chimney-clearance") < value("gable-rise")}
	for _, family := range []string{"rowhouse", "apartment", "tower"} {
		wide, high := value(family+"-window-width"),
			value(family+"-window-height")
		solid := tall - high
		fits = append(fits, pier(wide, value("sill-overhang")),
			solid*value("sill-share") >= value("sill-thickness"),
			solid*(1-value("sill-share")) >= value("lintel-height"))
	}
	return append(fits, cityRoofFits(value)...)
}

// cityRoofFits are the conditions for roofs and the narrowest faces:
// parapets and plant under the old roof top on the shortest storey,
// a whole bay on every face, and a tower's mast beside its crown.
func cityRoofFits(value func(string) float64) []bool {
	room := value("storey-min") + value("roof-thickness") -
		value("roof-seat") - value("cornice-height")
	tallest := slices.Max([]float64{value("parapet-height"),
		value("overrun-height"), value("cooler-height"), value("tank-height"),
		value("mast-height")})
	least := math.Min(value("width-min"), value("depth-min"))
	out, shaft := value("facade-projection"), value("shaft-width-share")
	faces := math.Min(least*shaft, least-cityWhole*value("setback-depth"))
	mast := least*shaft*spaceHalf - out -
		cityWhole*value("parapet-thickness") - value("mast-radius")
	return []bool{tallest <= room,
		faces-cityWhole*out >= value("bay-pitch"),
		mast >= least*value("crown-width-share")*spaceHalf+
			value("louvre-projection")}
}
