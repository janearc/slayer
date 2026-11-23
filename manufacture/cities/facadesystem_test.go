package cities

import (
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// systemMaterial is a material for a facade system's parts in a test.
func systemMaterial(name string) *thing.Material {
	return paint(name, option{Name: name, Colour: []float64{0.1, 0.1, 0.1}})
}

// systemFacade is a front drawn from a seed's households, built by the
// method of a kind.
func systemFacade(t *testing.T, seed Seed, kind string) facade {
	t.Helper()
	facade := homeFacade(t, seed)
	facade.System = system{Kind: kind, Joint: systemMaterial("joint"),
		Spandrel: systemMaterial("spandrel"),
		Mullion:  systemMaterial("mullion")}
	return facade
}

func TestAPrecastFaceHasSeamsThatNeverCrossAWindow(t *testing.T) {
	seams := 0
	for seed := range Seed(homeSeeds) {
		facade := systemFacade(t, seed, systemPrecast)
		for _, openings := range facade.Storeys {
			for _, part := range facade.system(openings,
				facade.storeyHeight()) {
				low, high := boxReach(t, part)
				seams++
				if low.X < -tolerance || high.X > facade.Width+tolerance {
					t.Errorf("seed %d: a seam from %v to %v is off the face",
						seed, low.X, high.X)
				}
				upright := high.Y-low.Y > facade.storeyHeight()-tolerance
				if upright && crosses(openings, low.X, high.X) {
					t.Errorf("seed %d: a seam over a window", seed)
				}
			}
		}
	}
	if seams == 0 {
		t.Error("no seam drawn")
	}
}

func TestACurtainWallHasABandUnderAndOverEachWindowAndTwoMullions(t *testing.T) {
	for seed := range Seed(homeSeeds) {
		facade := systemFacade(t, seed, systemCurtain)
		storey := facade.storeyHeight()
		for _, openings := range facade.Storeys[1:] {
			parts := facade.system(openings, storey)
			mullions, bands := 0, 0
			for _, part := range parts {
				low, high := boxReach(t, part)
				if high.Y > storey+tolerance || low.Y < -tolerance {
					t.Errorf("seed %d: a part from %v to %v is out of its "+
						"storey", seed, low.Y, high.Y)
				}
				if high.Y-low.Y > storey-tolerance {
					mullions++
				} else {
					bands++
				}
			}
			if mullions != 2*len(openings) || bands < len(openings) {
				t.Errorf("seed %d: %d mullions and %d bands for %d windows",
					seed, mullions, bands, len(openings))
			}
		}
	}
}

func TestAStuccoFaceShowsNoMethodOfBuilding(t *testing.T) {
	facade := systemFacade(t, 1, systemRender)
	if parts := facade.system(facade.Storeys[1], facade.storeyHeight()); len(
		parts) != 0 {
		t.Errorf("%d parts of a method a stucco face does not show", len(parts))
	}
}

func TestEveryBuildingIsBuiltByOneMethodAndBrickByNone(t *testing.T) {
	kinds := map[string]bool{}
	district := Seed(drawerOf(t, condominiumCulture, 1).district)
	for index := range Seed(seeds) {
		drawing := homeDrawer(t, index*district)
		for bay := range 4 {
			at := place{Bay: bay}
			plain := wall(option{Name: "cream", Colour: []float64{1, 1, 1}},
				option{Name: renderCladding, Colour: []float64{1, 1, 1}})
			brick := wall(option{Name: "cream", Colour: []float64{1, 1, 1}},
				option{Name: "brick-slips", Colour: []float64{1, 1, 1}})
			kinds[drawing.system(at, plain).Kind] = true
			if got := drawing.system(at, brick).Kind; got != systemRender {
				t.Errorf("a brick building built by %s", got)
			}
		}
	}
	for _, kind := range []string{systemRender, systemPrecast, systemCurtain} {
		if !kinds[kind] {
			t.Errorf("no building built by %s in %d seeds", kind, seeds)
		}
	}
}

func TestEveryStairCoreHasAPortalAndAGuardAtEachLandingWindow(t *testing.T) {
	district := Seed(drawerOf(t, condominiumCulture, 1).district)
	for index := range Seed(homeSeeds) {
		seed := index * district
		design, err := CondominiumAt(seed, nil, demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := design.Rendered(demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		cores := mustCount(t, seed)
		for _, name := range []string{"portal lintel", "entry panel"} {
			if got := len(named(drawn, name)); got != cores {
				t.Errorf("seed %d: %d %s for %d stair cores", seed, got, name,
					cores)
			}
		}
		if got := len(named(drawn, "portal jamb")); got != 2*cores {
			t.Errorf("seed %d: %d jambs for %d stair cores", seed, got, cores)
		}
	}
}

func TestSomeCondominiumsShowSeamsAndSomeShowCurtainWalls(t *testing.T) {
	district := Seed(drawerOf(t, condominiumCulture, 1).district)
	shown := map[string][]Seed{}
	for index := range Seed(seeds) {
		seed := index * district
		design, err := CondominiumAt(seed, nil, demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := design.Rendered(demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"joint", "mullion"} {
			if len(named(drawn, name)) != 0 {
				shown[name] = append(shown[name], seed)
			}
		}
	}
	t.Logf("seams in %v, curtain walls in %v", shown["joint"], shown["mullion"])
	if len(shown["joint"]) == 0 || len(shown["mullion"]) == 0 {
		t.Errorf("seams in %d seeds, curtain walls in %d, of %d",
			len(shown["joint"]), len(shown["mullion"]), seeds)
	}
}

// mustCount is how many buildings a seed's condominium has.
func mustCount(t *testing.T, seed Seed) int {
	t.Helper()
	return drawerOf(t, condominiumCulture, seed).count("buildings", place{})
}
