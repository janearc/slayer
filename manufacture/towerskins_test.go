package manufacture

import (
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

func TestAPlantedTowersTreesAreKindsTheCatalogueHasAndGrowOnTheTower(
	t *testing.T) {
	yes := true
	planted := TowerSpec{Skin: "concrete", Planted: &yes, Storeys: 30}
	tower, err := cities.NewTower(3, cities.Irregularity{},
		cities.Pins{Skin: "concrete", Planted: &yes, Storeys: 30})
	if err != nil {
		t.Fatal(err)
	}
	design, err := tower.Design("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	// A design draws its segments when it is rendered, and the trees are
	// grown with them.
	if _, err := design.Rendered(cities.Detail("quick")); err != nil {
		t.Fatal(err)
	}
	if len(tower.Trees()) == 0 {
		t.Fatal("a planted tower has no trees")
	}
	for _, tree := range tower.Trees() {
		if _, err := KindNamed(tree.Kind); err != nil {
			t.Errorf("the catalogue has no %q: %v", tree.Kind, err)
		}
	}
	row, err := Tower(planted)
	if err != nil {
		t.Fatal(err)
	}
	bare, err := Tower(TowerSpec{Skin: "concrete", Storeys: 30})
	if err != nil {
		t.Fatal(err)
	}
	grown, err := makeOf(row, sow(3))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := makeOf(bare, sow(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(grown.groups) <= len(plain.groups) {
		t.Errorf("trees added no groups: %d with, %d without",
			len(grown.groups), len(plain.groups))
	}
}

// TestATowerThatIsNotPlantedHoldsNoTrees is the anti-test of the planted
// tower: the same tower, drawn and rendered, holds none.
func TestATowerThatIsNotPlantedHoldsNoTrees(t *testing.T) {
	no := false
	tower, err := cities.NewTower(3, cities.Irregularity{},
		cities.Pins{Skin: "concrete", Planted: &no, Storeys: 30})
	if err != nil {
		t.Fatal(err)
	}
	design, err := tower.Design("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := design.Rendered(cities.Detail("quick")); err != nil {
		t.Fatal(err)
	}
	if trees := tower.Trees(); len(trees) != 0 {
		t.Errorf("a tower that is not planted holds %d trees", len(trees))
	}
}
