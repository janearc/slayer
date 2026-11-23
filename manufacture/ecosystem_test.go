package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/biome"
)

// testEcosystem is a population of one recipe over a footprint that runs
// past the east edge of a 200 by 100 metre window whose corner is at 1000
// east and 500 south, on ground between 0 and 40 metres over a lowest point
// of 10 metres above the sea.
func testEcosystem(t *testing.T) Ecosystem {
	t.Helper()
	population, err := biome.New(biome.Definition{Name: "grove", Seed: 1,
		Bounds: biome.Rect{MinEast: 1100, MaxEast: 1500, MinSouth: 400,
			MaxSouth: 700},
		Cell: measure.Metres(50), PerHectare: 10,
		Species: []biome.Species{{Recipe: "spruce", Weight: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	return Ecosystem{Name: "grove", Population: population,
		Window: biome.Rect{MinEast: 1000, MaxEast: 1200, MinSouth: 500,
			MaxSouth: 600},
		Within: maths.Box{Max: maths.V(200, 40, 100)}, Lowest: 10,
		Above: math.Inf(-1), Below: math.Inf(1), Canopy: 0}
}

func TestAnEcosystemsBoxIsItsFootprintWhereItMeetsTheWindow(t *testing.T) {
	box, err := testEcosystem(t).Extent()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.X != 100 || box.Max.X != 200 || box.Min.Z != 0 ||
		box.Max.Z != 100 {
		t.Errorf("a footprint meeting the window gave the box %+v", box)
	}
	if box.Min.Y != 0 || box.Max.Y != 40 {
		t.Errorf("open band gave the box %+v, not the ground's heights", box)
	}
}

func TestAHeightBandHoldsAnEcosystemsBoxToItAndTheCanopyStandsOver(t *testing.T) {
	banded := testEcosystem(t)
	banded.Above, banded.Below, banded.Canopy = 20, 40, 12
	box, err := banded.Extent()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.Y != 10 || box.Max.Y != 30+12 {
		t.Errorf("a band of 20 to 40 over a lowest of 10 gave %+v", box)
	}
}

func TestAnEcosystemWithNoGroundInItsBandHasNoBox(t *testing.T) {
	high := testEcosystem(t)
	high.Above = 500
	if _, err := high.Extent(); err == nil {
		t.Error("a band over all the ground was given a box")
	}
	apart := testEcosystem(t)
	apart.Window = biome.Rect{MinEast: 0, MaxEast: 100, MinSouth: 0,
		MaxSouth: 100}
	if _, err := apart.Extent(); err == nil {
		t.Error("a footprint apart from the window was given a box")
	}
}

func TestAnEcosystemChoosesItsRootsOverTheWindowWithinABudget(t *testing.T) {
	selected, err := testEcosystem(t).Roots(biome.Budget{Cells: 1000,
		Candidates: 1000})
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range selected.Roots {
		if root.East < 1100 || root.East >= 1200 {
			t.Errorf("a root at %v east is outside the window's footprint",
				root.East)
		}
	}
	if _, err := testEcosystem(t).Roots(biome.Budget{}); err == nil {
		t.Error("a budget of no cells was enough to choose roots")
	}
}
