package cities

import (
	"math"
	"testing"
)

func TestTheGroundStoreyIsTallerAndTheFloorsAboveItStandOnIt(t *testing.T) {
	facade := facade{Height: 10, Ground: 1,
		Storeys: make([][]facadeOpening, 3)}
	if got := facade.heightOf(0); math.Abs(got-4) > tolerance {
		t.Errorf("the ground storey is %v, not 4", got)
	}
	for index, want := range []float64{0, 4, 7} {
		if got := facade.floorAt(index); math.Abs(got-want) > tolerance {
			t.Errorf("floor %d is at %v, not %v", index, got, want)
		}
	}
	var sum float64
	for index := range facade.Storeys {
		sum += facade.heightOf(index)
	}
	if math.Abs(sum-facade.Height) > tolerance {
		t.Errorf("the storeys add up to %v, not %v", sum, facade.Height)
	}
}

func TestAtTheDemoLevelAHouseIsTallerByTheGroundStoreysExtra(t *testing.T) {
	extra := drawerOf(t, "portofino", 1).fixed("ground-extra")
	quick := highest(terraceAt(t, "portofino", 3, "quick"))
	demo := highest(terraceAt(t, "portofino", 3, "demo"))
	if demo < quick+extra-tolerance {
		t.Errorf("a row is %v high at the demo level and %v at quick, the "+
			"extra is %v", demo, quick, extra)
	}
}

func TestTheGroundFloorHasAPlinthShopfrontsArchedDoorsAndSteps(t *testing.T) {
	found := map[string]int{}
	wide := 0
	for _, culture := range []string{"portofino", "groningen", "midwest"} {
		for seed := range Seed(detailSeeds) {
			for _, one := range terraceAt(t, culture, seed, "demo") {
				found[one.Mesh.Name]++
				if one.Mesh.Name == "light" {
					low, high := reach(one)
					wide += btoi(high.X-low.X > 0.8)
				}
				if !closed(one.Mesh) {
					t.Fatalf("%s seed %d: %s is not closed", culture, seed,
						one.Mesh.Name)
				}
			}
		}
	}
	for _, part := range []string{"plinth", "spandrel", "stoop", "step"} {
		if found[part] == 0 {
			t.Errorf("no %s", part)
		}
	}
	if wide == 0 {
		t.Error("no shopfront: no light of a window is wider than 0.8 m")
	}
}

func TestBelowTheDemoLevelThereIsNoPlinthOrArchOrStoop(t *testing.T) {
	for _, level := range []Detail{"hull", "quick"} {
		drawn := terraceAt(t, "portofino", 3, level)
		for _, part := range []string{"plinth", "spandrel", "stoop"} {
			if got := len(named(drawn, part)); got != 0 {
				t.Errorf("%d %s at %s", got, part, level)
			}
		}
	}
}
