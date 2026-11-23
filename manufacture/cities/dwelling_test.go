package cities

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/maths"
)

// homeSeeds are how many seeds the dwelling tests draw.
const homeSeeds = 6

// homeFront is the width of the front the dwelling tests draw, and the
// depth and storeys of the building behind it.
const (
	homeFront   = 30.0
	homeDepth   = 11.5
	homeStoreys = 5
	homeFrom    = 3.0
)

// homeDrawer is a drawer of the condominium at the demo level, with its
// windows belonging to households.
func homeDrawer(t *testing.T, seed Seed) *drawer {
	t.Helper()
	drawing := drawerOf(t, condominiumCulture, seed)
	drawing.level, drawing.homes = demoLevel, true
	return drawing
}

// homeFacade is a front drawn from a seed's households.
func homeFacade(t *testing.T, seed Seed) facade {
	t.Helper()
	drawing := homeDrawer(t, seed)
	cover := option{Name: "render", Colour: []float64{1, 1, 1}}
	tone := option{Name: "tone", Colour: []float64{0.5, 0.5, 0.5}}
	facade := drawing.facade(place{}, massShape{width: homeFront,
		depth: homeDepth, storeys: homeStoreys, from: homeFrom},
		wall(tone, cover))
	if drawing.trouble != nil {
		t.Fatal(drawing.trouble)
	}
	return facade
}

// signature is what a household chose, as one string.
func signature(home *household) string {
	return fmt.Sprint(home.Blind != nil, home.Drop, home.Curtain != nil,
		home.Cover, home.Balcony, len(home.Pieces), home.Unit != nil,
		home.Replaced != nil)
}

// boxReach is the corners of a placed box, in the frame it is placed in.
func boxReach(t *testing.T, part placedElement) (low, high maths.Vec) {
	t.Helper()
	solid, isSolid := part.Element.(solid)
	box, isBox := solid.Form.(Box)
	if !isSolid || !isBox {
		t.Fatalf("a placed part that is not a box: %T", part.Element)
	}
	low = part.Place.Origin
	return low, low.Add(maths.V(box.Width, box.Height, box.Depth))
}

func TestTwoDwellingsDrawnFromDifferentSeedsDiffer(t *testing.T) {
	seen := map[string]bool{}
	for seed := range Seed(homeSeeds) {
		drawer := drawerOf(t, condominiumCulture,
			1)
		drawing := homeDrawer(t, seed*Seed(drawer.district))
		for dwelling := range 20 {
			at := place{Storey: 1 + dwelling%homeStoreys, Dwelling: dwelling}
			seen[signature(drawing.household(at))] = true
		}
	}
	if len(seen) < 8 {
		t.Errorf("%d kinds of household in %d dwellings, want a spread",
			len(seen), homeSeeds*20)
	}
	if drawingsAgree(t) {
		t.Error("dwellings of two seeds are all alike")
	}
}

// drawingsAgree is whether twenty dwellings of two seeds are the same.
func drawingsAgree(t *testing.T) bool {
	t.Helper()
	first, second := homeDrawer(t, 1), homeDrawer(t, 77)
	for dwelling := range 20 {
		at := place{Storey: 2, Dwelling: dwelling}
		if signature(first.household(at)) != signature(second.household(at)) {
			return false
		}
	}
	return true
}

func TestTheWindowsOfOneDwellingShareTheirLights(t *testing.T) {
	for seed := range Seed(homeSeeds) {
		facade := homeFacade(t, seed)
		for _, openings := range facade.Storeys {
			lit := map[*household]bool{}
			for _, opening := range openings {
				if opening.Home == nil {
					continue
				}
				on := opening.Lamp != nil
				if known, seen := lit[opening.Home]; seen && known != on {
					t.Errorf("seed %d: one dwelling, lit and dark", seed)
				}
				lit[opening.Home] = on
			}
		}
	}
}

func TestEveryThingOnABalconyIsInsideItsDwelling(t *testing.T) {
	balconies := 0
	for seed := range Seed(homeSeeds) {
		facade := homeFacade(t, seed)
		side := facade.measured("home-balcony-side")
		deep := facade.measured("home-balcony-depth")
		slab := facade.measured("balcony-slab")
		for _, openings := range facade.Storeys {
			for _, run := range flats(openings) {
				for _, part := range facade.balconyOf(run) {
					low, high := boxReach(t, part)
					balconies++
					if low.X < run.left-side-tolerance ||
						high.X > run.right+side+tolerance ||
						low.Z < -tolerance || high.Z > deep+tolerance ||
						low.Y < -slab-tolerance ||
						high.Y > facade.storeyHeight()+tolerance {
						t.Errorf("seed %d: a part at %v to %v is out of "+
							"its dwelling %v to %v", seed, low, high,
							run.left, run.right)
					}
				}
			}
		}
	}
	if balconies == 0 {
		t.Error("no balcony drawn in any seed")
	}
}

func TestADwellingWithNoBalconyDrawsNone(t *testing.T) {
	run := flat{home: &household{}, left: 1, right: 4}
	if parts := homeFacade(t, 1).balconyOf(run); len(parts) != 0 {
		t.Errorf("%d parts of a balcony that is not there", len(parts))
	}
}

func TestTheSameSeedDrawsTheSameCondominiumAtTheDemoLevel(t *testing.T) {
	draw := func() []Rendered {
		design, err := CondominiumAt(4, nil, demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := design.Rendered(demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		return drawn
	}
	if first, second := digest(draw()), digest(draw()); first != second {
		t.Error("two draws of one seed differ")
	}
}

func TestAClutteredBalconyHoldsOnlyWhatItsHouseholdKeeps(t *testing.T) {
	for seed := range Seed(homeSeeds) {
		facade := homeFacade(t, seed)
		for _, openings := range facade.Storeys {
			for _, run := range flats(openings) {
				if run.home.Balcony != balconyOpen &&
					len(run.home.Pieces) != 0 {
					t.Errorf("seed %d: clutter on a balcony that is %q",
						seed, run.home.Balcony)
				}
			}
		}
	}
}
