package manufacture

import "testing"

// retailTowerMost is how far past its plan a tower with shops may reach, in
// metres: an awning, a step, goods on the pavement, and no more. Far less
// than a podium.
const retailTowerMost = 3.0

// retailTower is a tower of a recipe with retail as given, nil for not
// pinned, of the plan and the number of shafts given, the plan being a
// rectangle if it is empty.
func retailTower(t *testing.T, retail *bool, plan string, shafts int) good {
	t.Helper()
	off := false
	if plan == "" {
		plan = "rectangle"
	}
	row, err := Tower(TowerSpec{Plan: plan, Skin: "concrete", Storeys: 8,
		Width: 36, Shafts: shafts, Taper: &off, Twist: &off, Setbacks: &off,
		Spire: &off, Retail: retail})
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(row, Sow(21), "quick")
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// A tower with retail pinned on has shops in its foot: more lamps, one
// storey of a unit taller, and no wider than an awning, a step and goods on
// the pavement: its shops stand flush, not out in the street.
func TestTowerWithRetailHasShopsInItsFoot(t *testing.T) {
	on := true
	plain, shops := retailTower(t, nil, "", 0), retailTower(t, &on, "", 0)
	if len(shops.lights) <= len(plain.lights) {
		t.Errorf("%d lamps with retail, %d without", len(shops.lights),
			len(plain.lights))
	}
	before, after := plain.Bounds(), shops.Bounds()
	if rise := after.Max.Y - before.Max.Y; rise < cityValue("storey-max")-
		retailEdgeSlack {
		t.Errorf("the tower is %g m taller, want a storey", rise)
	}
	for _, grown := range []float64{before.Min.X - after.Min.X,
		after.Max.X - before.Max.X, before.Min.Z - after.Min.Z,
		after.Max.Z - before.Max.Z} {
		if grown > retailTowerMost {
			t.Errorf("shops stand %g m out of the tower, at most %g",
				grown, retailTowerMost)
		}
	}
}

// Retail is off unless pinned on: a tower pinned off, or not pinned, is the
// same tower, faces and lamps, and no taller.
func TestTowerRetailIsOffByDefault(t *testing.T) {
	off := false
	plain, pinned := retailTower(t, nil, "", 0), retailTower(t, &off, "", 0)
	if plain.faces() != pinned.faces() ||
		len(plain.lights) != len(pinned.lights) ||
		plain.Bounds() != pinned.Bounds() {
		t.Error("a tower pinned off is not the tower that is not pinned")
	}
}

// A tower that stands on a base has no shops, as its foot is not the
// ground: pinning retail on changes nothing, not even its height.
func TestTowerOnABaseTakesNoShops(t *testing.T) {
	on := true
	plain, pinned := retailTower(t, nil, "", 2), retailTower(t, &on, "", 2)
	if plain.faces() != pinned.faces() ||
		len(plain.lights) != len(pinned.lights) ||
		plain.Bounds() != pinned.Bounds() {
		t.Error("a tower on a base was given shops")
	}
}

// A tower of any plan is either lifted on an open storey and given shops, or
// left as it is: never one without the other. The open storey is only made
// for a tower that can fill it.
func TestTowerIsLiftedOnlyToTakeShops(t *testing.T) {
	on := true
	for _, plan := range TowerPlans() {
		plain, pinned := retailTower(t, nil, plan, 0), retailTower(t, &on, plan, 0)
		lifted := pinned.Bounds().Max.Y > plain.Bounds().Max.Y
		shops := len(pinned.lights) > len(plain.lights)
		if lifted != shops {
			t.Errorf("plan %s: lifted %v, shops %v", plan, lifted, shops)
		}
	}
}

// The shops of a tower are the same for a seed.
func TestTowerRetailIsTheSameForASeed(t *testing.T) {
	on := true
	first, second := retailTower(t, &on, "", 0), retailTower(t, &on, "", 0)
	if first.faces() != second.faces() || len(first.lights) != len(second.lights) {
		t.Error("one seed made two different towers with shops")
	}
}

// A stretch of wall too short for a unit has no shops, and no error.
func TestRetailStripWantsRoom(t *testing.T) {
	_, found, err := retailStrip(Sow(1), 1, 1)
	if err != nil || found {
		t.Errorf("a strip of 1 m: found %v, error %v", found, err)
	}
	_, found, err = retailStrip(Sow(1), 1, 30)
	if err != nil || !found {
		t.Errorf("a strip of 30 m: found %v, error %v", found, err)
	}
}
