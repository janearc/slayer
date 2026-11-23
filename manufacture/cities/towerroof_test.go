package cities

import (
	"strings"
	"testing"
)

// roofParts are what stands on a demo tower's roof, where a writer goes.
var roofParts = []string{"roof", "parapet", "water tank", "tank lid",
	"tank leg", "tank hoop", "billboard back", "billboard face", "catwalk",
	"billboard post"}

func TestADemoRoofHasItsParapetTankAndBillboardAndAQuickRoofNone(
	t *testing.T) {
	pins := plainPins(skinConcrete)
	quickTower := towerOf(t, 3, 0, pins)
	quick := counts(drawnTower(t, quickTower, "quick", 0))
	demoTower := towerOf(t, 3, 0, pins)
	demo := counts(drawnTower(t, demoTower, "demo", 0))
	for _, name := range roofParts {
		if quick[name] != 0 || demo[name] == 0 {
			t.Errorf("%d %q at quick, %d at demo", quick[name], name,
				demo[name])
		}
	}
	if demo["water tank"] != 1 || demo["tank leg"] != 4 ||
		demo["tank hoop"] != int(towerNumber(t, "tank-hoops")) {
		t.Errorf("a tank of %d, on %d legs with %d hoops",
			demo["water tank"], demo["tank leg"], demo["tank hoop"])
	}
}

func TestASpireCarriesNoTankOrBillboardButItsRoofHasAParapet(
	t *testing.T) {
	pins := plainPins(skinConcrete)
	pins.Spire = &yes
	demoTower := towerOf(t, 3, 0, pins)
	demo := counts(drawnTower(t, demoTower, "demo", 0))
	if demo["water tank"] != 0 || demo["billboard back"] != 0 ||
		demo["parapet"] == 0 {
		t.Errorf("a spire's roof held %d tanks, %d billboards and %d "+
			"parapets", demo["water tank"], demo["billboard back"],
			demo["parapet"])
	}
}

func TestTheParapetStandsOnTheRoofInsideTheFaces(t *testing.T) {
	pins := plainPins(skinConcrete)
	tower := towerOf(t, 3, 0, pins)
	drawn := drawnTower(t, tower, "demo", 0)
	// The faces a parapet is flush with are the roof's own plan, the one
	// it stands on: the demo faces stand a little proud of the hull's, so
	// the hull's box is not where they end.
	roof := named(drawn, "roof")
	if len(roof) == 0 {
		t.Fatal("no roof")
	}
	inner, outer := boundsOf(roof)
	foot := tower.height(0) + towerNumber(t, "roof-slab")
	for _, one := range named(drawn, "parapet") {
		low, high := boundsOf([]Rendered{one})
		if low.Y < foot-tolerance || high.Y > foot+
			towerNumber(t, "parapet-height")+tolerance {
			t.Errorf("a parapet stands from %v to %v, not on the roof at %v",
				low.Y, high.Y, foot)
		}
		if high.X > outer.X+tolerance || high.Z > outer.Z+tolerance ||
			low.X < inner.X-tolerance || low.Z < inner.Z-tolerance {
			t.Errorf("a parapet reaches %v to %v past the roof at %v to %v",
				low, high, inner, outer)
		}
	}
}

func TestOnlyAMirrorOrGraniteLobbyIsPolished(t *testing.T) {
	for _, skin := range []Skin{skinMirror, skinGranite, skinConcrete,
		skinCurtain} {
		pins := plainPins(skin)
		pins.Style = stylePunched
		drawn := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
		plinths := named(drawn, "plinth")
		if len(plinths) == 0 {
			t.Fatalf("%s: no plinth", skin)
		}
		polished := strings.Contains(plinths[0].Material.Name, "polished")
		if want := skin == skinMirror || skin == skinGranite; polished != want {
			t.Errorf("%s: the lobby's plinth is %q", skin,
				plinths[0].Material.Name)
		}
	}
}

func TestOnlyADarkShopHasItsShutterDown(t *testing.T) {
	pins := plainPins(skinConcrete)
	quickTower := towerOf(t, 3, 0, pins)
	quick := counts(drawnTower(t, quickTower, "quick", 0))
	demoTower := towerOf(t, 3, 0, pins)
	demo := counts(drawnTower(t, demoTower, "demo", 0))
	if quick["shutter"] != 0 {
		t.Errorf("%d shutters at quick", quick["shutter"])
	}
	if demo["shutter"] == 0 || demo["shutter"] >= demo["shop glass"] {
		t.Errorf("%d shutters over %d bays of shop glass", demo["shutter"],
			demo["shop glass"])
	}
}
