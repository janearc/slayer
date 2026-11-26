package seasonal

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// mangroveKinds are the profiles of the tidal shore, in the profile data.
var mangroveKinds = []string{"kandelia-obovata", "aegiceras-corniculatum",
	"avicennia-marina", "bruguiera-gymnorhiza", "sonneratia-apetala"}

// mangroveSeeds is how many different seeds a test grows of each kind.
const mangroveSeeds = 12

// rootsSignature is every root's two ends, in the plan's own order.
func rootsSignature(roots []plantAxis) string {
	var signature string
	for _, root := range roots {
		signature += fmt.Sprintf("%.6f,%.6f,%.6f>%.6f,%.6f,%.6f;",
			root.root.X, root.root.Y, root.root.Z,
			root.end.X, root.end.Y, root.end.Z)
	}
	return signature
}

// mangroveRoots is the profile's roots as a function of one profile's
// copy, so a change made by a test never reaches the shared book.
func mangroveRoots(t *testing.T, change func(*Roots)) Profile {
	t.Helper()
	profile := plantProfiles["avicennia-marina"]
	if profile.Roots == nil {
		t.Fatal("avicennia-marina has no roots")
	}
	copied := *profile.Roots
	change(&copied)
	profile.Roots = &copied
	return profile
}

func TestMangroveProfilesCarryRoots(t *testing.T) {
	for _, name := range mangroveKinds {
		if plantProfiles[name].Roots == nil {
			t.Fatalf("%s shows no roots", name)
		}
	}
}

func TestMangroveRootsDifferBySeed(t *testing.T) {
	for _, name := range mangroveKinds {
		profile := plantProfiles[name]
		seen := map[string]bool{}
		for number := range uint64(mangroveSeeds) {
			plan, err := profile.Plan(sow(number))
			if err != nil {
				t.Fatal(err)
			}
			seen[rootsSignature(plan.roots)] = true
		}
		if len(seen) != mangroveSeeds {
			t.Fatalf("%s: %d seeds grew %d different root systems", name,
				mangroveSeeds, len(seen))
		}
	}
}

func TestMangroveRootsAreTheSameForTheSameSeed(t *testing.T) {
	for _, name := range mangroveKinds {
		profile := plantProfiles[name]
		first, err := profile.Plan(sow(5))
		if err != nil {
			t.Fatal(err)
		}
		second, err := profile.Plan(sow(5))
		if err != nil {
			t.Fatal(err)
		}
		if rootsSignature(first.roots) != rootsSignature(second.roots) {
			t.Fatalf("%s: one seed grew two root systems", name)
		}
	}
}

func TestMangroveRootsStandInTheGroundAndRiseOutOfIt(t *testing.T) {
	for _, name := range mangroveKinds {
		profile := plantProfiles[name]
		for number := range uint64(mangroveSeeds) {
			plan, err := profile.Plan(sow(number))
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.roots) == 0 {
				t.Fatalf("%s seed %d grew no roots", name, number)
			}
			rooted, risen := 0, 0
			for _, root := range plan.roots {
				if root.root.Y < 0 || root.end.Y < 0 {
					rooted++
				}
				if root.root.Y > 0 || root.end.Y > 0 {
					risen++
				}
			}
			if rooted == 0 || risen != len(plan.roots) {
				t.Fatalf("%s seed %d: %d roots in the ground, %d above, of %d",
					name, number, rooted, risen, len(plan.roots))
			}
		}
	}
}

func TestMangroveRootCountsStayInTheirRange(t *testing.T) {
	for _, name := range mangroveKinds {
		profile := plantProfiles[name]
		roots := profile.Roots
		per := 1
		if roots.Habit == rootProp {
			per = 2
		}
		for number := range uint64(mangroveSeeds) {
			plan, err := profile.Plan(sow(number))
			if err != nil {
				t.Fatal(err)
			}
			count := len(plan.roots) / per
			if len(plan.roots)%per != 0 || count < roots.Count.Minimum ||
				count > roots.Count.Maximum {
				t.Fatalf("%s seed %d: %d roots outside %d to %d", name, number,
					count, roots.Count.Minimum, roots.Count.Maximum)
			}
		}
	}
}

func TestMangroveBoundsEncloseEveryRoot(t *testing.T) {
	for _, name := range mangroveKinds {
		plan, err := plantProfiles[name].Plan(sow(3))
		if err != nil {
			t.Fatal(err)
		}
		box := plan.Bounds()
		for _, root := range plan.roots {
			for _, end := range []float64{root.root.X, root.end.X} {
				if end < box.Min.X || end > box.Max.X {
					t.Fatalf("%s: a root's end at x %v leaves its bounds", name, end)
				}
			}
			for _, end := range []float64{root.root.Z, root.end.Z} {
				if end < box.Min.Z || end > box.Max.Z {
					t.Fatalf("%s: a root's end at z %v leaves its bounds", name, end)
				}
			}
		}
	}
}

// mangroveManySeeds is how many seeds a budget test grows of each kind,
// enough to meet the seeds that put a kind nearest its cell budget.
const mangroveManySeeds = 2000

func TestMangroveFoliageStaysInItsCellBudgetForManySeeds(t *testing.T) {
	budget := int(plantValue("maximum-cells"))
	for _, name := range mangroveKinds {
		profile := plantProfiles[name]
		most := 0
		for number := range uint64(mangroveManySeeds) {
			plan, err := profile.Plan(sow(number))
			if err != nil {
				t.Fatalf("%s seed %d: %v", name, number, err)
			}
			cells, err := profile.Attachments(plan, "clumps")
			if err != nil {
				t.Fatalf("%s seed %d: %v", name, number, err)
			}
			most = max(most, len(cells))
		}
		if most == 0 || most > budget {
			t.Fatalf("%s: at most %d cells of %d", name, most, budget)
		}
	}
}

func TestPlantsWithoutARootsBlockGrowNoRoots(t *testing.T) {
	for _, name := range []string{"valley-oak", "alder", "poplar", "manzanita",
		"chaparral-brush", "joshua-tree"} {
		plan, err := plantProfiles[name].Plan(sow(5))
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.roots) != 0 {
			t.Fatalf("%s grew %d roots without asking", name, len(plan.roots))
		}
	}
}

func TestPencilRootsStandApartFromTheTrunk(t *testing.T) {
	clearance := plantValue("root-clearance")
	profile := plantProfiles["avicennia-marina"]
	for number := range uint64(mangroveSeeds) {
		plan, err := profile.Plan(sow(number))
		if err != nil {
			t.Fatal(err)
		}
		for _, root := range plan.roots {
			if away := math.Hypot(root.root.X, root.root.Z); away < clearance {
				t.Fatalf("a pencil root stands %v m from the trunk, under %v",
					away, clearance)
			}
		}
	}
}

func TestMangroveRootsValidateTheirOwnBlock(t *testing.T) {
	if err := plantProfiles["avicennia-marina"].validRoots(); err != nil {
		t.Fatal(err)
	}
	if err := plantProfiles["valley-oak"].validRoots(); err != nil {
		t.Fatalf("a profile without roots is refused: %v", err)
	}
}

func TestMangroveRootsRefuseWhatCannotBeRoots(t *testing.T) {
	for name, change := range map[string]func(*Roots){
		"an unknown habit":       func(roots *Roots) { roots.Habit = "stilt" },
		"no habit":               func(roots *Roots) { roots.Habit = "" },
		"no roots":               func(roots *Roots) { roots.Count.Minimum = 0 },
		"fewer than the least":   func(roots *Roots) { roots.Count.Maximum = roots.Count.Minimum - 1 },
		"more than the budget":   func(roots *Roots) { roots.Count.Maximum = int(plantValue("maximum-roots")) + 1 },
		"no height":              func(roots *Roots) { roots.Height.Minimum = 0 },
		"a negative height":      func(roots *Roots) { roots.Height.Minimum = -0.1 },
		"a most under the least": func(roots *Roots) { roots.Height.Maximum = roots.Height.Minimum / 2 },
		"no radius":              func(roots *Roots) { roots.Radius = 0 },
		"a tip as wide as foot":  func(roots *Roots) { roots.Tip = 1 },
		"a tip wider than foot":  func(roots *Roots) { roots.Tip = 2 },
		"a negative tip":         func(roots *Roots) { roots.Tip = -0.1 },
		"no spread":              func(roots *Roots) { roots.Spread = 0 },
		"a field inside the trunk's clearance": func(roots *Roots) {
			roots.Spread = plantValue("root-clearance") / 100
		},
		"a radius that is not a number": func(roots *Roots) { roots.Radius = math.NaN() },
		"a spread without end":          func(roots *Roots) { roots.Spread = math.Inf(1) },
	} {
		if err := mangroveRoots(t, change).validRoots(); err == nil {
			t.Fatalf("%s was accepted as a root system", name)
		}
	}
}

func TestPropRootsMayReachNearerTheTrunkThanPencilRootsMay(t *testing.T) {
	// a field inside the clearance is nonsense for upright roots, which
	// must stand clear of the trunk, but a prop starts at the trunk
	profile := mangroveRoots(t, func(roots *Roots) {
		roots.Habit = rootProp
		roots.Spread = plantValue("root-clearance") / 100
	})
	if err := profile.validRoots(); err != nil {
		t.Fatalf("a short prop root was refused: %v", err)
	}
}

func TestMangroveProfilesAreNamedAsRecalledOrPicked(t *testing.T) {
	for _, name := range mangroveKinds {
		profile := plantProfiles[name]
		if profile.Status != "picked" ||
			!strings.Contains(profile.By, "recalled") ||
			!strings.Contains(profile.By, "not looked up") {
			t.Fatalf("%s does not say what was recalled: %q", name, profile.By)
		}
	}
}
