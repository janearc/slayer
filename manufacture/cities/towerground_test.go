package cities

import (
	"testing"
)

// groundParts are the parts of a tower's ground floor.
var groundParts = []string{"plinth", "shop glass", "revolving drum",
	"revolving leaf", "swing door", "canopy", "canopy rod", "roller door",
	"roller slat", "loading frame", "bollard", "shop sign", "awning", "shelf"}

func TestEveryDemoSkinHasAGroundFloorOfItsOwnAndTheQuickLevelHasNone(
	t *testing.T) {
	for skin := range detailSkins {
		pins := plainPins(skin)
		pins.Style = stylePunched
		quickTower := towerOf(t, 3, 0, pins)
		quick := counts(drawnTower(t, quickTower, "quick", 0))
		demoTower := towerOf(t, 3, 0, pins)
		demo := counts(drawnTower(t, demoTower, "demo", 0))
		for _, name := range groundParts {
			if quick[name] != 0 || demo[name] == 0 {
				t.Errorf("%s: %d %q at quick, %d at demo", skin, quick[name],
					name, demo[name])
			}
		}
	}
}

func TestNoSkinIsDressedOverTheGroundFloorButItsPiersRunDownBesideIt(
	t *testing.T) {
	pins := plainPins(skinGranite)
	pins.Style = stylePunched
	drawn := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
	storey := towerNumber(t, "storey-height")
	for _, name := range []string{"apron", "lintel", "sill"} {
		for _, one := range named(drawn, name) {
			if low, _ := boundsOf([]Rendered{one}); low.Y < -tolerance &&
				low.Y < storey {
				t.Errorf("a %s stands in the ground storey at %v", name,
					low.Y)
			}
		}
	}
	low, _ := boundsOf(named(drawn, "pier")[:1])
	if low.Y > tolerance {
		t.Errorf("a pier starts at %v, not on the ground", low.Y)
	}
}

func TestOnlyOneFaceInFourHasTheWayInAndOnlyOneTheLoadingBay(t *testing.T) {
	pins := plainPins(skinConcrete)
	demoTower := towerOf(t, 3, 0, pins)
	counted := counts(drawnTower(t, demoTower, "demo", 0))
	if counted["revolving drum"] != 1 || counted["canopy"] != 1 ||
		counted["roller door"] != 1 {
		t.Errorf("one way in and one loading door, not %d, %d and %d",
			counted["revolving drum"], counted["canopy"],
			counted["roller door"])
	}
}

func TestBollardsStandOutInFrontOfTheFaceAndNoneStandsInsideIt(t *testing.T) {
	pins := plainPins(skinCurtain)
	drawn := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
	posts := named(drawn, "bollard")
	if len(posts) == 0 {
		t.Fatal("no bollards")
	}
	out := towerNumber(t, "bollard-out")
	quickTower := towerOf(t, 3, 0, pins)
	_, quickHigh := boundsOf(drawnTower(t, quickTower, "quick",
		0))
	for _, one := range posts {
		low, high := boundsOf([]Rendered{one})
		if low.Y < -tolerance || high.Y > towerNumber(t, "bollard-height")+
			tolerance {
			t.Errorf("a bollard stands from %v to %v", low.Y, high.Y)
		}
		// bollards stand before every face, the west and north ones at
		// negative coordinates, so how far out one is is by its distance
		// from the middle, either way
		reach := max(high.X, -low.X, high.Z, -low.Z)
		if reach < out-tolerance && reach < quickHigh.X-tolerance {
			t.Errorf("a bollard at %v is not out in front of the face", reach)
		}
	}
}
