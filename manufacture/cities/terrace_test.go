package cities

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

// seeds are the seeds the row tests draw, enough to meet each choice a
// culture makes.
const seeds = 24

// drawnDesign is a design of a culture and a seed, drawn at the quick
// level, or the test's end.
func drawnDesign(t *testing.T, culture string, seed Seed) []Rendered {
	t.Helper()
	design, err := Terrace(culture, seed)
	if err != nil {
		t.Fatalf("%s from seed %d: %v", culture, seed, err)
	}
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// planOf is the row plan of a culture and a seed.
func planOf(t *testing.T, culture string, seed Seed) (*drawer, rowPlan) {
	t.Helper()
	drawing := drawerOf(t, culture, seed)
	plan := drawing.row()
	if drawing.trouble != nil {
		t.Fatal(drawing.trouble)
	}
	return drawing, plan
}

// digest is every corner, material and name of what a design drew, as a
// string: equal for two draws only if they are the same drawing.
func digest(drawn []Rendered) string {
	var builder strings.Builder
	for _, one := range drawn {
		builder.WriteString(one.Mesh.Name + one.Material.Name)
		for _, triangle := range one.Mesh.Triangles {
			for _, corner := range triangle {
				placed := one.Place.Out(corner)
				fmt.Fprintf(&builder, "%.5f,%.5f,%.5f;", placed.X, placed.Y,
					placed.Z)
			}
		}
	}
	return builder.String()
}

func TestTheSameSeedMakesTheSameRowAndAnotherSeedAnotherRow(t *testing.T) {
	for _, culture := range []string{"groningen", "portofino", "midwest"} {
		first := digest(drawnDesign(t, culture, 11))
		if first != digest(drawnDesign(t, culture, 11)) {
			t.Errorf("%s: one seed made two rows", culture)
		}
		if first == digest(drawnDesign(t, culture, 12)) {
			t.Errorf("%s: two seeds made the same row", culture)
		}
	}
}

func TestNoTwoAdjacentBaysShareAColourWhenTheCultureSaysSo(t *testing.T) {
	for _, culture := range []string{"groningen", "portofino"} {
		for seed := range Seed(seeds) {
			_, plan := planOf(t, culture, seed)
			for index := 1; index < len(plan.colours); index++ {
				if plan.colours[index].Name ==
					plan.colours[index-1].Name {
					t.Errorf("%s seed %d: bays %d and %d are both %s",
						culture, seed, index-1, index,
						plan.colours[index].Name)
				}
			}
		}
	}
}

func TestWithTheRuleOffNeighboursMayShareAColour(t *testing.T) {
	_, plan := planOf(t, "midwest", 3)
	for _, colour := range plan.colours {
		if colour.Name != plan.colours[0].Name {
			t.Fatalf("a district of one colour drew %v", plan.colours)
		}
	}
	if len(plan.colours) < 2 {
		t.Fatal("a row of one bay shows nothing")
	}
}

func TestEveryBayOfARowIsDrawnFromTheDistrictsPalette(t *testing.T) {
	for seed := range Seed(seeds) {
		_, plan := planOf(t, "groningen", seed)
		used := map[string]bool{}
		for _, colour := range plan.colours {
			used[colour.Name] = true
		}
		if len(used) < 3 {
			t.Errorf("seed %d: a row of %d bays used only %v", seed,
				plan.bays, used)
		}
	}
}

func TestAFlatRoofHasNoEavesAndBothFormsAreMet(t *testing.T) {
	seen := map[roofForm]bool{}
	for seed := range Seed(seeds) {
		drawing, plan := planOf(t, "portofino", seed)
		for bay := range plan.bays {
			shape := plan.shape(drawing, place{Bay: bay}, false)
			roof := drawing.roof(place{Bay: bay}, shape, nil)
			seen[roof.Form] = true
			if roof.Form == roofFlat && roof.Eave != 0 {
				t.Errorf("a flat roof with eaves of %v", roof.Eave)
			}
			if roof.Form == hip && !(roof.Eave > 0) {
				t.Errorf("a hip with no eaves")
			}
		}
	}
	if !seen[roofFlat] || !seen[hip] {
		t.Errorf("portofino drew only %v", seen)
	}
}

func TestATowerStandsOnlyAtAnEndOfTheRow(t *testing.T) {
	for seed := range Seed(seeds) {
		_, plan := planOf(t, "groningen", seed)
		if plan.tower != 0 && plan.tower != plan.bays-1 {
			t.Errorf("seed %d: a tower at bay %d of %d", seed, plan.tower,
				plan.bays)
		}
		_, other := planOf(t, "midwest", seed)
		if other.tower != noTower {
			t.Errorf("seed %d: a midwest tower at %d", seed, other.tower)
		}
	}
}

func TestTheTallestPartOfAGroningenRowIsAtAnEnd(t *testing.T) {
	for seed := range Seed(seeds) {
		_, plan := planOf(t, "groningen", seed)
		var tallest, across float64
		for _, one := range drawnDesign(t, "groningen", seed) {
			if low, high := reach(one); high.Y > tallest {
				tallest, across = high.Y, (low.X+high.X)/2
			}
		}
		if bay := math.Floor(across / plan.pitch); bay != 0 &&
			int(bay) != plan.bays-1 {
			t.Errorf("seed %d: the tallest part is in bay %v of %d", seed,
				bay, plan.bays)
		}
	}
}

func TestAChimneyStandsOnAPartyWallAndNeverAtTheRowsEnd(t *testing.T) {
	met := 0
	for _, culture := range []string{"groningen", "midwest"} {
		for seed := range Seed(seeds) {
			_, plan := planOf(t, culture, seed)
			for _, one := range drawnDesign(t, culture, seed) {
				if one.Mesh.Name != "chimney" {
					continue
				}
				met++
				low, high := reach(one)
				boundary := (low.X + high.X) / 2 / plan.pitch
				if math.Abs(boundary-math.Round(boundary)) > 1e-6 ||
					boundary < 1-1e-6 ||
					boundary > float64(plan.bays-1)+1e-6 {
					t.Errorf("%s seed %d: a chimney at %v pitches",
						culture, seed, boundary)
				}
			}
		}
	}
	if met == 0 {
		t.Error("no chimney was drawn in any row")
	}
}

func TestWindowsAreRecessedIntoTheWallNotStuckOnIt(t *testing.T) {
	for seed := range Seed(8) {
		drawing, plan := planOf(t, "portofino", seed)
		recess := drawing.fixed("recess")
		for _, one := range drawnDesign(t, "portofino", seed) {
			if one.Mesh.Name != "pane" {
				continue
			}
			_, high := reach(one)
			if high.Z >= plan.depth-recess/2 || high.Z < plan.depth-recess {
				t.Errorf("seed %d: a pane reaches z %v, the wall is at %v "+
					"and the cut is %v deep", seed, high.Z, plan.depth,
					recess)
			}
		}
	}
}

func TestWindowsAreDrawnWithinTheCulturesSizesAndLightRange(t *testing.T) {
	lit, dark := 0, 0
	for seed := range Seed(seeds) {
		for _, one := range drawnDesign(t, "portofino", seed) {
			if one.Mesh.Name != "pane" {
				continue
			}
			if name, found := strings.CutPrefix(one.Material.Name,
				"cities lit window "); found {
				lit++
				heat, err := strconv.Atoi(strings.TrimSuffix(name, "K"))
				if err != nil || heat < 2200 || heat > 6500 {
					t.Errorf("a lit window at %q", name)
				}
				continue
			}
			dark++
		}
	}
	if lit == 0 || dark == 0 || lit > dark {
		t.Errorf("%d lit and %d dark panes, want some of each and most dark",
			lit, dark)
	}
}

func TestPortofinoHasShuttersAndGroningenHasNone(t *testing.T) {
	count := func(culture string) int {
		shutters := 0
		for _, one := range drawnDesign(t, culture, 2) {
			if one.Mesh.Name == "shutter" {
				shutters++
			}
		}
		return shutters
	}
	if count("portofino") == 0 {
		t.Error("portofino has no shutters")
	}
	if count("groningen") != 0 {
		t.Error("groningen has shutters")
	}
}

func TestADetachedMidwestHouseHasAGarageBesideIt(t *testing.T) {
	garages := 0
	for _, one := range drawnDesign(t, "midwest", 4) {
		if one.Mesh.Name == "wall" {
			if low, high := reach(one); high.Y < 3 && high.X-low.X > 3 &&
				high.X-low.X < 3.3 {
				garages++
			}
		}
	}
	_, plan := planOf(t, "midwest", 4)
	if garages < plan.bays {
		t.Errorf("%d garage walls for %d houses", garages, plan.bays)
	}
}

func TestATerraceOfACultureThatIsNotOneIsRefused(t *testing.T) {
	for _, name := range []string{"atlantis", commonCulture,
		condominiumCulture} {
		if _, err := Terrace(name, 1); err == nil {
			t.Errorf("a terrace was drawn of %q", name)
		}
	}
}

func TestARowIsAsWideAsItsBaysAtTheirPitch(t *testing.T) {
	_, plan := planOf(t, "groningen", 1)
	var wide float64
	for _, one := range drawnDesign(t, "groningen", 1) {
		if _, high := reach(one); high.X > wide {
			wide = high.X
		}
	}
	whole := float64(plan.bays) * plan.pitch
	if wide < whole-plan.pitch || wide > whole+1 {
		t.Errorf("a row of %d bays at %v spans %v", plan.bays, plan.pitch,
			wide)
	}
}
