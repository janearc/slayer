package set_test

import (
	"strings"
	"testing"
)

// leastWaterAnchor is the start of the first stand's line in the mangrove
// shore set, which the tests add a least water to.
const leastWaterAnchor = "kandelia a: {recipe: plant-kandelia-obovata, "

// withLeastWater is a set text whose first stand says the least water.
func withLeastWater(said string) func(string) string {
	return func(text string) string {
		return strings.Replace(text, leastWaterAnchor,
			leastWaterAnchor+"least-water: "+said+", ", 1)
	}
}

func TestStandReadsItsLeastWaterAndDefaultsToThreeMetres(test *testing.T) {
	made, err := readMangroveSet(test, "mangrove-shore.yaml",
		withLeastWater("{value: 0.25, unit: m}"))
	if err != nil {
		test.Fatal(err)
	}
	stands := made.Stands()
	if stands[0].LeastWater() != 0.25 {
		test.Fatal("lost the least water", stands[0].LeastWater())
	}
	if stands[1].LeastWater() != 3 {
		test.Fatal("a stand that says none has", stands[1].LeastWater())
	}
}

func TestStandRefusesALeastWaterThatIsNotADepth(test *testing.T) {
	for _, said := range []string{
		"{value: 0, unit: m}",
		"{value: -1, unit: m}",
		"{value: 0.25, unit: km}",
		"{value: 0.25}",
		"{value: .nan, unit: m}",
	} {
		_, err := readMangroveSet(test, "mangrove-shore.yaml",
			withLeastWater(said))
		if err == nil {
			test.Fatal("accepted a least water it should refuse:", said)
		}
	}
}
