package cities

import (
	"testing"
)

// useTower is a plain curtain wall tower of seed three, of one use, drawn
// at the demo level.
func useTower(t *testing.T, use Use) []Rendered {
	t.Helper()
	pins := plainPins(skinCurtain)
	pins.Use = use
	return drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
}

func TestOfficeAndHomeBandsDifferAtNightBehindTheGlass(t *testing.T) {
	offices, homes := counts(useTower(t, useOffice)),
		counts(useTower(t, useHome))
	for _, name := range []string{"ceiling light", "desk", "screen",
		"chair", "blind", "partition"} {
		if offices[name] == 0 || homes[name] != 0 {
			t.Errorf("%q: %d in offices, %d in homes", name, offices[name],
				homes[name])
		}
	}
	if offices["lit wall"] == 0 || homes["lit wall"] == 0 {
		t.Errorf("lit walls: %d in offices, %d in homes", offices["lit wall"],
			homes["lit wall"])
	}
	for _, name := range []string{"side wall", "room floor", "couch",
		"curtain"} {
		if homes[name] == 0 || offices[name] != 0 {
			t.Errorf("%q: %d in homes, %d in offices", name, homes[name],
				offices[name])
		}
	}
}

// An office has a back wall in front of the core, of the rooms' own
// colour, which its lamp lights; a home has its own side walls and none.
func TestAnOfficeHasABackWallAndAHomeNone(t *testing.T) {
	offices, homes := counts(useTower(t, useOffice)),
		counts(useTower(t, useHome))
	if offices["back wall"] == 0 || homes["back wall"] != 0 {
		t.Errorf("back walls: %d in offices, %d in homes",
			offices["back wall"], homes["back wall"])
	}
}

// Anti: an office's back wall glows with no light of its own; only the
// lit wall that stands in for it in an office with no lamp glows.
// anti-test of TestAnOfficeHasABackWallAndAHomeNone
func TestAnOfficesBackWallDoesNotGlow(t *testing.T) {
	for _, wall := range named(useTower(t, useOffice), "back wall") {
		if wall.Material != nil && wall.Material.Glow.Luminance() > 0 {
			t.Fatalf("a back wall glows: %v", wall.Material.Glow)
		}
	}
}

func TestEveryLampOfARoomIsALightAndOnlyOnALitFloorOrHome(t *testing.T) {
	for _, use := range UseNames() {
		lamps := named(useTower(t, use), "lamp")
		if len(lamps) == 0 {
			t.Errorf("%s: no lamps", use)
		}
		for _, one := range lamps {
			if one.Lamp == nil || one.Lamp.Luminance <= 0 ||
				one.Lamp.Heat <= 0 {
				t.Errorf("%s: a lamp that is no light: %v", use, one.Lamp)
			}
		}
	}
	offices := named(useTower(t, useOffice), "lamp")
	homes := named(useTower(t, useHome), "lamp")
	if offices[0].Lamp.Heat <= homes[0].Lamp.Heat {
		t.Errorf("an office lamp is %vK and a home's %vK: wanted the office "+
			"cooler", offices[0].Lamp.Heat, homes[0].Lamp.Heat)
	}
}

func TestTheRoomsAreHollowBehindTheGlassOnlyAtTheDemoLevel(t *testing.T) {
	pins := plainPins(skinCurtain)
	quick := drawnTower(t, towerOf(t, 3, 0, pins), "quick", 0)
	demo := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
	if len(named(quick, "floor slab")) != 0 ||
		len(named(demo, "floor slab")) == 0 {
		quickSlabs := len(named(quick, "floor slab"))
		demoSlabs := len(named(demo, "floor slab"))
		t.Errorf("floor slabs: %d at quick, %d at demo",
			quickSlabs, demoSlabs)
	}
	cavity := towerNumber(t, "room-cavity")
	low, high := boundsOf(named(quick, "glass"))
	bodyLow, bodyHigh := boundsOf(named(demo, "body"))
	if !(bodyHigh.X < high.X-cavity/2) || !(bodyLow.X > low.X+cavity/2) {
		t.Errorf("the core wall is %v to %v, the glass body %v to %v",
			bodyLow.X, bodyHigh.X, low.X, high.X)
	}
}

func TestTheRoomsOfANarrowSegmentAreNeverDeeperThanAShareOfItsPlan(
	t *testing.T) {
	pins := plainPins(skinCurtain)
	pins.Width = 8
	drawn := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
	share := towerNumber(t, "room-cavity-share")
	bodyLow, bodyHigh := boundsOf(named(drawn, "body"))
	if bodyHigh.X-bodyLow.X < 8*(1-2*share)-1 {
		t.Errorf("an eight metre tower has a core wall %v across",
			bodyHigh.X-bodyLow.X)
	}
}
