package cities

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// fastenerParts are the parts fasteners are, by the names they are drawn
// under.
var fastenerParts = []string{"post bolt", "anchor plate", "anchor bolt"}

// fastenedSkins are the skins that have posts and bands to fasten to,
// and the skins that have posts only.
var (
	fastenedSkins = []Skin{skinCurtain, skinMirror, skinConcrete,
		skinVaporwave}
	postsOnly = []Skin{skinGranite}
	unfixed   = []Skin{skinGothic, skinCantilever, skinStainless}
)

// coarseFrom is a leveller that draws a thing at the demo level only if
// it reaches across at least so many metres, and at the quick level if
// it does not.
func coarseFrom(least float64) Leveller {
	return func(_ maths.Subspace, extent float64) Detail {
		if extent >= least {
			return demoLevel
		}
		return quickLevel
	}
}

// levelledTower is a tower of a skin drawn by a leveller, in a design
// made for the demo level.
func levelledTower(t *testing.T, skin Skin, leveller Leveller) []Rendered {
	t.Helper()
	design, err := towerOf(t, 3, 0, plainPins(skin)).Design(demoLevel, 0)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Levelled(Pass{Leveller: leveller,
		Finest: demoLevel}, standingAt(maths.Vec{}))
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestADemoTowerOfASkinWithPostsAndBandsHasBoltsDownItsPostsAndPlates(
	t *testing.T) {
	for _, skin := range fastenedSkins {
		quickTower := towerOf(t, 3, 0, plainPins(skin))
		quick := counts(drawnTower(t, quickTower,
			"quick", 0))
		demoTower := towerOf(t, 3, 0, plainPins(skin))
		demo := counts(drawnTower(t, demoTower,
			"demo", 0))
		for _, name := range fastenerParts {
			if quick[name] != 0 {
				t.Errorf("%s: %d %q at the quick level", skin, quick[name],
					name)
			}
			if demo[name] == 0 {
				t.Errorf("%s: no %q at the demo level", skin, name)
			}
		}
	}
}

func TestAGraniteWallHasBoltsDownItsPiersAndNoAnchorPlates(t *testing.T) {
	for _, skin := range postsOnly {
		// Piers are what a punched wall has; a ribbon wall has bands of
		// stone and glass and no piers to bolt (fixingOf), and the seed
		// would otherwise draw either.
		pins := plainPins(skin)
		pins.Style = stylePunched
		demoTower := towerOf(t, 3, 0, pins)
		demo := counts(drawnTower(t, demoTower, "demo", 0))
		if demo["post bolt"] == 0 {
			t.Errorf("%s: no post bolts at the demo level", skin)
		}
		if demo["anchor plate"] != 0 || demo["anchor bolt"] != 0 {
			t.Errorf("%s: %d anchor plates and %d anchor bolts, on a wall "+
				"whose bands are not known", skin, demo["anchor plate"],
				demo["anchor bolt"])
		}
	}
}

func TestASkinWithNoKnownPostsOrBandsHasNoFasteners(t *testing.T) {
	for _, skin := range unfixed {
		demoTower := towerOf(t, 3, 0, plainPins(skin))
		demo := counts(drawnTower(t, demoTower,
			"demo", 0))
		for _, name := range fastenerParts {
			if demo[name] != 0 {
				t.Errorf("%s: %d %q", skin, demo[name], name)
			}
		}
	}
}

func TestEveryAnchorPlateHasFourBoltsAndEveryPostTwoPerStoreyAtLeast(
	t *testing.T) {
	for _, skin := range fastenedSkins {
		demoTower := towerOf(t, 3, 0, plainPins(skin))
		demo := counts(drawnTower(t, demoTower,
			"demo", 0))
		if demo["anchor bolt"] != 4*demo["anchor plate"] {
			t.Errorf("%s: %d anchor bolts on %d plates", skin,
				demo["anchor bolt"], demo["anchor plate"])
		}
		if demo["post bolt"]%2 != 0 {
			t.Errorf("%s: %d post bolts, which go two to a storey", skin,
				demo["post bolt"])
		}
	}
}

func TestFastenersAreSteelSoTheWorldRustsThem(t *testing.T) {
	tower := towerOf(t, 3, 0, plainPins(skinCurtain))
	drawn := drawnTower(t, tower, "demo", 0)
	for _, name := range fastenerParts {
		for _, one := range named(drawn, name) {
			if !strings.Contains(one.Material.Name, "steel") {
				t.Errorf("a %s is %q", name, one.Material.Name)
			}
		}
	}
}

func TestNoFastenerStandsOnTheGroundFloorOrBelowIt(t *testing.T) {
	tower := towerOf(t, 3, 0, plainPins(skinCurtain))
	drawn := drawnTower(t, tower, "demo", 0)
	ground := tower.baseHeight() + tower.storeyHeight()
	for _, name := range fastenerParts {
		for _, one := range named(drawn, name) {
			low, _ := boundsOf([]Rendered{one})
			if low.Y < ground-tolerance {
				t.Errorf("a %s at %v, under %v, the top of the ground floor",
					name, low.Y, ground)
			}
		}
	}
}

func TestFastenersAreLeftToTheTilesTheLevellerGivesTheDemoLevel(
	t *testing.T) {
	const bandsOnly, everyTile = 20.0, 2.0
	farLevel := coarseFrom(bandsOnly)
	far := counts(levelledTower(t, skinCurtain, farLevel))
	nearLevel := coarseFrom(everyTile)
	near := counts(levelledTower(t, skinCurtain, nearLevel))
	for _, name := range fastenerParts {
		if far[name] != 0 {
			t.Errorf("%d %q in tiles the leveller left at quick", far[name],
				name)
		}
		if near[name] == 0 {
			t.Errorf("no %q in tiles the leveller drew at demo", name)
		}
	}
	if far["transom"] == 0 {
		t.Error("a band at the demo level has no part-by-part wall drawn")
	}
}

func TestADemoDesignDrawnAtQuickHasAWallAtTheRecessAndAQuickDesignHasNone(
	t *testing.T) {
	everything := coarseFrom(1e9)
	quick, err := towerOf(t, 3, 0, plainPins(skinCurtain)).Design(quickLevel,
		0)
	if err != nil {
		t.Fatal(err)
	}
	made, err := quick.Levelled(Pass{Leveller: everything,
		Finest: quickLevel}, standingAt(maths.Vec{}))
	if err != nil {
		t.Fatal(err)
	}
	inDemo := levelledTower(t, skinCurtain, everything)
	demoTower := towerOf(t, 3, 0, plainPins(skinCurtain))
	demo := drawnTower(t, demoTower, "demo", 0)
	walls := func(drawn []Rendered) int {
		found := counts(drawn)
		return found["glass"] + found["body"]
	}
	if walls(inDemo) <= walls(demo) {
		t.Errorf("%d walls at the quick level in a demo design, and %d at "+
			"the demo level: no wall at the recess behind the glass",
			walls(inDemo), walls(demo))
	}
	if walls(made) >= walls(inDemo) {
		t.Errorf("%d walls in a quick design, %d in a demo design drawn "+
			"at quick: a quick design needs no more", walls(made),
			walls(inDemo))
	}
}
